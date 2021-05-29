package mailgunmail

import (
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/burner"
	"github.com/haydenwoodhead/burner.kiwi/email"
	log "github.com/sirupsen/logrus"
	mailgun "gopkg.in/mailgun/mailgun-go.v1"
)

var _ burner.EmailProvider = &MailgunMail{}

type mailgunAPI interface {
	DeleteRoute(id string) error
	GetRoutes(limit, skip int) (int, []mailgun.Route, error)
	CreateRoute(m mailgun.Route) (mailgun.Route, error)
	VerifyWebhookRequest(req *http.Request) (verified bool, err error)
}

// MailgunMail is a mailgun implementation of the EmailProvider interface
type MailgunMail struct {
	websiteAddr         string
	mg                  mailgunAPI
	db                  burner.Database
	isBlacklistedDomain func(string) bool
}

// NewMailProvider creates a new Mailgun EmailProvider
func NewMailProvider(domain string, key string) *MailgunMail {
	return &MailgunMail{
		mg: mailgun.NewMailgun(domain, key, ""),
	}
}

// Start implements EmailProvider Start()
func (m *MailgunMail) Start(websiteAddr string, db burner.Database, r *mux.Router, isBlacklistedDomain func(string) bool) error {
	m.db = db
	m.isBlacklistedDomain = isBlacklistedDomain
	m.websiteAddr = websiteAddr
	r.HandleFunc("/mg/incoming/{inboxID}/", m.mailgunIncoming).Methods(http.MethodPost)

	go func() {
		for {
			log.Info("Mailgun: deleting expired routes")
			err := m.deleteExpiredRoutes()
			if err != nil {
				log.WithError(err).Error("Mailgun: failed to delete expired routes")
			}
			log.Info("Mailgun: deleted expired routes")
			time.Sleep(1 * time.Hour)
		}
	}()

	return nil
}

// Stop implements EmailProvider Stop()
func (m *MailgunMail) Stop() error {
	return nil
}

// RegisterRoute implements RegisterRoute()
func (m *MailgunMail) RegisterRoute(i burner.Inbox) (string, error) {
	routeAddr := m.websiteAddr + "/mg/incoming/" + i.ID + "/"
	route, err := m.mg.CreateRoute(mailgun.Route{
		Priority:    1,
		Description: strconv.Itoa(int(i.TTL)),
		Expression:  "match_recipient(\"" + i.Address + "\")",
		Actions:     []string{"forward(\"" + routeAddr + "\")", "store()", "stop()"},
	})
	return route.ID, fmt.Errorf("Mailgun - failed to create route: %w", err)
}

func (m *MailgunMail) deleteExpiredRoutes() error {
	_, routes, err := m.mg.GetRoutes(1000, 0)
	if err != nil {
		return fmt.Errorf("Mailgun - failed to get routes to delete: %w", err)
	}

	for _, r := range routes {
		tInt, err := strconv.ParseInt(r.Description, 10, 64)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{"desc": r.Description, "id": r.ID}).Error("Mailgun.deleteExpiredRoutes: failed to parse route description as int")
			continue
		}

		t := time.Unix(tInt, 0)

		// if our route's ttl (expiration time) is before now... then delete it
		if t.Before(time.Now()) {
			err := m.mg.DeleteRoute(r.ID)
			if err != nil {
				log.WithError(err).WithField("id", r.ID).Error("Mailgun.deleteExpiredRoutes: failed to delete route")
				continue
			}
		}
	}

	return nil
}

func (m *MailgunMail) mailgunIncoming(w http.ResponseWriter, r *http.Request) {
	verified, err := m.mg.VerifyWebhookRequest(r)
	if err != nil {
		log.WithError(err).Error("MailgunIncoming: failed to verify request")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !verified {
		log.Info("MailgunIncoming: invalid request")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if m.isBlacklistedDomain(r.FormValue("sender")) {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	vars := mux.Vars(r)
	id := vars["inboxID"]

	inbox, err := m.db.GetInboxByID(id)
	if err != nil {
		log.WithError(err).WithField("id", id).Error("MailgunIncoming: failed to get inbox")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	address, err := mail.ParseAddress(r.FormValue("from"))
	if err != nil {
		log.WithError(err).WithField("id", id).Error("MailgunIncoming: failed to parse from address")
		return
	}

	msg := burner.Message{
		ID:              uuid.Must(uuid.NewRandom()).String(),
		InboxID:         inbox.ID,
		TTL:             inbox.TTL,
		ReceivedAt:      time.Now().Unix(),
		EmailProviderID: r.FormValue("message-id"),
		Sender:          r.FormValue("sender"),
		FromName:        address.Name,
		FromAddress:     address.Address,
		Subject:         r.FormValue("subject"),
		BodyPlain:       r.FormValue("body-plain"),
	}

	html := r.FormValue("body-html")

	// Check to see if there is anything in html before we modify it. Otherwise we end up setting a blank html doc
	// on all plaintext emails preventing them from being displayed.
	if html != "" {
		modifiedHTML, err := email.AddTargetBlank(html)
		if err != nil {
			log.WithError(err).Error("MailgunIncoming: failed to AddTargetBlank")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		msg.BodyHTML = modifiedHTML
	}

	err = m.db.SaveNewMessage(msg)
	if err != nil {
		log.WithError(err).Error("MailgunIncoming: failed to save message to db")
	}

	_, err = w.Write([]byte(id))
	if err != nil {
		log.WithError(err).Error("MailgunIncoming: failed to write response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
