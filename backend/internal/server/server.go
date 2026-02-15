package server

import (
	"encoding/json"
	"net/http"

	"github.com/christopherjohns/chatsphere/internal/room"
)

// Server is the main HTTP server for ChatSphere.
type Server struct {
	addr    string
	mux     *http.ServeMux
	rooms   *room.Manager
}

// New creates a new Server listening on addr.
func New(addr string) *Server {
	s := &Server{
		addr:  addr,
		mux:   http.NewServeMux(),
		rooms: room.NewManager(),
	}
	s.routes()
	return s
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	return http.ListenAndServe(s.addr, s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /api/rooms", s.handleListRooms)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleListRooms(w http.ResponseWriter, r *http.Request) {
	rooms := s.rooms.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rooms)
}
