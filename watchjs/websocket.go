package watchjs

import (
	"io"
	"io/ioutil"
	"sync"

	"github.com/gorilla/websocket"
)

func (server *Server) changes(conn *websocket.Conn) {
	go func() {
		_, _ = io.Copy(ioutil.Discard, conn.UnderlyingConn())
		_ = conn.Close()
	}()

	listener := newWebsocketListener(server.listeners, conn)
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
	listen.conn.Close()
	listen.hub.Unregister(listen)
}
