package logic

import (
	"math/rand"
	"time"

	"github.com/sabiasi777/mafia/internal/models"
)

func AssignRoles(room *models.Room) {
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
