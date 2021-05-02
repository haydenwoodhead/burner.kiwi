package burner

import (
	"github.com/gorilla/mux"
)

//EmailProvider represents a mail provider that burner.kiwi can use to receive mail from
type EmailProvider interface {
	Start(websiteAddr string, db Database, r *mux.Router, isBlacklistedDomain func(string) bool) error
	Stop() error
	RegisterRoute(i Inbox) (string, error)
}

type EmailGenerator interface {
	GetHosts() []string
	HostsContains(host string) bool
	NewRandom() string
	NewFromUserAndHost(r string, h string) (string, error)
	VerifyUser(r string) error
	VerifyHost(h string) error
}
