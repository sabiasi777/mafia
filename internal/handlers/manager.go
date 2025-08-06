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

func (rm *RoomManager) getLivePlayers(roomCode string) []models.Player {
	livePlayers := []models.Player{}

	if room, ok := rm.Rooms[roomCode]; ok {
		for _, player := range room.Players {
			if player.IsActive {
				livePlayers = append(livePlayers, player)
			}
		}
	}

	return livePlayers
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

	allActivePlayers := rm.getLivePlayers(roomCode)

	for _, player := range room.Players {
		if !player.IsActive {
			continue
		}

		validTargets := []models.Player{}
		switch player.Role {
		case "Mafia":
			for _, p := range allActivePlayers {
				if p.Role != "Mafia" {
					validTargets = append(validTargets, p)
				}
			}
		case "Doctor", "Detective":
			for _, p := range allActivePlayers {
				if p.Name != player.Name {
					validTargets = append(validTargets, p)
				}
			}
		}
		if conn, ok := rm.Connections[roomCode][player.Name]; ok {
			phaseChangeMsg := models.SignalingMessage{
				Type:         "phase-change",
				Phase:        "Night",
				ValidTargets: validTargets,
			}
			payload, _ := json.Marshal(phaseChangeMsg)
			conn.WriteMessage(websocket.TextMessage, payload)
		}
	}
}

func (rm *RoomManager) sendDetectiveResult(roomCode, senderName, target string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	room, roomExists := rm.Rooms[roomCode]
	connections, connectionsExist := rm.Connections[roomCode]

	if !roomExists || !connectionsExist {
		return
	}

	var investigationResult string = "Not Mafia"

	for _, player := range room.Players {
		if player.Name == target {
			if player.Role == "Mafia" {
				investigationResult = "Mafia"
			}
			break
		}
	}

	detectiveConn, ok := connections[senderName]
	if !ok {
		return
	}

	resultMsg := models.SignalingMessage{
		Type:   "detective-result",
		Result: investigationResult,
		Target: target,
	}
	payload, err := json.Marshal(resultMsg)
	if err != nil {
		fmt.Printf("Error marshaling detective result: %v\n", err)
		return
	}

	fmt.Printf("Sending private result to detective %s\n", senderName)
	detectiveConn.WriteMessage(websocket.TextMessage, payload)
}

func (rm *RoomManager) areNightActionsComplete(room *models.Room) bool {
	requiredRoles := []string{"Mafia", "Doctor", "Detective"}

	for _, role := range requiredRoles {
		isRoleInGame := false
		for _, player := range room.Players {
			if player.Role == role && player.IsActive {
				isRoleInGame = true
				break
			}
		}

		if isRoleInGame {
			if _, ok := room.NightActionsTaken[role]; !ok {
				return false
			}
		}
	}

	return true
}

func (rm *RoomManager) startDayPhase(roomCode string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, roomExists := rm.Rooms[roomCode]
	connections, connectionsExist := rm.Connections[roomCode]

	if !roomExists || !connectionsExist {
		return
	}
	eliminatedPlayerName := ""
	var resultMessage string = "No one died."

	room.GamePhase = "Day"
	if room.MafiaTarget != "" && room.MafiaTarget != room.DoctorSave {
		eliminatedPlayerName = room.MafiaTarget
		resultMessage = fmt.Sprintf("%s was eliminated last night.", eliminatedPlayerName)
	}

	if eliminatedPlayerName != "" {
		for i := range room.Players {
			if room.Players[i].Name == eliminatedPlayerName {
				room.Players[i].IsActive = false
				break
			}
		}
	}

	room.SpeakersThisRound = make(map[string]bool)
	room.CurrentSpeakerIndex = rm.findFirstActivePlayer(room)
	firstSpeaker := room.Players[room.CurrentSpeakerIndex]

	dayPhaseMsg := models.SignalingMessage{
		Type:        "phase-change",
		Phase:       "Day",
		Result:      resultMessage,
		Players:     rm.getCurrentPlayers(roomCode),
		SpeakerName: firstSpeaker.Name,
	}
	payload, err := json.Marshal(dayPhaseMsg)
	if err != nil {
		fmt.Printf("Error marshaling day-phase message: %v\n", err)
		return
	}

	fmt.Println("Starting day phase, broadcasting results.")
	fmt.Println("These players are still in the game:", room.Players)
	for _, conn := range connections {
		conn.WriteMessage(websocket.TextMessage, payload)
	}

	go rm.checkWinCondition(roomCode)
}

func (rm *RoomManager) checkWinCondition(roomCode string) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.Rooms[roomCode]
	if !ok {
		return false
	}

	if room.GamePhase == "" {
		return false
	}

	mafiaCount := 0
	townCount := 0

	for _, player := range room.Players {
		if player.IsActive {
			if player.Role == "Mafia" {
				mafiaCount++
			} else {
				townCount++
			}
		}
	}

	winner := ""
	if mafiaCount == 0 {
		winner = "Town"
	} else if mafiaCount >= townCount {
		winner = "Mafia"
	}

	if winner != "" {
		fmt.Printf("Game over in room %s. Winner: %s\n", roomCode, winner)

		if room.TurnTimer != nil {
			fmt.Println("room.TurnTimer.Stop()")
			room.TurnTimer.Stop()
		}
		fmt.Println("broadcasting Game Over")
		rm.broadcastGameOver(roomCode, winner)
		rm.resetRoomForNewGame(room)
		return true
	}
	return false
}

func (rm *RoomManager) broadcastGameOver(roomCode string, winner string) {
	connections, ok := rm.Connections[roomCode]
	if !ok {
		return
	}

	finalPlayerList := rm.Rooms[roomCode].Players

	gameOverMsg := models.SignalingMessage{
		Type:    "game-over",
		Winner:  winner,
		Players: finalPlayerList,
	}
	payload, _ := json.Marshal(gameOverMsg)

	for _, conn := range connections {
		fmt.Println("Sending game-over message")
		conn.WriteMessage(websocket.TextMessage, payload)
	}
}

func (rm *RoomManager) resetRoomForNewGame(room *models.Room) {
	room.GamePhase = "Lobby"
	room.CurrentSpeakerIndex = 0
	room.MafiaTarget = ""
	room.DoctorSave = ""
	room.DetectiveCheck = ""
	room.NightActionsTaken = nil

	for i := range room.Players {
		room.Players[i].Role = ""
		room.Players[i].IsActive = true
	}
}

func (rm *RoomManager) findNextActivePlayer(room *models.Room) int {
	fmt.Println("function findnextactiveplayer")
	currentIndex := room.CurrentSpeakerIndex
	numPlayers := len(room.Players)

	if numPlayers == 0 {
		return -1
	}

	for i := 1; i <= numPlayers; i++ {
		nextIndex := (currentIndex + i) % numPlayers

		if room.Players[nextIndex].IsActive {
			fmt.Println("next active player index", nextIndex)
			return nextIndex
		}
	}

	return -1
}

func (rm *RoomManager) findFirstActivePlayer(room *models.Room) int {
	for i, player := range room.Players {
		if player.IsActive {
			return i
		}
	}
	return -1
}

func (rm *RoomManager) isSpeakingRoundOver(room *models.Room) bool {
	fmt.Println("function isspeakingroundover")
	activePlayerCount := 0
	for _, player := range room.Players {
		if player.IsActive {
			activePlayerCount++
		}
	}

	speakersCount := len(room.SpeakersThisRound)
	return speakersCount >= activePlayerCount
}
