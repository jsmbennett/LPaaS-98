package lobby

import (
	"fmt"
	"sync"
)

type Client struct {
	ID       string
	Nickname string
	RoomID   string
}

type Room struct {
	ID         string
	GameID     string
	HostID     string
	Members    map[string]*Client
	MaxPlayers int
}

type Lobby struct {
	mu      sync.RWMutex
	rooms   map[string]*Room
	clients map[string]*Client
	roomSeq int
}

func New() *Lobby {
	return &Lobby{
		rooms:   make(map[string]*Room),
		clients: make(map[string]*Client),
	}
}

func (l *Lobby) AddClient(clientID, nickname string) (*Client, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.clients[clientID]; exists {
		return nil, fmt.Errorf("client already exists")
	}

	client := &Client{
		ID:       clientID,
		Nickname: nickname,
	}

	l.clients[clientID] = client
	return client, nil
}

func (l *Lobby) RemoveClient(clientID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	client, exists := l.clients[clientID]
	if !exists {
		return
	}

	if client.RoomID != "" {
		if room, ok := l.rooms[client.RoomID]; ok {
			delete(room.Members, clientID)
			if len(room.Members) == 0 {
				delete(l.rooms, client.RoomID)
			}
		}
	}

	delete(l.clients, clientID)
}

func (l *Lobby) CreateRoom(gameID, hostID string, maxPlayers int) (*Room, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	host, exists := l.clients[hostID]
	if !exists {
		return nil, fmt.Errorf("host client not found")
	}

	if host.RoomID != "" {
		return nil, fmt.Errorf("host already in a room")
	}

	l.roomSeq++
	roomID := fmt.Sprintf("room-%d", l.roomSeq)

	room := &Room{
		ID:         roomID,
		GameID:     gameID,
		HostID:     hostID,
		Members:    make(map[string]*Client),
		MaxPlayers: maxPlayers,
	}

	room.Members[hostID] = host
	l.rooms[roomID] = room
	host.RoomID = roomID

	return room, nil
}

func (l *Lobby) JoinRoom(clientID, roomID string) (*Room, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	client, exists := l.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client not found")
	}

	if client.RoomID != "" {
		return nil, fmt.Errorf("client already in a room")
	}

	room, exists := l.rooms[roomID]
	if !exists {
		return nil, fmt.Errorf("room not found")
	}

	if len(room.Members) >= room.MaxPlayers {
		return nil, fmt.Errorf("room is full")
	}

	room.Members[clientID] = client
	client.RoomID = roomID

	return room, nil
}

func (l *Lobby) LeaveRoom(clientID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	client, exists := l.clients[clientID]
	if !exists || client.RoomID == "" {
		return
	}

	roomID := client.RoomID
	if room, ok := l.rooms[roomID]; ok {
		delete(room.Members, clientID)
		if len(room.Members) == 0 {
			delete(l.rooms, roomID)
		}
	}

	client.RoomID = ""
}

func (l *Lobby) ListRooms() []*Room {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var result []*Room
	for _, room := range l.rooms {
		result = append(result, room)
	}

	return result
}

func (l *Lobby) GetRoom(roomID string) *Room {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.rooms[roomID]
}

func (l *Lobby) GetClient(clientID string) *Client {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.clients[clientID]
}
