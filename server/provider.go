package server

import (
	"github.com/gorilla/mux"
)

//EmailProvider represents a mail provider that burner.kiwi can use to receive mail from
type EmailProvider interface {
	Start(websiteAddr string, db Database, r *mux.Router, isBlacklisted func(string) bool) error
	Stop() error
	RegisterRoute(i Inbox) (string, error)
	DeleteExpiredRoutes() error
}
