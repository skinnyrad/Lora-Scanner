package gateway

import (
	"context"
	"errors"
	"fmt"
	"github.com/skinnyrad/Lora-Scanner/internal/hub"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/skinnyrad/Lora-Scanner/internal/frames"
)

func TestParseFixture(t *testing.T) {
	f, err := os.Open("testdata/log-traffic.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 3 {
		t.Fatalf("got %d rows", len(rows))
	}
	r := rows[0]
	if r.DevID != "27FD3566" || r.Freq != 914.1 || r.RSSI == nil || *r.RSSI != -24 {
		t.Fatalf("bad first row %+v", r)
	}
	want := "01/30-13:53:13 | Data Unconfirmed Up | LoRa | 914.1 | SF10 BW125 | 12 | Dev Addr: 27FD3566, Size: 20"
	if r.Text != want {
		t.Fatalf("text %q", r.Text)
	}
	if !strings.Contains(r.Detail, `"Rssi":-24`) {
		t.Fatalf("detail %q", r.Detail)
	}
	if *rows[1].RSSI != -93 {
		t.Fatalf("second rssi %v", *rows[1].RSSI)
	}
}

func TestScrapeDedupesAndAuths(t *testing.T) {
	page, _ := os.ReadFile("testdata/log-traffic.html")
	var gotUser, gotPass string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, _ = r.BasicAuth()
		w.Write(page)
	}))
	defer ts.Close()

	cfg := NewConfig(filepath.Join(t.TempDir(), "c.json"))
	if _, err := cfg.Add("127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	st := frames.NewStore(nil, 0)
	sc := NewScraper(cfg, st, "root", "dragino")
	sc.URLFor = func(string) string { return ts.URL }

	sc.ScrapeAll(context.Background())
	n := len(st.Since(0))
	if n == 0 {
		t.Fatal("no frames scraped")
	}
	f := st.Since(0)[0]
	if f.Kind != frames.KindLoRaWAN || f.Source != "127.0.0.1" || f.Identifier != "27FD3566" ||
		f.RSSI == nil || !strings.Contains(f.Raw, `"Rssi":`) {
		t.Fatalf("frame: %+v", f)
	}
	if ds := st.Devices(); len(ds) != 1 || ds[0].Identifier != "27FD3566" || ds[0].Frames != n {
		t.Fatalf("devices: %+v", ds)
	}
	sc.ScrapeAll(context.Background())
	if len(st.Since(0)) != n {
		t.Fatal("second scrape added duplicates")
	}
	if gotUser != "root" || gotPass != "dragino" {
		t.Fatalf("auth %q:%q", gotUser, gotPass)
	}
	if st := sc.Statuses()["127.0.0.1"]; st.State != StateOnline || st.LastSuccess == nil || st.Error != "" {
		t.Fatalf("status: %+v", st)
	}
}

func TestConfigAddRemovePersist(t *testing.T) {
	p := filepath.Join(t.TempDir(), "gateways.json")
	c := NewConfig(p)
	if err := c.Load(""); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "999.1.1.1", "nope", "::1"} {
		if _, err := c.Add(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
	// More than the old cap of 10, with whitespace tolerated.
	for i := 1; i <= 25; i++ {
		if _, err := c.Add(fmt.Sprintf(" 10.0.0.%d ", i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.Add("10.0.0.3"); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("want ErrDuplicate, got %v", err)
	}
	if err := c.Remove("10.0.0.2"); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove("10.0.0.2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	c2 := NewConfig(p)
	if err := c2.Load(""); err != nil {
		t.Fatal(err)
	}
	got := c2.List()
	if len(got) != 24 || got[0] != "10.0.0.1" || got[1] != "10.0.0.3" || got[23] != "10.0.0.25" {
		t.Fatalf("reload: %v", got)
	}
}

func TestConfigCreatesMissingDirectory(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a", "b", "gateways.json")
	c := NewConfig(p)
	if _, err := c.Add("10.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("config not written: %v", err)
	}
}

func TestDefaultPathIsAbsoluteUserDir(t *testing.T) {
	if p := DefaultPath(); !filepath.IsAbs(p) || filepath.Base(p) != "gateways.json" || filepath.Base(filepath.Dir(p)) != "lora-scanner" {
		t.Fatalf("default path %q", p)
	}
}

func TestConfigImportsLegacyFlatFile(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "gateway_config.json")
	body := `{"gateway1": "192.168.8.101", "gateway10": "192.168.8.110", "gateway2": "", "gateway3": "192.168.8.103"}`
	os.WriteFile(legacy, []byte(body), 0o644)

	c := NewConfig(filepath.Join(dir, "gateways.json"))
	if err := c.Load(legacy); err != nil {
		t.Fatal(err)
	}
	want := []string{"192.168.8.101", "192.168.8.103", "192.168.8.110"}
	if got := c.List(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("import: %v", got)
	}
	if b, _ := os.ReadFile(legacy); string(b) != body {
		t.Fatal("legacy file must not be modified")
	}
	if _, err := os.Stat(filepath.Join(dir, "gateways.json")); err == nil {
		t.Fatal("import alone should not write the new file")
	}
	if _, err := c.Add("10.1.1.1"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(legacy); string(b) != body {
		t.Fatal("legacy file must not be modified after save")
	}

	// A legacy-format file at the main path loads too.
	c3 := NewConfig(legacy)
	if err := c3.Load(""); err != nil || len(c3.List()) != 3 {
		t.Fatalf("flat at main path: %v %v", err, c3.List())
	}
}

func TestPollIntervalPersistsAndValidates(t *testing.T) {
	p := filepath.Join(t.TempDir(), "gateways.json")
	c := NewConfig(p)
	if c.PollInterval() != 30*time.Second {
		t.Fatalf("default %v", c.PollInterval())
	}
	for _, bad := range []time.Duration{0, time.Second, 4 * time.Second, 2 * time.Hour} {
		if err := c.SetPollInterval(bad); !errors.Is(err, ErrBadInterval) {
			t.Errorf("%v: want ErrBadInterval, got %v", bad, err)
		}
	}
	if err := c.SetPollInterval(90 * time.Second); err != nil {
		t.Fatal(err)
	}
	// Adding a gateway must not reset the interval.
	if _, err := c.Add("10.0.0.1"); err != nil {
		t.Fatal(err)
	}
	c2 := NewConfig(p)
	if err := c2.Load(); err != nil || c2.PollInterval() != 90*time.Second || len(c2.List()) != 1 {
		t.Fatalf("reload: %v %v %v", err, c2.PollInterval(), c2.List())
	}
	// Legacy imports keep the default.
	if c3 := NewConfig(filepath.Join(t.TempDir(), "x.json")); c3.PollInterval() != DefaultPollInterval {
		t.Fatal("default lost")
	}
}

func TestOfflineGatewayRetriesFastThenSettles(t *testing.T) {
	page, _ := os.ReadFile("testdata/log-traffic.html")
	var hits atomic.Int32
	var failing atomic.Bool
	failing.Store(true)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if failing.Load() {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Write(page)
	}))
	defer ts.Close()

	h := hub.New()
	events, cancelSub := h.Subscribe()
	defer cancelSub()
	cfg := NewConfig(filepath.Join(t.TempDir(), "g.json"))
	cfg.Add("127.0.0.1")
	sc := NewScraper(cfg, frames.NewStore(nil, 0), "u", "p")
	sc.Hub = h
	sc.URLFor = func(string) string { return ts.URL }
	sc.retry, sc.tick = 40*time.Millisecond, 10*time.Millisecond

	if st := sc.Statuses()["127.0.0.1"]; st.State != StateUnknown {
		t.Fatalf("before first poll: %+v", st)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sc.Run(ctx)

	waitFor := func(what string, cond func() bool) {
		t.Helper()
		for i := 0; i < 200; i++ {
			if cond() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("timed out: %s", what)
	}
	waitFor("first poll", func() bool { return hits.Load() >= 1 })
	waitFor("offline", func() bool { return sc.Statuses()["127.0.0.1"].State == StateOffline })
	if hits.Load() < FailThreshold {
		t.Fatalf("went offline after %d poll(s), want %d", hits.Load(), FailThreshold)
	}
	if e := sc.Statuses()["127.0.0.1"].Error; !strings.Contains(e, "500") {
		t.Fatalf("error should say why: %q", e)
	}
	// Offline gateways are retried on the short cadence, not the 30s poll interval.
	waitFor("retries", func() bool { return hits.Load() >= 4 })

	failing.Store(false)
	waitFor("online", func() bool { return sc.Statuses()["127.0.0.1"].State == StateOnline })
	settled := hits.Load()
	time.Sleep(250 * time.Millisecond) // many retry periods
	if got := hits.Load(); got > settled+1 {
		t.Fatalf("online gateway still polled at retry cadence: %d -> %d", settled, got)
	}

	// Exactly two transitions were announced: unknown->offline, offline->online.
	var states []string
	for len(events) > 0 {
		m := (<-events).(StatusMsg)
		states = append(states, m.Gateways["127.0.0.1"].State)
	}
	if strings.Join(states, ",") != "offline,online" {
		t.Fatalf("broadcasts: %v", states)
	}
}

func TestRemovedGatewayStatusIsDropped(t *testing.T) {
	cfg := NewConfig(filepath.Join(t.TempDir(), "g.json"))
	cfg.Add("127.0.0.1")
	sc := NewScraper(cfg, frames.NewStore(nil, 0), "u", "p")
	sc.URLFor = func(string) string { return "http://127.0.0.1:1" } // refused
	sc.ScrapeAll(context.Background())
	sc.ScrapeAll(context.Background())
	if sc.Statuses()["127.0.0.1"].State != StateOffline {
		t.Fatal("expected offline")
	}
	cfg.Remove("127.0.0.1")
	if len(sc.Statuses()) != 0 {
		t.Fatal("removed gateway must not be reported")
	}
}

func TestSlowGatewayDoesNotBlockOthersOrOverlap(t *testing.T) {
	page, _ := os.ReadFile("testdata/log-traffic.html")
	var slowHits, fastHits atomic.Int32
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slowHits.Add(1)
		time.Sleep(600 * time.Millisecond)
		w.Write(page)
	}))
	defer slow.Close()
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fastHits.Add(1)
		w.Write(page)
	}))
	defer fast.Close()

	cfg := NewConfig(filepath.Join(t.TempDir(), "g.json"))
	cfg.Add("10.0.0.1") // slow
	cfg.Add("10.0.0.2") // fast
	sc := NewScraper(cfg, frames.NewStore(nil, 0), "u", "p")
	sc.URLFor = func(ip string) string {
		if ip == "10.0.0.1" {
			return slow.URL
		}
		return fast.URL
	}

	ctx := context.Background()
	w1 := sc.launch(ctx, true)
	time.Sleep(150 * time.Millisecond)
	// Next cycle while the slow one is still running: only the fast gateway is scraped.
	w2 := sc.launch(ctx, true)
	w1()
	w2()
	if slowHits.Load() != 1 {
		t.Errorf("slow gateway scraped %d times, want 1 (no overlap)", slowHits.Load())
	}
	if fastHits.Load() != 2 {
		t.Errorf("fast gateway scraped %d times, want 2", fastHits.Load())
	}
	// Once finished it is scraped again.
	sc.ScrapeAll(ctx)
	if slowHits.Load() != 2 {
		t.Errorf("slow gateway not rescheduled: %d", slowHits.Load())
	}
}

func TestStalledBodyIsReportedAndRetriedCleanly(t *testing.T) {
	page, _ := os.ReadFile("testdata/log-traffic.html")
	var stall atomic.Bool
	stall.Store(true)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if stall.Load() {
			w.Write(page[:len(page)/2]) // headers + half the page, then hang
			w.(http.Flusher).Flush()
			time.Sleep(500 * time.Millisecond)
			return
		}
		w.Write(page)
	}))
	defer ts.Close()

	cfg := NewConfig(filepath.Join(t.TempDir(), "g.json"))
	cfg.Add("127.0.0.1")
	st := frames.NewStore(nil, 0)
	sc := NewScraper(cfg, st, "u", "p")
	sc.URLFor = func(string) string { return ts.URL }
	sc.Client.Timeout = 150 * time.Millisecond

	if err := sc.scrapeOne(context.Background(), "127.0.0.1"); err == nil || !strings.Contains(err.Error(), "reading page") {
		t.Fatalf("want a reading-page error, got %v", err)
	}
	if len(st.Since(0)) != 0 {
		t.Fatal("a truncated page must not produce frames")
	}
	stall.Store(false)
	if err := sc.scrapeOne(context.Background(), "127.0.0.1"); err != nil || len(st.Since(0)) == 0 {
		t.Fatalf("retry: %v, %d frames", err, len(st.Since(0)))
	}
}

func TestSingleFailureDoesNotFlipOnlineGateway(t *testing.T) {
	page, _ := os.ReadFile("testdata/log-traffic.html")
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 2 { // exactly one bad response
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Write(page)
	}))
	defer ts.Close()

	h := hub.New()
	events, cancelSub := h.Subscribe()
	defer cancelSub()
	cfg := NewConfig(filepath.Join(t.TempDir(), "g.json"))
	cfg.Add("127.0.0.1")
	sc := NewScraper(cfg, frames.NewStore(nil, 0), "u", "p")
	sc.Hub = h
	sc.URLFor = func(string) string { return ts.URL }
	sc.retry = 20 * time.Millisecond

	ctx := context.Background()
	sc.ScrapeAll(ctx) // ok -> online
	if sc.Statuses()["127.0.0.1"].State != StateOnline {
		t.Fatal("expected online")
	}
	sc.ScrapeAll(ctx) // the one failure
	st := sc.Statuses()["127.0.0.1"]
	if st.State != StateOnline || st.Error != "" {
		t.Fatalf("a single failure must not change the state: %+v", st)
	}
	// The failed gateway is rechecked on the short cadence to confirm.
	time.Sleep(40 * time.Millisecond)
	sc.launch(ctx, false)()
	if hits.Load() != 3 {
		t.Fatalf("confirming poll not made on retry cadence: %d hits", hits.Load())
	}
	if sc.state["127.0.0.1"].fails != 0 {
		t.Fatal("success must reset the failure count")
	}
	// Online->online: nothing was announced beyond the initial unknown->online.
	n := 0
	for len(events) > 0 {
		<-events
		n++
	}
	if n != 1 {
		t.Fatalf("expected 1 broadcast, got %d", n)
	}
}

func TestTwoFailuresInARowGoOffline(t *testing.T) {
	cfg := NewConfig(filepath.Join(t.TempDir(), "g.json"))
	cfg.Add("127.0.0.1")
	sc := NewScraper(cfg, frames.NewStore(nil, 0), "u", "p")
	sc.URLFor = func(string) string { return "http://127.0.0.1:1" } // refused
	sc.ScrapeAll(context.Background())
	if st := sc.Statuses()["127.0.0.1"]; st.State != StateUnknown {
		t.Fatalf("after 1 failure: %+v", st)
	}
	sc.ScrapeAll(context.Background())
	if st := sc.Statuses()["127.0.0.1"]; st.State != StateOffline || st.Error == "" {
		t.Fatalf("after 2 failures: %+v", st)
	}
}
