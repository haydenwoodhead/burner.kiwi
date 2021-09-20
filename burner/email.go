package burner

import (
	"strings"
	"sync"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

//EmailProvider represents a mail provider that burner.kiwi can use to receive mail from
type EmailProvider interface {
	Start(websiteAddr string, db Database, r *mux.Router, isBlacklistedDomain func(string) bool) error
	Stop() error
	RegisterRoute(i Inbox) (string, error)
}

type EmailGenerator interface {
	NewRandom() string
	NewFromUserAndHost(user string, host string) (string, error)
}

func (s *Server) isBlacklistedDomain(email string) bool {
	for _, domain := range s.cfg.BlacklistedDomains {
		if strings.Contains(email, domain) {
			return true
		}
	}
	return false
}

//createRouteAndUpdate is intended to be run in a goroutine. It creates an email route and updates the db with
//the result. Otherwise it fails silently and this failure is picked up in the next request.
func (s *Server) createRouteAndUpdate(i Inbox) {
	routeID, err := s.email.RegisterRoute(i)
	if err != nil {
		log.WithField("inbox", i.ID).WithError(err).Error("createRouteAndUpdate: failed to create route")

		i.FailedToCreate = true
		err = s.db.SetInboxFailed(i)
		if err != nil {
			log.WithField("inbox", i.ID).WithError(err).Error("createRouteAndUpdate: failed to set route as having failed to create")
		}

		return
	}

	i.EmailProviderRouteID = routeID
	i.FailedToCreate = false
	err = s.db.SetInboxCreated(i)
	if err != nil {
		log.WithField("inbox", i.ID).WithError(err).Error("createRouteAndUpdate: failed to set inbox created")
	}
}

//lambdaCreateRouteAndUpdate makes use of the waitgroup then calls createRouteAndUpdate. This is because lambda
//will exit as soon as we return the response so we must make it wait
func (s *Server) lambdaCreateRouteAndUpdate(wg *sync.WaitGroup, i Inbox) {
	defer wg.Done()
	s.createRouteAndUpdate(i)
}
