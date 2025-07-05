package handlers

import (
	"html/template"
	"sync"

	"github.com/sabiasi777/mafia/internal/models"
	"github.com/gorilla/websocket"
)

type RoomManager struct {
	Rooms (map[string]*models.Room)
	Tmpl  *template.Template
	Connections map[string]map[string]*websocket.Conn
	mu sync.Mutex
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms: make(map[string]*models.Room),
		Connections: make(map[string]map[string]*websocket.Conn),
		Tmpl: template.Must(template.ParseGlob("templates/*.html")),
	}
}
