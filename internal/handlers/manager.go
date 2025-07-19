package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sabiasi777/mafia/internal/models"
)

type RoomManager struct {
	Rooms       (map[string]*models.Room)
	Tmpl        *template.Template
	Connections map[string]map[string]*websocket.Conn
	mu          sync.Mutex
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms:       make(map[string]*models.Room),
		Connections: make(map[string]map[string]*websocket.Conn),
		Tmpl:        template.Must(template.ParseGlob("templates/*.html")),
	}
}

func (rm *RoomManager) startNightPhase(roomCode string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.Rooms[roomCode]
	if !ok {
		return
	}

	fmt.Printf("Starting night phase for room %s\n", roomCode)
	room.GamePhase = "Night"
	room.MafiaTarget = ""
	room.DoctorSave = ""
	room.DetectiveCheck = ""
	room.NightActionsTaken = make(map[string]bool)

	phaseChangeMsg := models.SignalingMessage{
		Type:    "phase-change",
		Phase:   "Night",
		Players: rm.getCurrentPlayers(roomCode),
	}
	payload, _ := json.Marshal(phaseChangeMsg)

	for _, conn := range rm.Connections[roomCode] {
		conn.WriteMessage(websocket.TextMessage, payload)
	}
}
