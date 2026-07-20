package watchjs

import (
	"io"
	"sync"

	"github.com/gorilla/websocket"
)

func (server *Server) changes(conn *websocket.Conn) {
	listener := newWebsocketListener(server.listeners, conn)
	go func() {
		_, _ = io.Copy(io.Discard, conn.UnderlyingConn())
		listener.mu.Lock()
		defer listener.mu.Unlock()
		listener.internalClose()
	}()

	listener.Dispatch(Message{Type: "hello"})
	server.listeners.Register(listener)
	go listener.writer()
}

type websocketListener struct {
	hub  *Hub
	conn *websocket.Conn
	in   chan Message

	mu     sync.Mutex
	closed bool
}

func newWebsocketListener(hub *Hub, conn *websocket.Conn) *websocketListener {
	return &websocketListener{
		hub:  hub,
		conn: conn,
		in:   make(chan Message, 2),
	}
}

func (listen *websocketListener) writer() {
	for m := range listen.in {
		err := listen.conn.WriteJSON(m)
		if err != nil {
			break
		}
	}

	listen.mu.Lock()
	defer listen.mu.Unlock()
	listen.internalClose()
}

func (listen *websocketListener) Dispatch(message Message) {
	listen.mu.Lock()
	defer listen.mu.Unlock()

	if listen.closed {
		return
	}

	select {
	case listen.in <- message:
	default:
		listen.internalClose()
	}
}

func (listen *websocketListener) internalClose() {
	if listen.closed {
		return
	}

	listen.closed = true
	// closing in stops the writer goroutine; Dispatch never sends
	// after closed is set, so this cannot panic
	close(listen.in)
	listen.conn.Close()
	// async, because Hub.Dispatch calls into Dispatch while holding
	// hub.mu, and Unregister here would deadlock on the same lock
	go listen.hub.Unregister(listen)
}
