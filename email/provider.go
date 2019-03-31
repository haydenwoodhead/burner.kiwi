package email

import (
	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/data"
)

//Provider represents a mail provider that burner.kiwi can use to receive mail from
type Provider interface {
	Start(websiteAddr string, db data.Database, r *mux.Router, isBlacklisted func(string) bool) error
	Stop() error
	RegisterRoute(i data.Inbox) (string, error)
	DeleteExpiredRoutes() error
}

