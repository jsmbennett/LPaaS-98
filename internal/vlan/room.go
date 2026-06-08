package vlan

import (
	"fmt"
	"sync"
	"time"
)

type Peer struct {
	ID        int
	Nickname  string
	IsHost    bool
	Transport Transport
	JoinedAt  time.Time
	BytesIn   int64
	PacketsIn int64
}

type Room struct {
	ID           string
	GameID       string
	NetworkModel string
	HostPeerID   int
	MaxPlayers   int

	mu      sync.RWMutex
	peers   map[int]*Peer
	nextID  int
	closed  bool

	closeCh chan struct{}
}

func NewRoom(id, gameID, networkModel string, maxPlayers int) *Room {
	return &Room{
		ID:           id,
		GameID:       gameID,
		NetworkModel: networkModel,
		MaxPlayers:   maxPlayers,
		peers:        make(map[int]*Peer),
		nextID:       1,
		closeCh:      make(chan struct{}),
	}
}

func (r *Room) AddPeer(nickname string, transport Transport) (*Peer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, fmt.Errorf("room closed")
	}

	if len(r.peers) >= r.MaxPlayers {
		return nil, fmt.Errorf("room full")
	}

	peerID := r.nextID
	r.nextID++

	isHost := len(r.peers) == 0
	if isHost {
		r.HostPeerID = peerID
	}

	peer := &Peer{
		ID:        peerID,
		Nickname:  nickname,
		IsHost:    isHost,
		Transport: transport,
		JoinedAt:  time.Now(),
	}

	r.peers[peerID] = peer
	return peer, nil
}

func (r *Room) RemovePeer(peerID int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.peers, peerID)

	if peerID == r.HostPeerID {
		r.closed = true
	}
}

func (r *Room) Peers() []*Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Peer
	for _, p := range r.peers {
		result = append(result, p)
	}
	return result
}

func (r *Room) GetPeer(peerID int) *Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.peers[peerID]
}

func (r *Room) IsClosed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.closed
}

func (r *Room) ForwardData(fromPeerID int, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	fromPeer := r.peers[fromPeerID]
	if fromPeer == nil {
		return fmt.Errorf("peer not found")
	}

	fromPeer.BytesIn += int64(len(data))
	fromPeer.PacketsIn++

	switch r.NetworkModel {
	case "host-client":
		return r.forwardHostClient(fromPeer, data)
	case "broadcast":
		return r.forwardBroadcast(fromPeer, data)
	default:
		return fmt.Errorf("unknown network model: %s", r.NetworkModel)
	}
}

func (r *Room) forwardBroadcast(fromPeer *Peer, data []byte) error {
	for _, peer := range r.peers {
		if peer.ID == fromPeer.ID {
			continue
		}

		if err := peer.Transport.SendData(data); err != nil {
			continue
		}
	}

	return nil
}

func (r *Room) forwardHostClient(fromPeer *Peer, data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("host-client: no routing byte")
	}

	routeByte := data[0]
	payload := data[1:]

	switch routeByte {
	case 0x00:
		hostPeer := r.peers[r.HostPeerID]
		if hostPeer == nil {
			return fmt.Errorf("host not found")
		}

		if err := hostPeer.Transport.SendData(data); err != nil {
			return err
		}

	case 0xFF:
		if !fromPeer.IsHost {
			return fmt.Errorf("only host can broadcast")
		}

		for _, peer := range r.peers {
			if peer.ID == fromPeer.ID {
				continue
			}

			if err := peer.Transport.SendData(payload); err != nil {
				continue
			}
		}

	default:
		targetID := int(routeByte)
		targetPeer := r.peers[targetID]
		if targetPeer == nil {
			return fmt.Errorf("target peer %d not found", targetID)
		}

		if !fromPeer.IsHost {
			return fmt.Errorf("only host can address specific peers")
		}

		if err := targetPeer.Transport.SendData(payload); err != nil {
			return err
		}
	}

	return nil
}
