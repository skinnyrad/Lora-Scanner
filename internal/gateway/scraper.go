package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/skinnyrad/Lora-Scanner/internal/frames"
	"github.com/skinnyrad/Lora-Scanner/internal/hub"
)

const (
	// A dead gateway fails fast, but a slow one (the log page is large and
	// the gateway's CGI is slow) gets time to finish sending it.
	dialTimeout    = 5 * time.Second
	headerTimeout  = 10 * time.Second
	requestTimeout = 45 * time.Second
	// Offline gateways are retried this often until they answer again.
	RetryInterval = 5 * time.Second
	// A gateway is only marked offline after this many failed polls in a row,
	// so one slow or dropped response doesn't flip it.
	FailThreshold = 2
	tickEvery     = time.Second
)

var (
	rssiRe      = regexp.MustCompile(`"Rssi":(-?\d+)`)
	hiddenStyle = "display: none;"
)

// Row is one packet parsed from the gateway's log-traffic page.
type Row struct {
	DevID string
	Freq  float64
	RSSI  *int
	Text  string
	// Detail is the hidden row's JSON-ish text (address, size, RSSI, SNR, MIC...).
	Detail string
}

// Parse extracts packet rows from a Dragino /cgi-bin/log-traffic.has page.
// Visible rows are summaries; each is followed by a hidden row holding JSON-ish
// details (not valid JSON, hence the regex for RSSI).
func Parse(r io.Reader) ([]Row, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}
	table := doc.Find("table").First()
	var rows []Row
	table.Find("tr").Each(func(i int, tr *goquery.Selection) {
		if i == 0 || strings.Contains(tr.AttrOr("style", ""), hiddenStyle) {
			return
		}
		var cells []string
		tr.Find("td").Each(func(_ int, td *goquery.Selection) {
			cells = append(cells, strings.TrimSpace(td.Text()))
		})
		if len(cells) < 2 {
			return
		}
		fields := cells[1:] // first cell is the chevron
		if len(fields) < 4 {
			return
		}
		freq, err := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64)
		if err != nil {
			return
		}
		devID := extractDevID(fields[len(fields)-1])
		if devID == "" {
			return
		}
		row := Row{DevID: devID, Freq: freq, Text: strings.Join(fields, " | ")}
		hidden := tr.NextFiltered("tr")
		if hidden.Length() > 0 && strings.Contains(hidden.AttrOr("style", ""), hiddenStyle) {
			row.Detail = strings.Join(strings.Fields(hidden.Text()), " ")
			if m := rssiRe.FindStringSubmatch(hidden.Text()); m != nil {
				if v, err := strconv.Atoi(m[1]); err == nil {
					row.RSSI = &v
				}
			}
		}
		rows = append(rows, row)
	})
	return rows, nil
}

// extractDevID turns "Dev Addr: 27FD3566, Size: 20" into "27FD3566".
func extractDevID(content string) string {
	first := strings.SplitN(content, ",", 2)[0]
	parts := strings.Split(first, ":")
	return strings.TrimSpace(parts[len(parts)-1])
}

// Gateway states reported to the UI.
const (
	StateUnknown = "unknown" // not polled yet
	StateOnline  = "online"  // last poll succeeded
	StateOffline = "offline" // last poll failed
)

// GatewayStatus is what the last poll of a gateway found.
type GatewayStatus struct {
	State       string     `json:"state"`
	Error       string     `json:"error,omitempty"`
	LastSuccess *time.Time `json:"lastSuccess"`
}

// StatusMsg is broadcast on the hub whenever any gateway changes state.
type StatusMsg struct {
	Type     string                   `json:"type"` // "gateways"
	Gateways map[string]GatewayStatus `json:"gateways"`
}

type gwState struct {
	GatewayStatus
	lastFinish time.Time
	running    bool
	fails      int // consecutive failed polls
}

type Scraper struct {
	Config *Config
	Frames *frames.Store
	Hub    *hub.Hub // optional: receives StatusMsg on state changes
	User   string
	Pass   string
	Client *http.Client
	URLFor func(ip string) string

	mu    sync.Mutex // guards seen and states
	seen  map[string]struct{}
	state map[string]*gwState

	retry time.Duration // offline retry cadence (RetryInterval; tests shorten it)
	tick  time.Duration
}

func NewScraper(cfg *Config, f *frames.Store, user, pass string) *Scraper {
	return &Scraper{
		Config: cfg, Frames: f, User: user, Pass: pass,
		Client: &http.Client{
			Timeout: requestTimeout,
			Transport: &http.Transport{
				// A host that silently drops packets must fail fast, not after the full request timeout.
				DialContext:           (&net.Dialer{Timeout: dialTimeout}).DialContext,
				ResponseHeaderTimeout: headerTimeout,
			},
		},
		URLFor: func(ip string) string { return fmt.Sprintf("http://%s:8000/cgi-bin/log-traffic.has", ip) },
		seen:   map[string]struct{}{},
		state:  map[string]*gwState{},
		retry:  RetryInterval,
		tick:   tickEvery,
	}
}

// Run polls until ctx ends. Each gateway has its own schedule: every
// Config.PollInterval() while online, every RetryInterval while offline or
// confirming a failed poll. A gateway whose previous scrape is still running is skipped.
func (s *Scraper) Run(ctx context.Context) {
	t := time.NewTicker(s.tick)
	defer t.Stop()
	for {
		s.launch(ctx, false)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// ScrapeAll scrapes every idle gateway now, regardless of schedule, and waits.
func (s *Scraper) ScrapeAll(ctx context.Context) { s.launch(ctx, true)() }

// launch starts a scrape for each gateway that is due (or all idle ones when
// force is set) and returns a func that waits for them.
func (s *Scraper) launch(ctx context.Context, force bool) (wait func()) {
	ips := s.Config.List()
	interval := s.Config.PollInterval()
	now := time.Now()

	s.mu.Lock()
	configured := make(map[string]bool, len(ips))
	for _, ip := range ips {
		configured[ip] = true
	}
	for ip, st := range s.state {
		if !configured[ip] && !st.running {
			delete(s.state, ip) // removed from the list
		}
	}
	var start []string
	for _, ip := range ips {
		st := s.state[ip]
		if st == nil {
			st = &gwState{GatewayStatus: GatewayStatus{State: StateUnknown}}
			s.state[ip] = st
		}
		if st.running {
			continue
		}
		wait := interval
		if st.State != StateOnline || st.fails > 0 { // offline, or confirming a failure
			wait = s.retry
		}
		if force || st.lastFinish.IsZero() || now.Sub(st.lastFinish) >= wait {
			st.running = true
			start = append(start, ip)
		}
	}
	s.mu.Unlock()

	var wg sync.WaitGroup
	for _, ip := range start {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			began := time.Now()
			s.finish(ip, s.scrapeOne(ctx, ip), time.Since(began))
		}(ip)
	}
	return wg.Wait
}

// finish records a poll result and announces state changes.
func (s *Scraper) finish(ip string, err error, took time.Duration) {
	s.mu.Lock()
	st := s.state[ip]
	if st == nil { // removed while the scrape was running
		s.mu.Unlock()
		return
	}
	prev := st.State
	st.running = false
	st.lastFinish = time.Now()
	if err != nil {
		st.fails++
		// One failure is only a warning: the state flips after FailThreshold in a row.
		if st.fails >= FailThreshold {
			st.State, st.Error = StateOffline, err.Error()
		}
	} else {
		now := time.Now()
		st.fails = 0
		st.State, st.Error, st.LastSuccess = StateOnline, "", &now
	}
	cur := st.State
	changed := prev != cur
	s.mu.Unlock()

	if !changed {
		return
	}
	if cur == StateOffline {
		fmt.Printf("gateway %s offline after %d failed polls: %v (last took %s); retrying every %s\n",
			ip, FailThreshold, err, took.Round(time.Millisecond), s.retry)
	} else if prev == StateOffline {
		fmt.Printf("gateway %s back online\n", ip)
	}
	if s.Hub != nil {
		s.Hub.Broadcast(s.StatusMsg())
	}
}

// Statuses reports the last poll result for every configured gateway.
func (s *Scraper) Statuses() map[string]GatewayStatus {
	ips := s.Config.List()
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]GatewayStatus, len(ips))
	for _, ip := range ips {
		if st := s.state[ip]; st != nil {
			out[ip] = st.GatewayStatus
		} else {
			out[ip] = GatewayStatus{State: StateUnknown}
		}
	}
	return out
}

func (s *Scraper) StatusMsg() StatusMsg { return StatusMsg{Type: "gateways", Gateways: s.Statuses()} }

func (s *Scraper) request(ctx context.Context, method, ip string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, s.URLFor(ip), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.User, s.Pass)
	return s.Client.Do(req)
}

func (s *Scraper) scrapeOne(ctx context.Context, ip string) error {
	resp, err := s.request(ctx, http.MethodGet, ip)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %s", resp.Status)
	}
	// Read the whole page first so a stalled download is reported as such and
	// nothing is half-parsed; the next poll sees the same rows again (deduped).
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading page (got %d bytes): %w", len(body), err)
	}
	rows, err := Parse(bytes.NewReader(body))
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Oldest first so survey order is chronological (the page lists newest first).
	for i := len(rows) - 1; i >= 0; i-- {
		r := rows[i]
		key := fmt.Sprintf("%s_%v_%s", r.DevID, r.Freq, r.Text)
		if _, dup := s.seen[key]; dup {
			continue
		}
		s.seen[key] = struct{}{}
		raw := r.Text
		if r.Detail != "" {
			raw += "\n" + r.Detail
		}
		s.Frames.Add(frames.Frame{
			Kind: frames.KindLoRaWAN, Source: ip, Identifier: r.DevID,
			Freq: r.Freq, RSSI: r.RSSI, Info: r.Text, Raw: raw,
		})
	}
	return nil
}
