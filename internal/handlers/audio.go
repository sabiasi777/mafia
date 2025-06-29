package handlers

// import (
// 	"fmt"
// 	"io"
// 	"net/http"

// 	"github.com/gorilla/websocket"
// )

// func HandleAudio(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Access-Control-Allow-Origin", "*")
// 	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
// 	w.Header().Set("Access-Control-Allow-Headers", "Room-Code, Content-Type, X-Mime-Type")

// 	if r.Method == "OPTIONS" {
// 		return
// 	}

// 	roomCode := r.Header.Get("Room-Code")

// 	fmt.Println("roomCode in handleAudio:", roomCode)

// 	audioData, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		fmt.Println("Error reading audio data:", err)
// 		http.Error(w, "Error reading audio data", http.StatusBadRequest)
// 		return
// 	}

// 	mu.Lock()
// 	for client := range roomsConnections[roomCode] {
// 		if err := client.WriteMessage(websocket.BinaryMessage, audioData); err != nil {
// 			fmt.Println("WriteMessage message:", err)
// 		}
// 	}
// 	mu.Unlock()

// 	w.WriteHeader(http.StatusOK)
// }
