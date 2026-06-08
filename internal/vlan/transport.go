package vlan

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type Message struct {
	IsText bool
	Data   []byte
}

type Transport interface {
	SendControl(msg interface{}) error
	SendData(data []byte) error
	Recv() (Message, error)
	Close() error
}

type WebSocketTransport struct {
	ws      *websocket.Conn
	mu      sync.Mutex
	closed  bool
	recvCh  chan Message
	errCh   chan error
	stopCh  chan struct{}
}

func NewWebSocketTransport(ws *websocket.Conn) *WebSocketTransport {
	s := &WebSocketTransport{
		ws:     ws,
		recvCh: make(chan Message, 32),
		errCh:  make(chan error, 1),
		stopCh: make(chan struct{}),
	}

	go s.readLoop()
	return s
}

func (t *WebSocketTransport) SendControl(msg interface{}) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return fmt.Errorf("transport closed")
	}
	t.mu.Unlock()

	t.ws.SetDeadline(time.Now().Add(5 * time.Second))
	return websocket.JSON.Send(t.ws, msg)
}

func (t *WebSocketTransport) SendData(data []byte) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return fmt.Errorf("transport closed")
	}
	t.mu.Unlock()

	t.ws.SetDeadline(time.Now().Add(5 * time.Second))
	return websocket.Message.Send(t.ws, data)
}

func (t *WebSocketTransport) Recv() (Message, error) {
	select {
	case msg := <-t.recvCh:
		return msg, nil
	case err := <-t.errCh:
		return Message{}, err
	case <-t.stopCh:
		return Message{}, fmt.Errorf("transport closed")
	}
}

func (t *WebSocketTransport) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	t.mu.Unlock()

	close(t.stopCh)
	return t.ws.Close()
}

func (t *WebSocketTransport) readLoop() {
	for {
		select {
		case <-t.stopCh:
			return
		default:
		}

		t.ws.SetDeadline(time.Now().Add(30 * time.Second))

		var msg []byte
		err := websocket.Message.Receive(t.ws, &msg)
		if err != nil {
			t.errCh <- err
			return
		}

		select {
		case t.recvCh <- Message{IsText: false, Data: msg}:
		case <-t.stopCh:
			return
		}
	}
}
