package vlan

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"golang.org/x/net/websocket"
)

type Relay struct {
	Mu    sync.RWMutex
	rooms map[string]*Room
}

func NewRelay() *Relay {
	return &Relay{
		rooms: make(map[string]*Room),
	}
}

func (r *Relay) ServeRoom(ws *websocket.Conn) {
	roomID := ws.Request().PathValue("roomID")
	nickname := ws.Request().URL.Query().Get("nickname")

	if nickname == "" {
		ws.Close()
		return
	}

	r.Mu.RLock()
	room := r.rooms[roomID]
	r.Mu.RUnlock()

	if room == nil {
		ws.Close()
		return
	}

	transport := NewWebSocketTransport(ws)

	peer, err := room.AddPeer(nickname, transport)
	if err != nil {
		transport.SendControl(NewError("room_full", "Room is full"))
		transport.Close()
		return
	}

	r.handlePeer(room, peer)
}

func (r *Relay) CreateRoom(roomID, gameID, networkModel string, maxPlayers int) (*Room, error) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if _, exists := r.rooms[roomID]; exists {
		return nil, fmt.Errorf("room already exists")
	}

	room := NewRoom(roomID, gameID, networkModel, maxPlayers)
	r.rooms[roomID] = room
	return room, nil
}

func (r *Relay) GetRoom(roomID string) *Room {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	return r.rooms[roomID]
}

func (r *Relay) handlePeer(room *Room, peer *Peer) {
	slog.Info("Peer joined", "room", room.ID, "peer", peer.ID, "nickname", peer.Nickname)

	err := peer.Transport.SendControl(NewHello(
		peer.Nickname,
		peer.ID,
		peer.IsHost,
		&RoomInfo{
			ID:           room.ID,
			GameID:       room.GameID,
			NetworkModel: room.NetworkModel,
			Peers:        r.getRoomPeers(room),
			MaxPlayers:   room.MaxPlayers,
		},
	))
	if err != nil {
		slog.Error("Failed to send hello", "error", err)
		return
	}

	r.broadcastPeerJoined(room, peer)

	for {
		msg, err := peer.Transport.Recv()
		if err != nil {
			break
		}

		if msg.IsText {
			r.handleControlMessage(room, peer, msg.Data)
		} else {
			room.ForwardData(peer.ID, msg.Data)
		}
	}

	room.RemovePeer(peer.ID)
	r.broadcastPeerLeft(room, peer.ID)

	if room.IsClosed() {
		r.Mu.Lock()
		delete(r.rooms, room.ID)
		r.Mu.Unlock()
	}

	slog.Info("Peer left", "room", room.ID, "peer", peer.ID)
}

func (r *Relay) handleControlMessage(room *Room, peer *Peer, data []byte) {
	var msg struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		peer.Transport.SendControl(NewError("invalid_json", "Could not parse JSON"))
		return
	}

	switch msg.Type {
	case "ping":
		var ping Ping
		if err := json.Unmarshal(data, &ping); err == nil {
			peer.Transport.SendControl(Pong{Type: "pong", ID: ping.ID})
		}
	case "ready":
		slog.Info("Peer ready", "room", room.ID, "peer", peer.ID)
	case "":
		break
	default:
		slog.Debug("Unknown control message type", "type", msg.Type)
	}
}

func (r *Relay) broadcastPeerJoined(room *Room, newPeer *Peer) {
	msg := NewPeerJoined(newPeer.ID, newPeer.Nickname, newPeer.IsHost)

	for _, peer := range room.Peers() {
		if peer.ID != newPeer.ID {
			peer.Transport.SendControl(msg)
		}
	}
}

func (r *Relay) broadcastPeerLeft(room *Room, peerID int) {
	msg := NewPeerLeft(peerID)

	for _, peer := range room.Peers() {
		peer.Transport.SendControl(msg)
	}
}

func (r *Relay) getRoomPeers(room *Room) []PeerInfo {
	var result []PeerInfo
	for _, peer := range room.Peers() {
		result = append(result, PeerInfo{
			PeerID:   peer.ID,
			Nickname: peer.Nickname,
			IsHost:   peer.IsHost,
		})
	}
	return result
}
