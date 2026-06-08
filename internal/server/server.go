package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/joeyb/lpaas-98/internal/lobby"
	"github.com/joeyb/lpaas-98/internal/registry"
)

type Server struct {
	addr     string
	mux      *http.ServeMux
	registry *registry.Registry
	lobby    *lobby.Lobby
}

func New(addr string, reg *registry.Registry) *Server {
	s := &Server{
		addr:     addr,
		mux:      http.NewServeMux(),
		registry: reg,
		lobby:    lobby.New(),
	}

	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("GET /api/games", s.handleListGames)
	s.mux.HandleFunc("GET /api/rooms", s.handleListRooms)
	s.mux.HandleFunc("POST /api/rooms", s.handleCreateRoom)
	s.mux.HandleFunc("POST /api/rooms/{roomID}/join", s.handleJoinRoom)

	return s
}

func (s *Server) ListenAndServe() error {
	addrs := listLANAddresses()
	if len(addrs) > 0 {
		slog.Info("LPaaS 98 listening on:")
		for _, addr := range addrs {
			slog.Info(fmt.Sprintf("  http://%s:9898", addr))
		}
	}

	return http.ListenAndServe(s.addr, s.mux)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
	<title>LPaaS 98</title>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
	<h1>LPaaS 98</h1>
	<p>LAN Party as a Service</p>
	<p>Loading...</p>
</body>
</html>`)
}

func (s *Server) handleListGames(w http.ResponseWriter, r *http.Request) {
	games := s.registry.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

func (s *Server) handleListRooms(w http.ResponseWriter, r *http.Request) {
	rooms := s.lobby.ListRooms()
	w.Header().Set("Content-Type", "application/json")

	roomData := make([]map[string]interface{}, len(rooms))
	for i, room := range rooms {
		roomData[i] = map[string]interface{}{
			"id":          room.ID,
			"game_id":     room.GameID,
			"host_id":     room.HostID,
			"player_count": len(room.Members),
			"max_players": room.MaxPlayers,
		}
	}

	json.NewEncoder(w).Encode(roomData)
}

type createRoomReq struct {
	ClientID   string `json:"client_id"`
	Nickname   string `json:"nickname"`
	GameID     string `json:"game_id"`
	MaxPlayers int    `json:"max_players"`
}

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	var req createRoomReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	_, err := s.lobby.AddClient(req.ClientID, req.Nickname)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	game := s.registry.Get(req.GameID)
	if game == nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	room, err := s.lobby.CreateRoom(req.GameID, req.ClientID, req.MaxPlayers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"room_id": room.ID,
		"game_id": room.GameID,
		"host_id": room.HostID,
	})
}

type joinRoomReq struct {
	ClientID string `json:"client_id"`
	Nickname string `json:"nickname"`
}

func (s *Server) handleJoinRoom(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	var req joinRoomReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	_, err := s.lobby.AddClient(req.ClientID, req.Nickname)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	room, err := s.lobby.JoinRoom(req.ClientID, roomID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"room_id": room.ID,
		"game_id": room.GameID,
		"players": len(room.Members),
	})
}

func listLANAddresses() []string {
	var result []string

	ifaces, err := net.Interfaces()
	if err != nil {
		return result
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipnet.IP.To4()
			if ip == nil {
				continue
			}

			result = append(result, ip.String())
		}
	}

	return result
}
