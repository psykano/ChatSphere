package message

import "time"

// Type represents the kind of message.
type Type string

const (
	TypeChat   Type = "chat"
	TypeSystem Type = "system"
	TypeTyping Type = "typing"
)

// Action describes what triggered a system message.
type Action string

const (
	ActionJoin       Action = "join"
	ActionLeave      Action = "leave"
	ActionKick       Action = "kick"
	ActionBan        Action = "ban"
	ActionMute       Action = "mute"
	ActionExpiration   Action = "expiration"
	ActionSetUsername  Action = "set_username"
)

// Message represents a chat message.
type Message struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id,omitempty"`
	Username  string    `json:"username,omitempty"`
	Content   string    `json:"content"`
	Type      Type      `json:"type"`
	Action    Action    `json:"action,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
