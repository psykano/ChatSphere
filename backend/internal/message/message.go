package message

import "time"

// Type represents the kind of message.
type Type string

const (
	TypeChat   Type = "chat"
	TypeSystem Type = "system"
)

// Message represents a chat message.
type Message struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id,omitempty"`
	Username  string    `json:"username,omitempty"`
	Content   string    `json:"content"`
	Type      Type      `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}
