package handlers

import (
	"html/template"

	"github.com/sabiasi777/mafia/internal/models"
)

type RoomManager struct {
	Rooms (map[string]*models.Room)
	Tmpl  *template.Template
}
