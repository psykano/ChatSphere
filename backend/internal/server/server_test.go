package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	srv := New(":0")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", body["status"])
	}
}

func TestListRoomsEmpty(t *testing.T) {
	srv := New(":0")

	req := httptest.NewRequest(http.MethodGet, "/api/rooms", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var rooms []interface{}
	if err := json.NewDecoder(w.Body).Decode(&rooms); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(rooms) != 0 {
		t.Errorf("expected empty room list, got %d rooms", len(rooms))
	}
}

func TestListRoomsWithData(t *testing.T) {
	srv := New(":0")
	srv.rooms.Create("Room A", "desc", "user1", 50, true)
	srv.rooms.Create("Room B", "", "user2", 20, true)
	srv.rooms.Create("Private Room", "", "user3", 10, false)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var rooms []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&rooms); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(rooms) != 2 {
		t.Errorf("expected 2 public rooms, got %d", len(rooms))
	}
}

func postJSON(srv *Server, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/rooms", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)
	return w
}

func TestCreateRoomPublic(t *testing.T) {
	srv := New(":0")

	w := postJSON(srv, `{"name":"General","description":"Main chat","capacity":50,"public":true}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	var room map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&room); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if room["name"] != "General" {
		t.Errorf("expected name 'General', got %v", room["name"])
	}
	if room["description"] != "Main chat" {
		t.Errorf("expected description 'Main chat', got %v", room["description"])
	}
	if room["capacity"] != float64(50) {
		t.Errorf("expected capacity 50, got %v", room["capacity"])
	}
	if room["public"] != true {
		t.Errorf("expected public true, got %v", room["public"])
	}
	if room["id"] == nil || room["id"] == "" {
		t.Error("expected non-empty id")
	}
	if room["code"] != nil {
		t.Errorf("expected no code for public room, got %v", room["code"])
	}
}

func TestCreateRoomPrivate(t *testing.T) {
	srv := New(":0")

	w := postJSON(srv, `{"name":"Secret","capacity":10,"public":false}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	var room map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&room); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if room["public"] != false {
		t.Errorf("expected public false, got %v", room["public"])
	}
	code, ok := room["code"].(string)
	if !ok || len(code) != 6 {
		t.Errorf("expected 6-char code for private room, got %v", room["code"])
	}
}

func TestCreateRoomAppearsInList(t *testing.T) {
	srv := New(":0")

	postJSON(srv, `{"name":"Listed Room","capacity":20,"public":true}`)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	var rooms []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&rooms); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(rooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(rooms))
	}
	if rooms[0]["name"] != "Listed Room" {
		t.Errorf("expected name 'Listed Room', got %v", rooms[0]["name"])
	}
}

func TestCreateRoomMissingName(t *testing.T) {
	srv := New(":0")

	w := postJSON(srv, `{"capacity":10,"public":true}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestCreateRoomWhitespaceName(t *testing.T) {
	srv := New(":0")

	w := postJSON(srv, `{"name":"   ","capacity":10,"public":true}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestCreateRoomNameTooLong(t *testing.T) {
	srv := New(":0")

	longName := strings.Repeat("a", 101)
	w := postJSON(srv, `{"name":"`+longName+`","capacity":10,"public":true}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestCreateRoomDescriptionTooLong(t *testing.T) {
	srv := New(":0")

	longDesc := strings.Repeat("a", 501)
	w := postJSON(srv, `{"name":"Room","description":"`+longDesc+`","capacity":10,"public":true}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestCreateRoomCapacityTooLow(t *testing.T) {
	srv := New(":0")

	w := postJSON(srv, `{"name":"Room","capacity":1,"public":true}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestCreateRoomCapacityTooHigh(t *testing.T) {
	srv := New(":0")

	w := postJSON(srv, `{"name":"Room","capacity":101,"public":true}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestCreateRoomCapacityZero(t *testing.T) {
	srv := New(":0")

	w := postJSON(srv, `{"name":"Room","capacity":0,"public":true}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestCreateRoomInvalidJSON(t *testing.T) {
	srv := New(":0")

	w := postJSON(srv, `not json`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestCreateRoomTrimWhitespace(t *testing.T) {
	srv := New(":0")

	w := postJSON(srv, `{"name":"  Trimmed  ","description":"  Desc  ","capacity":10,"public":true}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	var room map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&room); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if room["name"] != "Trimmed" {
		t.Errorf("expected trimmed name 'Trimmed', got %v", room["name"])
	}
	if room["description"] != "Desc" {
		t.Errorf("expected trimmed description 'Desc', got %v", room["description"])
	}
}
