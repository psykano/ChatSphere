package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/christopherjohns/chatsphere/internal/room"
	"github.com/christopherjohns/chatsphere/internal/ws"
)

// Server is the main HTTP server for ChatSphere.
type Server struct {
	addr  string
	mux   *http.ServeMux
	rooms *room.Manager
	hub   *ws.Hub
}

// New creates a new Server listening on addr.
func New(addr string) *Server {
	rm := room.NewManager()
	s := &Server{
		addr:  addr,
		mux:   http.NewServeMux(),
		rooms: rm,
	}
	s.hub = ws.NewHub(func(roomID string, delta int) {
		if r := rm.Get(roomID); r != nil {
			r.AddActiveUsers(delta)
		}
	})
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

	sessions := ws.NewSessionStore(2 * time.Minute)
	wsHandler := ws.NewHandler(s.hub, func(roomID string) bool {
		return s.rooms.Get(roomID) != nil
	}, sessions)
	s.mux.Handle("GET /ws", wsHandler)
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
