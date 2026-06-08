package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/joeyb/lpaas-98/internal/lobby"
	"github.com/joeyb/lpaas-98/internal/registry"
	"github.com/joeyb/lpaas-98/internal/vlan"
	"golang.org/x/net/websocket"
)

type Server struct {
	addr     string
	mux      *http.ServeMux
	registry *registry.Registry
	lobby    *lobby.Lobby
	relay    *vlan.Relay
	gamesDir string
}

func New(addr string, reg *registry.Registry, gamesDir string) *Server {
	s := &Server{
		addr:     addr,
		mux:      http.NewServeMux(),
		registry: reg,
		lobby:    lobby.New(),
		relay:    vlan.NewRelay(),
		gamesDir: gamesDir,
	}

	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("GET /api/games", s.handleListGames)
	s.mux.HandleFunc("GET /api/rooms", s.handleListRooms)
	s.mux.HandleFunc("POST /api/rooms", s.handleCreateRoom)
	s.mux.HandleFunc("POST /api/rooms/{roomID}/join", s.handleJoinRoom)
	s.mux.HandleFunc("GET /game.html", s.handleGamePage)
	s.mux.HandleFunc("GET /loader.js", s.handleLoaderJS)
	s.mux.HandleFunc("GET /game/{gameID}/{file}", s.handleGameFile)
	s.mux.Handle("GET /ws/room/{roomID}", websocket.Handler(s.relay.ServeRoom))

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

	// Also create room in relay for WebSocket connections
	if _, err := s.relay.CreateRoom(room.ID, game.Game.ID, game.Game.NetworkModel, game.Game.MaxPlayers); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func (s *Server) handleGamePage(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room")
	gameID := r.URL.Query().Get("game")

	if roomID == "" || gameID == "" {
		http.Error(w, "Missing room or game parameter", http.StatusBadRequest)
		return
	}

	room := s.lobby.GetRoom(roomID)
	if room == nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	game := s.registry.Get(gameID)
	if game == nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>LPaaS 98 - ` + game.Game.Name + `</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: Arial, sans-serif; background-color: #000; color: #fff; overflow: hidden; }
    #game-container { width: 100vw; height: 100vh; display: flex; flex-direction: column; position: relative; }
    canvas { display: block; width: 100%; flex: 1; background-color: #000; }
    #hud { background-color: #222; padding: 10px; font-size: 12px; font-family: monospace; border-top: 1px solid #555; }
    .hud-row { display: flex; justify-content: space-between; margin: 2px 0; }
    .hud-label { color: #aaa; }
    .hud-value { color: #0f0; }
    #loading { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); text-align: center; background-color: rgba(0,0,0,0.8); padding: 30px; border-radius: 5px; z-index: 100; }
    .spinner { border: 4px solid #444; border-top: 4px solid #0f0; border-radius: 50%; width: 40px; height: 40px; animation: spin 1s linear infinite; margin: 0 auto 20px; }
    @keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }
    #debug-info { position: absolute; top: 10px; left: 10px; color: #0f0; font-family: monospace; font-size: 11px; z-index: 50; max-width: 300px; }
  </style>
</head>
<body>
  <div id="game-container">
    <canvas id="game-canvas"></canvas>
    <div id="hud">
      <div class="hud-row"><span class="hud-label">Room:</span><span class="hud-value" id="hud-room">` + roomID + `</span></div>
      <div class="hud-row"><span class="hud-label">Status:</span><span class="hud-value" id="hud-status">Connecting...</span></div>
    </div>
  </div>
  <div id="loading">
    <div class="spinner"></div>
    <p>Initializing game...</p>
  </div>
  <div id="debug-info"></div>
  <script type="module">
    const debug = document.getElementById('debug-info');
    function log(msg) {
      console.log(msg);
      debug.innerHTML += msg + '<br>';
    }

    // Define GameLoader inline (don't load it separately)
    class GameLoader {
      constructor(gameId, roomId) {
        this.gameId = gameId;
        this.roomId = roomId;
        this.ws = null;
        this.peerId = null;
        this.isHost = false;
        this.peers = new Map();
      }

      async init() {
        return new Promise((resolve, reject) => {
          const wsUrl = (window.location.protocol === 'https:' ? 'wss:' : 'ws:') + '//' +
            window.location.host + '/ws/room/' + this.roomId + '?nickname=' +
            encodeURIComponent(localStorage.nickname || 'Player');

          this.ws = new WebSocket(wsUrl);

          this.ws.onopen = () => {
            console.log('Connected to relay');
            resolve();
          };

          this.ws.onmessage = (event) => {
            if (typeof event.data === 'string') {
              this.handleControl(JSON.parse(event.data));
            } else {
              this.handleGamePacket(new Uint8Array(event.data));
            }
          };

          this.ws.onerror = (err) => {
            console.error('WebSocket error:', err);
            reject(err);
          };

          this.ws.onclose = () => {
            console.log('Disconnected from relay');
          };
        });
      }

      handleControl(msg) {
        switch (msg.type) {
          case 'hello':
            this.peerId = msg.peer_id;
            this.isHost = msg.is_host;
            log('Joined as peer ' + this.peerId + ' (host: ' + this.isHost + ')');
            this.sendReady();
            break;
          case 'peer_joined':
            console.log('Peer ' + msg.peer_id + ' joined: ' + msg.nickname);
            this.peers.set(msg.peer_id, msg);
            break;
          case 'peer_left':
            console.log('Peer ' + msg.peer_id + ' left');
            this.peers.delete(msg.peer_id);
            break;
        }
      }

      handleGamePacket(data) {
        console.log('Game packet: ' + data.length + ' bytes');
      }

      sendGamePacket(data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
          if (this.isHost) {
            this.ws.send(new Uint8Array([0xFF, ...data]));
          } else {
            this.ws.send(new Uint8Array([0x00, ...data]));
          }
        }
      }

      sendReady() {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
          this.ws.send(JSON.stringify({ type: 'ready' }));
        }
      }
    }

    (async () => {
      try {
        const canvas = document.getElementById('game-canvas');
        canvas.width = window.innerWidth;
        canvas.height = window.innerHeight - 50;
        log('Canvas: ' + canvas.width + 'x' + canvas.height);

        const gl = canvas.getContext('webgl') || canvas.getContext('webgl2');
        log('WebGL: ' + (gl ? 'OK' : 'Not supported'));

        log('Connecting to relay...');
        const loader = new GameLoader('` + gameID + `', '` + roomID + `');
        window.gameLoader = loader;

        try {
          await loader.init();
          log('Relay connected');
        } catch(e) {
          log('Relay error: ' + e.message);
        }

        log('Loading game module...');
        const ioquake3Factory = (await import('/game/` + gameID + `/` + game.Game.JSLoader + `')).default;
        log('Factory loaded');

        log('Initializing engine...');
        const gameModule = await ioquake3Factory({
          canvas: canvas,
          locateFile: (file) => {
            if (file.endsWith('.wasm')) {
              return '/game/` + gameID + `/` + game.Game.WASM + `';
            }
            return '/game/` + gameID + `/' + file;
          }
        });

        log('Engine ready!');
        window.gameModule = gameModule;

        // Start a render loop
        let frameCount = 0;
        function renderFrame() {
          frameCount++;
          if (frameCount % 60 === 0) {
            log('Frames: ' + frameCount);
          }
          requestAnimationFrame(renderFrame);
        }
        renderFrame();
        log('Render loop started');

        document.getElementById('loading').style.display = 'none';
      } catch (err) {
        console.error('Failed to initialize game:', err);
        log('ERROR: ' + err.message);
        document.getElementById('loading').innerHTML = '<p>Error: ' + err.message + '</p>';
        document.getElementById('hud-status').textContent = 'Failed to load';
      }
    })();
  </script>
  <script src="/loader.js?room=` + roomID + `&game=` + gameID + `"></script>
</body>
</html>`))
}

func (s *Server) handleLoaderJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write([]byte(`
class GameLoader {
  constructor(gameId, roomId) {
    this.gameId = gameId;
    this.roomId = roomId;
    this.ws = null;
    this.peerId = null;
    this.isHost = false;
    this.peers = new Map();
  }

  async init() {
    return new Promise((resolve, reject) => {
      const wsUrl = (window.location.protocol === 'https:' ? 'wss:' : 'ws:') + '//' + window.location.host + '/ws/room/' + this.roomId + '?nickname=' + encodeURIComponent(localStorage.nickname || 'Player');
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log('Connected to relay');
        resolve();
      };

      this.ws.onmessage = (event) => {
        if (typeof event.data === 'string') {
          this.handleControl(JSON.parse(event.data));
        } else {
          this.handleGamePacket(new Uint8Array(event.data));
        }
      };

      this.ws.onerror = (err) => {
        console.error('WebSocket error:', err);
        reject(err);
      };

      this.ws.onclose = () => {
        console.log('Disconnected from relay');
      };
    });
  }

  handleControl(msg) {
    switch (msg.type) {
      case 'hello':
        this.peerId = msg.peer_id;
        this.isHost = msg.is_host;
        document.getElementById('hud-status').textContent = 'Joined (Peer ' + this.peerId + ')';
        console.log('Joined as peer ' + this.peerId + ' (host: ' + this.isHost + ')');
        this.sendReady();
        setTimeout(() => {
          document.getElementById('loading').style.display = 'none';
        }, 1000);
        break;
      case 'peer_joined':
        console.log('Peer ' + msg.peer_id + ' joined: ' + msg.nickname);
        this.peers.set(msg.peer_id, msg);
        break;
      case 'peer_left':
        console.log('Peer ' + msg.peer_id + ' left');
        this.peers.delete(msg.peer_id);
        break;
    }
  }

  handleGamePacket(data) {
    // Game packet received from relay
    console.log('Received game packet, size: ' + data.length);
  }

  sendGamePacket(data) {
    if (this.isHost) {
      this.ws.send(new Uint8Array([0xFF, ...data]));
    } else {
      this.ws.send(new Uint8Array([0x00, ...data]));
    }
  }

  sendReady() {
    this.ws.send(JSON.stringify({ type: 'ready' }));
  }
}

const params = new URLSearchParams(window.location.search);
const roomId = params.get('room');
const gameId = params.get('game');

const loader = new GameLoader(gameId, roomId);
window.gameLoader = loader;

loader.init().catch(err => {
  console.error('Failed to initialize:', err);
  document.getElementById('hud-status').textContent = 'Connection failed';
});
`))
}

func (s *Server) handleRoom(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")

	s.relay.Mu.RLock()
	room := s.relay.GetRoom(roomID)
	s.relay.Mu.RUnlock()

	if room == nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           room.ID,
		"game_id":      room.GameID,
		"network_model": room.NetworkModel,
		"player_count": len(room.Peers()),
	})
}

func (s *Server) handleGameFile(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("gameID")
	file := r.PathValue("file")

	// Prevent directory traversal
	if strings.Contains(file, "..") || strings.HasPrefix(file, "/") {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	// Verify game exists
	game := s.registry.Get(gameID)
	if game == nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	filePath := filepath.Join(s.gamesDir, gameID, file)
	http.ServeFile(w, r, filePath)
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
