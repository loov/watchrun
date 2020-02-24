package jsreload

import "sync"

// Hub dispatches Message to multiple listeners.
type Hub struct {
	mu    sync.RWMutex
	conns map[Listener]struct{}
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		conns: map[Listener]struct{}{},
	}
}

// Listener is a connection that cares about the changes.
type Listener interface {
	// Dispatch must not block
	Dispatch(Message)
}

// Register adds connections to hub.
func (hub *Hub) Register(conn Listener) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.conns[conn] = struct{}{}
}

// Unregister removes connection from hub.
func (hub *Hub) Unregister(conn Listener) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	delete(hub.conns, conn)
}

// Dispatch message to all registered connections.
func (hub *Hub) Dispatch(message Message) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for conn := range hub.conns {
		conn.Dispatch(message)
	}
}
