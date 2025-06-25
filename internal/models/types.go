package models

type Player struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	IsActive bool   `json:"isactive"`
}

type Room struct {
	Code        string
	Players     []Player
	ActiveRoles []string
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
	RoomCode string `json:"roomCode"`
}
