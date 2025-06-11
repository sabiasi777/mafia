package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

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

var rooms = make(map[string]*Room)
var tmpl *template.Template
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var roomsConnections = make(map[string]map[*websocket.Conn]bool)
var mu sync.Mutex

func handleChat(w http.ResponseWriter, r *http.Request) {
	roomCode := r.URL.Query().Get("room")
	if roomCode == "" {
		http.Error(w, "Missing room code", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Websocket upgrade error:", err)
		return
	}

	mu.Lock()
	if roomsConnections[roomCode] == nil {
		roomsConnections[roomCode] = make(map[*websocket.Conn]bool)
	}
	roomsConnections[roomCode][conn] = true
	mu.Unlock()

	defer func() {
		mu.Lock()
		if clients, exists := roomsConnections[roomCode]; exists {
			delete(clients, conn)
			if len(clients) == 0 {
				delete(roomsConnections, roomCode)
			}
		}
		mu.Unlock()
		conn.Close()
	}()

	handleConnection(conn, roomCode)
}

func handleConnection(conn *websocket.Conn, roomCode string) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Read error:", err)
			break
		}

		mu.Lock()
		for client := range roomsConnections[roomCode] {
			if err := client.WriteMessage(websocket.TextMessage, msg); err != nil {
				fmt.Println("Error writing message:", err)
			}
		}
		mu.Unlock()
	}
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Joinhandler")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Println("join")

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	roomcode := r.FormValue("roomcode")

	fmt.Println("RoomCode:", roomcode)
	fmt.Println("Username:", username)

	room, exists := rooms[roomcode]
	if !exists {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	room.Players = append(room.Players, Player{Name: username, IsActive: true})

	http.Redirect(w, r, "/room/"+roomcode+"?user="+username, http.StatusSeeOther)
}

func createRoom(w http.ResponseWriter, r *http.Request) {
	fmt.Println("CreateRoom")
	fmt.Println(r.Method)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	roomCode := generateRoomCode(6)

	room := Room{
		Code:    roomCode,
		Players: []Player{},
	}

	room.Players = append(room.Players, Player{Name: username, IsActive: true})
	rooms[roomCode] = &room
	fmt.Println("Room Created:", rooms[roomCode])

	http.Redirect(w, r, "/room/"+roomCode+"?user="+username, http.StatusSeeOther)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("IndexHandler")
	if err := tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
}

func roomHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	roomCode := r.URL.Path[len("/room/"):]
	fmt.Println("ROOMCODE IN ROOM HANDLER:", roomCode)
	room, exists := rooms[roomCode]
	if !exists {
		http.NotFound(w, r)
		return
	}

	room.ActiveRoles = GetActiveRoles(len(room.Players))

	if err := tmpl.ExecuteTemplate(w, "game.html", room); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
}

type StartRequest struct {
	RoomCode string `json:"roomCode"`
}

func startGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req StartRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	roomCode := req.RoomCode
	room, exists := rooms[roomCode]
	if !exists {
		http.NotFound(w, r)
		return
	}

	assignRoles(room)
	fmt.Println("LETS SAY THE ROLE:")
	fmt.Println(room.Players[len(room.Players)-1].Role, room.Players[len(room.Players)-1].Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room.Players)
}

func handleAudio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Room-Code, Content-Type, X-Mime-Type")

	if r.Method == "OPTIONS" {
		return
	}

	roomCode := r.Header.Get("Room-Code")
	mimeType := r.Header.Get("X-Mime-Type")

	fmt.Println("roomCode in handleAudio:", roomCode)
	fmt.Println("mimeType:", mimeType)

	audioData, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Error reading audio data:", err)
		http.Error(w, "Error reading audio data", http.StatusBadRequest)
		return
	}

	message := struct {
		MimeType string `json:"mimeType"`
		Audio    string `json:"audio"`
	}{
		MimeType: mimeType,
		Audio:    base64.StdEncoding.EncodeToString(audioData),
	}

	messageBytes, _ := json.Marshal(message)

	fmt.Printf("Type of messageBytes %T\n", messageBytes)

	mu.Lock()
	for client := range roomsConnections[roomCode] {
		fmt.Println("Sending mimeType:", mimeType)
		if err := client.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
			fmt.Println("WriteMessage message:", err)
		}
	}
	mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func main() {
	permanentRoomCode := "ADMIN"
	room := Room{
		Code: permanentRoomCode,
		Players: []Player{
			{Name: "Saba", Role: "Villager", IsActive: true},
			{Name: "Beqa", Role: "Villager", IsActive: true},
			{Name: "Nodo", Role: "Doctor", IsActive: true},
		},
	}
	rooms[permanentRoomCode] = &room

	tmpl = template.Must(template.ParseGlob("templates/*.html"))
	fs := http.FileServer(http.Dir("assets"))

	http.Handle("/assets/", http.StripPrefix("/assets/", fs))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/room/", roomHandler)
	http.HandleFunc("/join", joinHandler)
	http.HandleFunc("/create", createRoom)
	http.HandleFunc("/start", startGame)
	http.HandleFunc("/ws/chat", handleChat)
	http.HandleFunc("/audio", handleAudio)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v\n", sig)
		os.Exit(0)
	}()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("error listening:%v\n", err)
		return
	}
}

func generateRoomCode(length int) string {
	const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	var roomCode strings.Builder

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < length; i++ {
		randomIndex := rand.Intn(len(characters))
		roomCode.WriteByte(characters[randomIndex])
	}

	return roomCode.String()
}

func assignRoles(room *Room) {
	numPlayers := len(room.Players)
	roles := []string{}

	switch {
	case numPlayers == 4:
		roles = []string{"Mafia", "Doctor", "Villager", "Villager"}
	case numPlayers <= 6:
		roles = []string{"Mafia", "Doctor", "Detective", "Villager", "Villager", "Villager"}
	case numPlayers <= 8:
		roles = []string{"Mafia", "Mafia", "Doctor", "Detective", "Villager", "Villager", "Villager", "Villager"}
	case numPlayers <= 10:
		roles = []string{"Mafia", "Mafia", "Doctor", "Detective", "Bodyguard", "Villager", "Villager", "Villager", "Villager", "Villager"}
	default:
		roles = []string{"Mafia", "Mafia", "Godfather", "Doctor", "Detective", "Bodyguard"}
		for len(roles) < numPlayers {
			roles = append(roles, "Villager")
		}
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(roles), func(i, j int) {
		roles[i], roles[j] = roles[j], roles[i]
	})

	for i := range room.Players {
		room.Players[i].Role = roles[i]
	}
}

func GetActiveRoles(playerCount int) []string {

	if playerCount < 4 {
		return []string{}
	}

	switch {
	case playerCount == 4:
		return []string{"Mafia", "Detective", "Doctor", "Villager"}
	case playerCount <= 6:
		return []string{"Mafia", "Detective", "Doctor", "2 Villagers"}
	case playerCount <= 8:
		return []string{"2 Mafia", "Detective", "Doctor", "2 Villagers"}
	case playerCount <= 10:
		return []string{"Mafia", "Mafia", "Doctor", "Detective", "Bodyguard", "Villager", "Villager", "Villager", "Villager", "Villager"}
	default:
		return []string{"Mafia", "Villager"}
	}
}
