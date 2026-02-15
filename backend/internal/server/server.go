package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/christopherjohns/chatsphere/internal/message"
	"github.com/christopherjohns/chatsphere/internal/ratelimit"
	"github.com/christopherjohns/chatsphere/internal/room"
	"github.com/christopherjohns/chatsphere/internal/ws"
)

// Server is the main HTTP server for ChatSphere.
type Server struct {
	addr        string
	mux         *http.ServeMux
	rooms       *room.Manager
	hub         *ws.Hub
	createLimit *ratelimit.IPLimiter
}

// New creates a new Server listening on addr.
func New(addr string) *Server {
	rm := room.NewManager()
	s := &Server{
		addr:        addr,
		mux:         http.NewServeMux(),
		rooms:       rm,
		createLimit: ratelimit.NewIPLimiter(3, time.Hour),
	}
	s.hub = ws.NewHub(func(roomID string, delta int) {
		if r := rm.Get(roomID); r != nil {
			r.AddActiveUsers(delta)
			if delta > 0 {
				r.ClearUserLeft()
			} else if r.ActiveUsers <= 0 {
				r.TouchUserLeft()
			}
		}
	})
	s.hub.SetOnBroadcast(func(roomID string) {
		if r := rm.Get(roomID); r != nil {
			r.TouchMessage()
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
	s.mux.HandleFunc("GET /api/rooms/code/{code}", s.handleGetRoomByCode)
	s.mux.HandleFunc("GET /api/rooms/{id}", s.handleGetRoom)
	s.mux.HandleFunc("POST /api/rooms", s.handleCreateRoom)

	sessions := ws.NewSessionStore(2 * time.Minute)
	messages := message.NewStore(200)
	s.hub.SetMessageStore(messages)
	s.hub.SetSessionStore(sessions)
	wsHandler := ws.NewHandler(s.hub, func(roomID string) string {
		r := s.rooms.Get(roomID)
		if r == nil {
			return "room not found"
		}
		if r.IsFull() {
			return "room is full"
		}
		return ""
	}, sessions, messages)
	s.mux.Handle("GET /ws", wsHandler)

	s.rooms.StartExpiration(room.ExpirationConfig{
		MsgTTL:   2 * time.Hour,
		EmptyTTL: 15 * time.Minute,
		MsgWarn:  5 * time.Minute,
		EmptyWarn: 2 * time.Minute,
		OnExpire: func(roomID string) {
			s.hub.DisconnectRoom(roomID)
			messages.DeleteRoom(roomID)
		},
		OnWarn: func(roomID string, reason room.WarningReason, remaining time.Duration) {
			mins := int(remaining.Minutes())
			if mins < 1 {
				mins = 1
			}
			var content string
			switch reason {
			case room.WarnMsgInactive:
				content = fmt.Sprintf("This room will expire in %d minutes due to inactivity", mins)
			case room.WarnEmpty:
				content = fmt.Sprintf("This room will expire in %d minutes because it is empty", mins)
			}
			b := make([]byte, 16)
			rand.Read(b)
			s.hub.Broadcast(roomID, &message.Message{
				ID:        hex.EncodeToString(b),
				RoomID:    roomID,
				Content:   content,
				Type:      message.TypeSystem,
				CreatedAt: time.Now(),
			})
		},
	})
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

func (s *Server) handleGetRoomByCode(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(strings.TrimSpace(r.PathValue("code")))
	if len(code) != 6 {
		http.Error(w, `{"error":"code must be 6 characters"}`, http.StatusBadRequest)
		return
	}

	rm := s.rooms.GetByCode(code)
	if rm == nil {
		http.Error(w, `{"error":"room not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rm)
}

func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rm := s.rooms.Get(id)
	if rm == nil {
		http.Error(w, `{"error":"room not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rm)
}

type createRoomRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Capacity    int    `json:"capacity"`
	Public      bool   `json:"public"`
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		// First entry is the original client
		if i := strings.IndexByte(fwd, ','); i > 0 {
			return strings.TrimSpace(fwd[:i])
		}
		return strings.TrimSpace(fwd)
	}
	// Strip port from RemoteAddr
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i > 0 {
		return addr[:i]
	}
	return addr
}

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	if !s.createLimit.Allow(clientIP(r)) {
		http.Error(w, `{"error":"rate limit exceeded, max 3 rooms per hour"}`, http.StatusTooManyRequests)
		return
	}

	var req createRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)

	if req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Name) > 100 {
		http.Error(w, `{"error":"name must be 100 characters or less"}`, http.StatusBadRequest)
		return
	}
	if len(req.Description) > 500 {
		http.Error(w, `{"error":"description must be 500 characters or less"}`, http.StatusBadRequest)
		return
	}
	if req.Capacity < 2 || req.Capacity > 100 {
		http.Error(w, `{"error":"capacity must be between 2 and 100"}`, http.StatusBadRequest)
		return
	}

	room := s.rooms.Create(req.Name, req.Description, "", req.Capacity, req.Public)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(room)
}

