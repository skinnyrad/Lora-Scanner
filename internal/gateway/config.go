// Package gateway stores gateway IPs and scrapes Dragino traffic logs.
package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrDuplicate = errors.New("gateway already added")
	ErrNotFound  = errors.New("gateway not found")
)

const (
	DefaultPollInterval = 30 * time.Second
	MinPollInterval     = 5 * time.Second
	MaxPollInterval     = time.Hour
)

var ErrBadInterval = fmt.Errorf("poll interval must be between %d and %d seconds",
	int(MinPollInterval/time.Second), int(MaxPollInterval/time.Second))

// Config is a mutex-guarded list of gateway IPv4 addresses plus the poll
// interval, persisted as {"gateways": ["192.168.8.101", ...], "pollIntervalSeconds": 30}.
type Config struct {
	mu       sync.RWMutex
	path     string
	ips      []string
	interval time.Duration
}

func NewConfig(path string) *Config {
	return &Config{path: path, interval: DefaultPollInterval}
}

// PollInterval is how often gateways are scraped.
func (c *Config) PollInterval() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.interval
}

func (c *Config) SetPollInterval(d time.Duration) error {
	if d < MinPollInterval || d > MaxPollInterval {
		return ErrBadInterval
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.save(c.ips, d); err != nil {
		return err
	}
	c.interval = d
	return nil
}

// DefaultPath is <user config dir>/lora-scanner/gateways.json (e.g.
// ~/.config on Linux, ~/Library/Application Support on macOS, %AppData% on
// Windows), so the binary can live anywhere, such as a read-only bin directory.
// It falls back to the working directory if there is no user config dir.
func DefaultPath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "lora-scanner", "gateways.json")
	}
	return "gateways.json"
}

// Load reads the file. If it doesn't exist, the first existing legacyPaths
// entry (the Flask app's flat {"gateway1": ..., "gateway10": ...} file) is
// imported read-only: the old file is never modified. A missing file is not an error.
func (c *Config) Load(legacyPaths ...string) error {
	b, err := os.ReadFile(c.path)
	if err == nil {
		return c.parse(b, c.path)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, p := range legacyPaths {
		b, err := os.ReadFile(p)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		return c.parse(b, p)
	}
	return nil
}

// parse accepts {"gateways": [...]} or the legacy flat gateway1..N object.
func (c *Config) parse(b []byte, name string) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	if list, ok := raw["gateways"]; ok {
		var ips []string
		if err := json.Unmarshal(list, &ips); err != nil {
			return fmt.Errorf("parse %s: %w", name, err)
		}
		if v, ok := raw["pollIntervalSeconds"]; ok {
			var secs int
			if err := json.Unmarshal(v, &secs); err != nil {
				return fmt.Errorf("parse %s: %w", name, err)
			}
			if d := time.Duration(secs) * time.Second; d >= MinPollInterval && d <= MaxPollInterval {
				c.mu.Lock()
				c.interval = d
				c.mu.Unlock()
			}
		}
		return c.setAll(ips)
	}
	type kv struct {
		n  int
		ip string
	}
	var found []kv
	for k, v := range raw {
		var ip string
		n, err := strconv.Atoi(strings.TrimPrefix(k, "gateway"))
		if err == nil && strings.HasPrefix(k, "gateway") && json.Unmarshal(v, &ip) == nil && strings.TrimSpace(ip) != "" {
			found = append(found, kv{n, ip})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].n < found[j].n })
	ips := make([]string, 0, len(found))
	for _, f := range found {
		ips = append(ips, f.ip)
	}
	return c.setAll(ips)
}

// setAll validates and replaces the list in memory (no save).
func (c *Config) setAll(in []string) error {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		ip, err := normalize(raw)
		if err != nil {
			return err
		}
		if !seen[ip] {
			seen[ip] = true
			out = append(out, ip)
		}
	}
	c.mu.Lock()
	c.ips = out
	c.mu.Unlock()
	return nil
}

func normalize(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	ip := net.ParseIP(raw)
	if ip == nil || ip.To4() == nil {
		return "", fmt.Errorf("%q is not a valid IPv4 address", raw)
	}
	return ip.To4().String(), nil
}

// List returns the gateway IPs in the order they were added.
func (c *Config) List() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string{}, c.ips...)
}

// Add validates, de-duplicates and persists. It returns the normalized IP.
func (c *Config) Add(raw string) (string, error) {
	ip, err := normalize(raw)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, have := range c.ips {
		if have == ip {
			return "", ErrDuplicate
		}
	}
	next := append(append([]string{}, c.ips...), ip)
	if err := c.save(next, c.interval); err != nil {
		return "", err
	}
	c.ips = next
	return ip, nil
}

func (c *Config) Remove(raw string) error {
	ip, err := normalize(raw)
	if err != nil {
		return ErrNotFound
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	next := make([]string, 0, len(c.ips))
	found := false
	for _, have := range c.ips {
		if have == ip {
			found = true
			continue
		}
		next = append(next, have)
	}
	if !found {
		return ErrNotFound
	}
	if err := c.save(next, c.interval); err != nil {
		return err
	}
	c.ips = next
	return nil
}

func (c *Config) save(ips []string, interval time.Duration) error {
	b, err := json.MarshalIndent(struct {
		Gateways            []string `json:"gateways"`
		PollIntervalSeconds int      `json:"pollIntervalSeconds"`
	}{ips, int(interval / time.Second)}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(c.path, append(b, '\n'), 0o644)
}
