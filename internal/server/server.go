package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/joeyb/lpaas-98/internal/registry"
)

type Server struct {
	addr     string
	mux      *http.ServeMux
	registry *registry.Registry
}

func New(addr string, reg *registry.Registry) *Server {
	s := &Server{
		addr:     addr,
		mux:      http.NewServeMux(),
		registry: reg,
	}

	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("GET /api/games", s.handleListGames)

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
