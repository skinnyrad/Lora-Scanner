// Package frames holds received frames and the devices aggregated from them.
package frames

import (
	"sort"
	"sync"
	"time"

	"github.com/skinnyrad/Lora-Scanner/internal/hub"
)

// DefaultMax is how many frames the ring keeps.
const DefaultMax = 10000

const (
	KindLoRa    = "lora"    // packet decoded by a Feather
	KindLoRaWAN = "lorawan" // row scraped from a gateway
	KindRaw     = "raw"     // any serial line
)

type Frame struct {
	ID     uint64    `json:"id"`
	Time   time.Time `json:"time"`
	Kind   string    `json:"kind"`
	Source string    `json:"source"`
	// Transceiver is the ID of the attached Feather that produced the frame (0 for gateway frames).
	Transceiver int     `json:"transceiver"`
	Identifier  string  `json:"identifier"`
	Freq        float64 `json:"freq"`
	RSSI        *int    `json:"rssi"`
	Info        string  `json:"info"`
	Raw         string  `json:"raw"`
}

type Device struct {
	Identifier string    `json:"identifier"`
	Freq       float64   `json:"freq"`
	RSSI       *int      `json:"rssi"`
	FirstSeen  time.Time `json:"firstSeen"`
	LastSeen   time.Time `json:"lastSeen"`
	Frames     int       `json:"frames"`
}

// Messages broadcast on the hub.
type FrameMsg struct {
	Type  string `json:"type"` // "frame"
	Frame Frame  `json:"frame"`
}

type DeviceMsg struct {
	Type   string `json:"type"` // "device"
	Device Device `json:"device"`
}

type Store struct {
	mu      sync.RWMutex
	hub     *hub.Hub
	max     int
	nextID  uint64
	frames  []Frame
	devices map[string]*Device
}

// NewStore keeps at most max frames (DefaultMax if <= 0). hub may be nil.
func NewStore(h *hub.Hub, max int) *Store {
	if max <= 0 {
		max = DefaultMax
	}
	return &Store{hub: h, max: max, nextID: 1, devices: map[string]*Device{}}
}

// Add assigns the ID, stores the frame, updates its device (non-raw only)
// and publishes both to the hub. A zero Time is set to now.
func (s *Store) Add(f Frame) Frame {
	if f.Time.IsZero() {
		f.Time = time.Now()
	}
	s.mu.Lock()
	f.ID = s.nextID
	s.nextID++
	s.frames = append(s.frames, f)
	if len(s.frames) > s.max {
		s.frames = append([]Frame(nil), s.frames[len(s.frames)-s.max:]...)
	}
	var dev *Device
	if f.Kind != KindRaw && f.Identifier != "" {
		d, ok := s.devices[f.Identifier]
		if !ok {
			d = &Device{Identifier: f.Identifier, FirstSeen: f.Time}
			s.devices[f.Identifier] = d
		}
		d.Freq, d.RSSI, d.LastSeen = f.Freq, f.RSSI, f.Time
		d.Frames++
		cp := *d
		dev = &cp
	}
	s.mu.Unlock()

	if s.hub != nil {
		s.hub.Broadcast(FrameMsg{Type: "frame", Frame: f})
		if dev != nil {
			s.hub.Broadcast(DeviceMsg{Type: "device", Device: *dev})
		}
	}
	return f
}

// Since returns frames with ID > after, oldest first.
func (s *Store) Since(after uint64) []Frame {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i := sort.Search(len(s.frames), func(i int) bool { return s.frames[i].ID > after })
	return append([]Frame{}, s.frames[i:]...)
}

// Devices returns a copy sorted by identifier.
func (s *Store) Devices() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Device, 0, len(s.devices))
	for _, d := range s.devices {
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Identifier < out[j].Identifier })
	return out
}

// Clear drops all frames and devices. IDs keep increasing so clients that
// resume with ?after= never see a repeated ID.
func (s *Store) Clear() {
	s.mu.Lock()
	s.frames = nil
	s.devices = map[string]*Device{}
	s.mu.Unlock()
}
