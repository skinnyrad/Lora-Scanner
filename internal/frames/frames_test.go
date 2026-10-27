package frames

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/skinnyrad/Lora-Scanner/internal/hub"
)

func TestRingAndIDs(t *testing.T) {
	s := NewStore(nil, 3)
	for i := 0; i < 5; i++ {
		s.Add(Frame{Kind: KindRaw, Info: "x"})
	}
	got := s.Since(0)
	if len(got) != 3 || got[0].ID != 3 || got[2].ID != 5 {
		t.Fatalf("ring: %+v", got)
	}
	if after := s.Since(4); len(after) != 1 || after[0].ID != 5 {
		t.Fatalf("since: %+v", after)
	}
	s.Clear()
	if f := s.Add(Frame{Kind: KindRaw}); f.ID != 6 {
		t.Fatalf("ids must keep increasing after Clear, got %d", f.ID)
	}
}

func TestDeviceAggregation(t *testing.T) {
	s := NewStore(nil, 0)
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Minute)
	r1, r2 := -80, -70
	s.Add(Frame{Kind: KindRaw, Info: "noise"})
	s.Add(Frame{Kind: KindLoRaWAN, Identifier: "A", Freq: 914.1, RSSI: &r1, Time: t1})
	s.Add(Frame{Kind: KindLoRaWAN, Identifier: "A", Freq: 914.9, RSSI: &r2, Time: t2})
	ds := s.Devices()
	if len(ds) != 1 {
		t.Fatalf("raw frames must not create devices: %+v", ds)
	}
	d := ds[0]
	if d.Frames != 2 || !d.FirstSeen.Equal(t1) || !d.LastSeen.Equal(t2) || d.Freq != 914.9 || *d.RSSI != -70 {
		t.Fatalf("device: %+v", d)
	}
	s.Clear()
	if len(s.Devices()) != 0 {
		t.Fatal("clear should drop devices")
	}
}

func TestPublishes(t *testing.T) {
	h := hub.New()
	ch, cancel := h.Subscribe()
	defer cancel()
	s := NewStore(h, 0)
	s.Add(Frame{Kind: KindLoRa, Identifier: "d"})
	if m, ok := (<-ch).(FrameMsg); !ok || m.Type != "frame" || m.Frame.ID != 1 {
		t.Fatalf("frame msg: %+v", m)
	}
	if m, ok := (<-ch).(DeviceMsg); !ok || m.Type != "device" || m.Device.Frames != 1 {
		t.Fatalf("device msg: %+v", m)
	}
	s.Add(Frame{Kind: KindRaw})
	select {
	case m := <-ch:
		if _, ok := m.(FrameMsg); !ok {
			t.Fatalf("raw frame should only publish a frame, got %T", m)
		}
	default:
		t.Fatal("expected frame msg")
	}
}

func TestCSV(t *testing.T) {
	rssi := -7
	var buf bytes.Buffer
	WriteCSV(&buf, []Frame{
		{ID: 1, Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Kind: KindLoRa, Source: "Feather 915 MHz", Identifier: "d", Freq: 915, RSSI: &rssi, Info: "a, b"},
		{ID: 2, Time: time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC), Kind: KindRaw, Info: "c"},
	})
	want := "No,Time,Kind,Source,Identifier,Frequency,RSSI,Info\n" +
		"1,2026-01-01T00:00:00Z,lora,Feather 915 MHz,d,915,-7,\"a, b\"\n" +
		"2,2026-01-01T00:00:01Z,raw,,,,,c\n"
	if buf.String() != want {
		t.Fatalf("got %q", buf.String())
	}
}

func TestConcurrent(t *testing.T) {
	s := NewStore(hub.New(), 100)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Add(Frame{Kind: KindLoRa, Identifier: "d"})
			s.Since(0)
			s.Devices()
		}()
	}
	wg.Wait()
}
