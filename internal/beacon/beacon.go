// Package beacon runs a periodic transmit on the server.
package beacon

import (
	"context"
	"sync"
	"time"
)

const MinInterval = time.Second

type State struct {
	Running    bool   `json:"running"`
	Text       string `json:"text"`
	IntervalMs int64  `json:"intervalMs"`
}

type Beacon struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	gen    uint64
	state  State
}

// Start (re)starts the beacon, sending immediately and then every interval.
// It stops itself if send returns an error.
func (b *Beacon) Start(text string, interval time.Duration, send func(string) error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.stopLocked()
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	b.gen++
	gen := b.gen
	b.state = State{Running: true, Text: text, IntervalMs: interval.Milliseconds()}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			if send(text) != nil {
				b.stopIfCurrent(gen)
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

func (b *Beacon) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.stopLocked()
}

func (b *Beacon) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

func (b *Beacon) stopLocked() {
	if b.cancel != nil {
		b.cancel()
		b.cancel = nil
	}
	b.state.Running = false
}

// stopIfCurrent avoids a stale goroutine stopping a newer beacon.
func (b *Beacon) stopIfCurrent(gen uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.gen == gen {
		b.stopLocked()
	}
}
