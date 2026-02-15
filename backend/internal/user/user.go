package user

import "time"

// User represents an anonymous user session.
type User struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	Username  string    `json:"username,omitempty"`
	IP        string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}
