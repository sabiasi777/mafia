package models

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

type Player struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	IsActive bool   `json:"isactive"`
}

type Room struct {
	Code        string
	Players     []Player // Players map[string]*Player
	ActiveRoles []string
	CurrentUser string
	Owner       string
}

type Page struct {
	Title string
	Body  []byte
}

type AudioMessage struct {
	MimeType string `json:"mimeType"`
	Audio    string `json:"audio"` // base64
}

type StartRequest struct {
	RoomCode    string `json:"roomCode"`
	CurrentUser string `json:"currentUserName"`
}

type Client struct {
	Conn     *websocket.Conn
	UserName string
}

type SignalingMessage struct {
	Type      string          `json:"type"`
	Sender    string          `json:"sender"`
	Receiver  string          `json:"receiver,omitempty"`
	Content   string          `json:"content,omitempty"`
	Sdp       json.RawMessage `json:"sdp,omitempty"`
	Candidate json.RawMessage `json:"candidate,omitempty"`
	Name      string          `json:"name,omitempty"`
	Players   []Player        `json:"players,omitempty"`
	Me        *Player         `json:"me,omitempty"`
}
