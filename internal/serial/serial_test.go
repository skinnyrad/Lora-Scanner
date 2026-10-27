package serial

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/skinnyrad/Lora-Scanner/internal/frames"
	"github.com/skinnyrad/Lora-Scanner/internal/hub"
)

func TestParserPairsLines(t *testing.T) {
	var p Parser
	if _, _, ok := p.Feed("noise"); ok {
		t.Fatal("unexpected packet")
	}
	if _, _, ok := p.Feed("RSSI: -80"); ok {
		t.Fatal("rssi alone should not complete")
	}
	rssi, dec, ok := p.Feed("Decoded Value: hello world")
	if !ok || rssi != -80 || dec != "hello world" {
		t.Fatalf("got %v %q %v", rssi, dec, ok)
	}
	if _, _, ok := p.Feed("Decoded Value: again"); ok {
		t.Fatal("state should reset after a packet")
	}
}

// fakePort is a duplex pipe: device->app via r, app->device captured in written.
type fakePort struct {
	r       *io.PipeReader
	w       *io.PipeWriter
	mu      sync.Mutex
	written []byte
}

func newFake() *fakePort {
	r, w := io.Pipe()
	return &fakePort{r: r, w: w}
}
func (f *fakePort) Read(b []byte) (int, error) { return f.r.Read(b) }
func (f *fakePort) Write(b []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.written = append(f.written, b...)
	return len(b), nil
}
func (f *fakePort) Close() error { return f.r.Close() }

func eventually(t *testing.T, cond func() bool) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met")
}

func TestParseFreq(t *testing.T) {
	for line, want := range map[string]float64{"Freq set to: 915.00": 915, "Freq set to: 433.0": 433, "Freq set to: 902.3": 902.3} {
		if f, ok := ParseFreq(line); !ok || f != want {
			t.Errorf("%q -> %v %v", line, f, ok)
		}
	}
	for _, line := range []string{"Welcome to the skinny-lora transceiver", "Freq set to: abc", "RSSI: -80"} {
		if _, ok := ParseFreq(line); ok {
			t.Errorf("%q should not parse", line)
		}
	}
}

func newTestManager(t *testing.T, ports map[string]*fakePort) (*Manager, *frames.Store) {
	h := hub.New()
	st := frames.NewStore(h, 0)
	m := NewManager(h, st)
	m.open = func(name string) (io.ReadWriteCloser, error) {
		if p, ok := ports[name]; ok {
			return p, nil
		}
		return nil, io.ErrClosedPipe
	}
	return m, st
}

func TestManagerLifecycle(t *testing.T) {
	fake := newFake()
	m, st := newTestManager(t, map[string]*fakePort{"/dev/x": fake})

	if _, err := m.Add("/dev/missing"); err == nil {
		t.Fatal("expected open error")
	}
	tr, err := m.Add("/dev/x")
	if err != nil || tr.ID != 1 || tr.Freq != nil {
		t.Fatalf("add: %+v %v", tr, err)
	}
	if _, err := m.Add("/dev/x"); err != ErrPortInUse {
		t.Fatalf("want ErrPortInUse, got %v", err)
	}

	// Frequency is detected from the firmware banner.
	fake.w.Write([]byte("Welcome to the skinny-lora transceiver\nFreq set to: 915.00\n"))
	eventually(t, func() bool { l := m.List(); return len(l) == 1 && l[0].Freq != nil })
	if f := *m.List()[0].Freq; f != 915 {
		t.Fatalf("freq %v", f)
	}

	fake.w.Write([]byte("RSSI: -42\nDecoded Value: hi\n"))
	eventually(t, func() bool { return len(st.Since(0)) == 5 })
	fs := st.Since(0)
	if fs[0].Kind != frames.KindRaw || fs[0].Transceiver != 1 || fs[1].Info != "Freq set to: 915.00" || fs[1].Source != "x (915 MHz)" {
		t.Fatalf("raw frames: %+v", fs)
	}
	lora := fs[4]
	if lora.Kind != frames.KindLoRa || lora.Identifier != "Raw LoRa Device 915 MHz" ||
		lora.RSSI == nil || *lora.RSSI != -42 || lora.Freq != 915 || lora.Info != "hi" {
		t.Fatalf("lora frame: %+v", lora)
	}
	if ds := st.Devices(); len(ds) != 1 || ds[0].Frames != 1 {
		t.Fatalf("devices: %+v", ds)
	}

	if err := m.Transmit(1, "ping"); err != nil {
		t.Fatal(err)
	}
	if got := string(fake.written); got != "TX:ping " {
		t.Fatalf("wrote %q", got)
	}
	if err := m.Transmit(9, "x"); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	if err := m.StartBeacon(1, "b", time.Second); err != nil {
		t.Fatal(err)
	}
	if err := m.Remove(1); err != nil {
		t.Fatal(err)
	}
	if len(m.List()) != 0 {
		t.Fatal("transceiver should be gone")
	}
	if err := m.Remove(1); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestReaddRemembersFrequency(t *testing.T) {
	fake := newFake()
	m, _ := newTestManager(t, map[string]*fakePort{"/dev/x": fake})
	if _, err := m.Add("/dev/x"); err != nil {
		t.Fatal(err)
	}
	fake.w.Write([]byte("Freq set to: 868.0\n"))
	eventually(t, func() bool { return m.List()[0].Freq != nil })
	m.Remove(1)

	// A board that is already running won't print the banner again.
	fake2 := newFake()
	m.open = func(string) (io.ReadWriteCloser, error) { return fake2, nil }
	tr, err := m.Add("/dev/x")
	if err != nil || tr.Freq == nil || *tr.Freq != 868 || tr.ID != 2 {
		t.Fatalf("re-add: %+v %v", tr, err)
	}
	m.Remove(2)
}

func TestUnplugReleasesTransceiver(t *testing.T) {
	fake := newFake()
	m, _ := newTestManager(t, map[string]*fakePort{"/dev/x": fake})
	if _, err := m.Add("/dev/x"); err != nil {
		t.Fatal(err)
	}
	fake.w.Close() // EOF, like a vanished device
	eventually(t, func() bool { return len(m.List()) == 0 })
	if _, err := m.Add("/dev/x"); err != nil {
		t.Fatalf("port should be free after unplug: %v", err)
	}
	m.Remove(2)
}
