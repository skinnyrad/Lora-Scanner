// Package hub fans messages out to WebSocket subscribers.
package hub

import "sync"

const clientBuffer = 256

type Hub struct {
	mu      sync.Mutex
	clients map[chan any]struct{}
}

func New() *Hub { return &Hub{clients: map[chan any]struct{}{}} }

// Subscribe returns a message channel and a cancel func. The channel is
// closed if the subscriber falls too far behind or cancels.
func (h *Hub) Subscribe() (<-chan any, func()) {
	ch := make(chan any, clientBuffer)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() { h.drop(ch) }
}

func (h *Hub) drop(ch chan any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
}

// Broadcast never blocks; slow subscribers are dropped.
func (h *Hub) Broadcast(msg any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
			delete(h.clients, ch)
			close(ch)
		}
	}
}
