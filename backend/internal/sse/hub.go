package sse

import "sync"

// Hub maintains connected SSE clients and fans out broadcast messages.
type Hub struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[chan string]struct{})}
}

func (h *Hub) Register(ch chan string) {
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) Unregister(ch chan string) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

// Broadcast sends data to all connected clients. Slow clients are skipped (non-blocking).
func (h *Hub) Broadcast(data string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- data:
		default:
		}
	}
}
