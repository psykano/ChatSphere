package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/christopherjohns/chatsphere/internal/ws"
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

func TestListRoomsSortedByActiveUsers(t *testing.T) {
	srv := New(":0")
	r1 := srv.rooms.Create("Low Activity", "", "user1", 50, true)
	r2 := srv.rooms.Create("High Activity", "", "user1", 50, true)
	r3 := srv.rooms.Create("Mid Activity", "", "user1", 50, true)

	r1.AddActiveUsers(2)
	r2.AddActiveUsers(15)
	r3.AddActiveUsers(7)

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
	if len(rooms) != 3 {
		t.Fatalf("expected 3 rooms, got %d", len(rooms))
	}

	// Verify sorted by active_users descending
	if rooms[0]["name"] != "High Activity" {
		t.Errorf("expected first room 'High Activity', got %v", rooms[0]["name"])
	}
	if rooms[0]["active_users"] != float64(15) {
		t.Errorf("expected first room active_users=15, got %v", rooms[0]["active_users"])
	}
	if rooms[1]["name"] != "Mid Activity" {
		t.Errorf("expected second room 'Mid Activity', got %v", rooms[1]["name"])
	}
	if rooms[1]["active_users"] != float64(7) {
		t.Errorf("expected second room active_users=7, got %v", rooms[1]["active_users"])
	}
	if rooms[2]["name"] != "Low Activity" {
		t.Errorf("expected third room 'Low Activity', got %v", rooms[2]["name"])
	}
	if rooms[2]["active_users"] != float64(2) {
		t.Errorf("expected third room active_users=2, got %v", rooms[2]["active_users"])
	}
}

func TestListRoomsExcludesPrivateRooms(t *testing.T) {
	srv := New(":0")
	srv.rooms.Create("Public Room", "", "user1", 50, true)
	priv := srv.rooms.Create("Private Room", "", "user1", 10, false)
	priv.AddActiveUsers(100) // High activity, but should still be excluded

	req := httptest.NewRequest(http.MethodGet, "/api/rooms", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	var rooms []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&rooms)

	if len(rooms) != 1 {
		t.Fatalf("expected 1 public room, got %d", len(rooms))
	}
	if rooms[0]["name"] != "Public Room" {
		t.Errorf("expected 'Public Room', got %v", rooms[0]["name"])
	}
}

func TestListRoomsResponseFields(t *testing.T) {
	srv := New(":0")
	r := srv.rooms.Create("Test Room", "A description", "user1", 50, true)
	r.AddActiveUsers(3)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	var rooms []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&rooms)

	if len(rooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(rooms))
	}

	room := rooms[0]
	if room["id"] == nil || room["id"] == "" {
		t.Error("expected non-empty id")
	}
	if room["name"] != "Test Room" {
		t.Errorf("expected name 'Test Room', got %v", room["name"])
	}
	if room["description"] != "A description" {
		t.Errorf("expected description 'A description', got %v", room["description"])
	}
	if room["capacity"] != float64(50) {
		t.Errorf("expected capacity 50, got %v", room["capacity"])
	}
	if room["public"] != true {
		t.Errorf("expected public true, got %v", room["public"])
	}
	if room["active_users"] != float64(3) {
		t.Errorf("expected active_users 3, got %v", room["active_users"])
	}
}

func postJSON(srv *Server, body string) *httptest.ResponseRecorder {
	return postJSONFrom(srv, body, "")
}

func postJSONFrom(srv *Server, body, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/rooms", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
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

func TestGetRoomByID(t *testing.T) {
	srv := New(":0")

	w := postJSON(srv, `{"name":"Test Room","description":"A room","capacity":10,"public":true}`)
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	id := created["id"].(string)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms/"+id, nil)
	w = httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var room map[string]interface{}
	json.NewDecoder(w.Body).Decode(&room)
	if room["id"] != id {
		t.Errorf("expected room ID %v, got %v", id, room["id"])
	}
	if room["name"] != "Test Room" {
		t.Errorf("expected name 'Test Room', got %v", room["name"])
	}
	if room["capacity"] != float64(10) {
		t.Errorf("expected capacity 10, got %v", room["capacity"])
	}
}

func TestGetRoomByIDNotFound(t *testing.T) {
	srv := New(":0")

	req := httptest.NewRequest(http.MethodGet, "/api/rooms/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestGetRoomByIDIncludesActiveUsers(t *testing.T) {
	srv := New(":0")
	r := srv.rooms.Create("Active Room", "", "user1", 50, true)
	r.AddActiveUsers(5)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms/"+r.ID, nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var room map[string]interface{}
	json.NewDecoder(w.Body).Decode(&room)
	if room["active_users"] != float64(5) {
		t.Errorf("expected active_users 5, got %v", room["active_users"])
	}
}

func TestGetRoomByCode(t *testing.T) {
	srv := New(":0")

	w := postJSON(srv, `{"name":"Secret","capacity":10,"public":false}`)
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	code := created["code"].(string)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms/code/"+code, nil)
	w = httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var room map[string]interface{}
	json.NewDecoder(w.Body).Decode(&room)
	if room["id"] != created["id"] {
		t.Errorf("expected room ID %v, got %v", created["id"], room["id"])
	}
}

func TestGetRoomByCodeNotFound(t *testing.T) {
	srv := New(":0")

	req := httptest.NewRequest(http.MethodGet, "/api/rooms/code/ZZZZZZ", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestGetRoomByCodeInvalidLength(t *testing.T) {
	srv := New(":0")

	req := httptest.NewRequest(http.MethodGet, "/api/rooms/code/AB", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestGetRoomByCodeCaseInsensitive(t *testing.T) {
	srv := New(":0")

	w := postJSON(srv, `{"name":"Secret","capacity":10,"public":false}`)
	var created map[string]interface{}
	json.NewDecoder(w.Body).Decode(&created)
	code := created["code"].(string)
	lowerCode := strings.ToLower(code)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms/code/"+lowerCode, nil)
	w = httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for lowercase code, got %d", w.Code)
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

func TestCreateRoomRateLimitEnforced(t *testing.T) {
	srv := New(":0")
	ip := "10.0.0.1:12345"
	body := `{"name":"Room","capacity":10,"public":true}`

	for i := 0; i < 3; i++ {
		w := postJSONFrom(srv, body, ip)
		if w.Code != http.StatusCreated {
			t.Fatalf("request %d: expected 201, got %d", i+1, w.Code)
		}
	}

	w := postJSONFrom(srv, body, ip)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request: expected 429, got %d", w.Code)
	}
}

func TestCreateRoomRateLimitPerIP(t *testing.T) {
	srv := New(":0")
	body := `{"name":"Room","capacity":10,"public":true}`

	// Exhaust limit for IP A
	for i := 0; i < 3; i++ {
		postJSONFrom(srv, body, "10.0.0.1:12345")
	}

	// IP B should still be allowed
	w := postJSONFrom(srv, body, "10.0.0.2:12345")
	if w.Code != http.StatusCreated {
		t.Fatalf("different IP should be allowed, got %d", w.Code)
	}
}

func TestSessionEndpointCreatesSession(t *testing.T) {
	srv := New(":0")

	req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["token"] == nil || body["token"] == "" {
		t.Error("expected non-empty token")
	}
	if body["user_id"] == nil || body["user_id"] == "" {
		t.Error("expected non-empty user_id")
	}

	// Verify cookie is set.
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			found = true
			if c.Value != body["token"] {
				t.Errorf("cookie value %q != response token %q", c.Value, body["token"])
			}
			if !c.HttpOnly {
				t.Error("expected HttpOnly cookie")
			}
		}
	}
	if !found {
		t.Error("expected session cookie to be set")
	}
}

func TestSessionEndpointReturnsSameSession(t *testing.T) {
	srv := New(":0")

	// First request creates a session.
	req1 := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	w1 := httptest.NewRecorder()
	srv.mux.ServeHTTP(w1, req1)

	var first map[string]interface{}
	json.NewDecoder(w1.Body).Decode(&first)
	token := first["token"].(string)
	userID := first["user_id"].(string)

	// Second request with cookie returns same session.
	req2 := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	req2.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	w2 := httptest.NewRecorder()
	srv.mux.ServeHTTP(w2, req2)

	var second map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&second)

	if second["token"] != token {
		t.Errorf("expected same token %q, got %q", token, second["token"])
	}
	if second["user_id"] != userID {
		t.Errorf("expected same user_id %q, got %q", userID, second["user_id"])
	}
}

func TestSessionEndpointInvalidCookieCreatesNew(t *testing.T) {
	srv := New(":0")

	req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "bogus"})
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	if body["token"] == nil || body["token"] == "" {
		t.Error("expected a new session to be created")
	}
	if body["token"] == "bogus" {
		t.Error("expected a different token than the bogus one")
	}
}

func TestCreateRoomRateLimitXForwardedFor(t *testing.T) {
	srv := New(":0")
	body := `{"name":"Room","capacity":10,"public":true}`

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/rooms", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18")
		w := httptest.NewRecorder()
		srv.mux.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("request %d: expected 201, got %d", i+1, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/rooms", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request via X-Forwarded-For should be 429, got %d", w.Code)
	}
}

func TestRoomUsersEndpointNotFound(t *testing.T) {
	srv := New(":0")

	req := httptest.NewRequest(http.MethodGet, "/api/room-users/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRoomUsersEndpointEmpty(t *testing.T) {
	srv := New(":0")
	rm := srv.rooms.Create("Test Room", "", "creator", 10, true)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/room-users/%s", rm.ID), nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var users []ws.RoomUser
	if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}
