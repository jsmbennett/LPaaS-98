package vlan

import (
	"encoding/json"
)

type ControlMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"-"`
}

type Hello struct {
	Type    string `json:"type"`
	You     string `json:"you"`
	PeerID  int    `json:"peer_id"`
	IsHost  bool   `json:"is_host"`
	Room    RoomInfo `json:"room"`
}

type RoomInfo struct {
	ID           string `json:"id"`
	GameID       string `json:"game_id"`
	NetworkModel string `json:"network_model"`
	Peers        []PeerInfo `json:"peers"`
	MaxPlayers   int `json:"max_players"`
}

type PeerInfo struct {
	PeerID   int    `json:"peer_id"`
	Nickname string `json:"nickname"`
	IsHost   bool   `json:"is_host"`
}

type PeerJoined struct {
	Type     string `json:"type"`
	PeerID   int    `json:"peer_id"`
	Nickname string `json:"nickname"`
	IsHost   bool   `json:"is_host"`
}

type PeerLeft struct {
	Type   string `json:"type"`
	PeerID int    `json:"peer_id"`
}

type RoomClosing struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

type ErrorMsg struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Ready struct {
	Type string `json:"type"`
}

type Ping struct {
	Type string `json:"type"`
	ID   int    `json:"id"`
}

type Pong struct {
	Type string `json:"type"`
	ID   int    `json:"id"`
}

func NewHello(nickname string, peerID int, isHost bool, room *RoomInfo) Hello {
	return Hello{
		Type:    "hello",
		You:     nickname,
		PeerID:  peerID,
		IsHost:  isHost,
		Room:    *room,
	}
}

func NewPeerJoined(peerID int, nickname string, isHost bool) PeerJoined {
	return PeerJoined{
		Type:     "peer_joined",
		PeerID:   peerID,
		Nickname: nickname,
		IsHost:   isHost,
	}
}

func NewPeerLeft(peerID int) PeerLeft {
	return PeerLeft{
		Type:   "peer_left",
		PeerID: peerID,
	}
}

func NewError(code, message string) ErrorMsg {
	return ErrorMsg{
		Type:    "error",
		Code:    code,
		Message: message,
	}
}
