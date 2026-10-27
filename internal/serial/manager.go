package serial

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/skinnyrad/Lora-Scanner/internal/beacon"
	"github.com/skinnyrad/Lora-Scanner/internal/frames"
	"github.com/skinnyrad/Lora-Scanner/internal/hub"
	gs "go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

const BaudRate = 9600

var (
	ErrNotFound  = errors.New("transceiver not found")
	ErrPortInUse = errors.New("port already in use")
)

// Opener opens a serial device. It is a field so tests can inject a fake.
type Opener func(name string) (io.ReadWriteCloser, error)

func defaultOpener(name string) (io.ReadWriteCloser, error) {
	return gs.Open(name, &gs.Mode{BaudRate: BaudRate})
}

// Transceiver is an attached Feather. Freq (MHz) is nil until detected from
// the firmware's boot banner.
type Transceiver struct {
	ID   int      `json:"id"`
	Port string   `json:"port"`
	Freq *float64 `json:"freq"`
}

// ListMsg is broadcast on the hub whenever the set of transceivers (or a
// detected frequency) changes.
type ListMsg struct {
	Type         string        `json:"type"` // "transceivers"
	Transceivers []Transceiver `json:"transceivers"`
}

type slot struct {
	t       Transceiver
	conn    io.ReadWriteCloser
	done    chan struct{}
	beacon  beacon.Beacon
	writeMu sync.Mutex
}

type Manager struct {
	mu       sync.Mutex
	slots    map[int]*slot
	used     map[string]int     // port -> id
	lastFreq map[string]float64 // port -> last detected frequency
	nextID   int
	hub      *hub.Hub
	frames   *frames.Store
	open     Opener
}

func NewManager(h *hub.Hub, f *frames.Store) *Manager {
	return &Manager{
		slots: map[int]*slot{}, used: map[string]int{}, lastFreq: map[string]float64{},
		nextID: 1, hub: h, frames: f, open: defaultOpener,
	}
}

// ListPorts returns the serial device paths available on this machine.
func ListPorts() ([]string, error) {
	details, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}
	ports := make([]string, 0, len(details))
	for _, d := range details {
		ports = append(ports, d.Name)
	}
	sort.Strings(ports)
	return ports, nil
}

// Add opens the port and starts reading from it.
func (m *Manager) Add(port string) (Transceiver, error) {
	port = strings.TrimSpace(port)
	if port == "" {
		return Transceiver{}, errors.New("port is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, busy := m.used[port]; busy {
		return Transceiver{}, ErrPortInUse
	}
	conn, err := m.open(port)
	if err != nil {
		return Transceiver{}, fmt.Errorf("open %s: %w", port, err)
	}
	s := &slot{conn: conn, done: make(chan struct{}), t: Transceiver{ID: m.nextID, Port: port}}
	m.nextID++
	// The firmware only prints its frequency at boot, so a reconnect to a
	// board that is already running remembers what we saw last time.
	if f, ok := m.lastFreq[port]; ok {
		s.t.Freq = &f
	}
	m.slots[s.t.ID] = s
	m.used[port] = s.t.ID
	go m.read(s)
	m.broadcastLocked()
	return s.t, nil
}

func (m *Manager) Remove(id int) error {
	m.mu.Lock()
	s, ok := m.slots[id]
	if !ok {
		m.mu.Unlock()
		return ErrNotFound
	}
	m.removeLocked(s)
	m.mu.Unlock()

	s.beacon.Stop()
	s.conn.Close() // unblocks the reader
	<-s.done
	return nil
}

// removeLocked drops the slot from the registry and tells clients.
func (m *Manager) removeLocked(s *slot) {
	delete(m.slots, s.t.ID)
	delete(m.used, s.t.Port)
	m.broadcastLocked()
}

func (m *Manager) listLocked() []Transceiver {
	out := make([]Transceiver, 0, len(m.slots))
	for _, s := range m.slots {
		out = append(out, s.t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (m *Manager) broadcastLocked() {
	m.hub.Broadcast(ListMsg{Type: "transceivers", Transceivers: m.listLocked()})
}

func (m *Manager) List() []Transceiver {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.listLocked()
}

func (m *Manager) get(id int) (*slot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.slots[id]
	if !ok {
		return nil, ErrNotFound
	}
	return s, nil
}

// Transmit writes the legacy "TX:<text> " frame.
func (m *Manager) Transmit(id int, text string) error {
	s, err := m.get(id)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err = s.conn.Write([]byte("TX:" + text + " "))
	return err
}

func (m *Manager) StartBeacon(id int, text string, interval time.Duration) error {
	s, err := m.get(id)
	if err != nil {
		return err
	}
	s.beacon.Start(text, interval, func(t string) error { return m.Transmit(id, t) })
	return nil
}

func (m *Manager) StopBeacon(id int) error {
	s, err := m.get(id)
	if err != nil {
		return err
	}
	s.beacon.Stop()
	return nil
}

func (m *Manager) BeaconState(id int) (beacon.State, error) {
	s, err := m.get(id)
	if err != nil {
		return beacon.State{}, err
	}
	return s.beacon.State(), nil
}

// setFreq records a detected frequency and returns the slot's current label info.
func (m *Manager) setFreq(s *slot, f float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastFreq[s.t.Port] = f
	if _, live := m.slots[s.t.ID]; !live {
		return
	}
	s.t.Freq = &f
	m.broadcastLocked()
}

func (m *Manager) snapshot(s *slot) Transceiver {
	m.mu.Lock()
	defer m.mu.Unlock()
	return s.t
}

func sourceLabel(t Transceiver) string {
	name := filepath.Base(t.Port)
	if t.Freq != nil {
		return fmt.Sprintf("%s (%s MHz)", name, fmtFreq(*t.Freq))
	}
	return name
}

func fmtFreq(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", f), "0"), ".")
}

func (m *Manager) read(s *slot) {
	defer close(s.done)
	var p Parser
	sc := bufio.NewScanner(s.conn)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if f, ok := ParseFreq(line); ok {
			m.setFreq(s, f)
		}
		t := m.snapshot(s)
		source := sourceLabel(t)
		m.frames.Add(frames.Frame{Kind: frames.KindRaw, Source: source, Transceiver: t.ID, Info: line, Raw: line})

		if rssi, decoded, ok := p.Feed(line); ok {
			id := "Raw LoRa Device (" + filepath.Base(t.Port) + ")"
			var freq float64
			if t.Freq != nil {
				freq = *t.Freq
				id = fmt.Sprintf("Raw LoRa Device %s MHz", fmtFreq(freq))
			}
			m.frames.Add(frames.Frame{
				Kind: frames.KindLoRa, Source: source, Transceiver: t.ID, Identifier: id, Freq: freq, RSSI: &rssi,
				Info: decoded, Raw: fmt.Sprintf("RSSI: %d\nDecoded Value: %s", rssi, decoded),
			})
		}
	}
	// The device vanished (unplugged) rather than an explicit Remove: release it.
	m.mu.Lock()
	if m.slots[s.t.ID] == s {
		m.removeLocked(s)
		m.mu.Unlock()
		s.beacon.Stop()
		s.conn.Close()
		return
	}
	m.mu.Unlock()
}
