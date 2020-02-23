package smtpmail

import (
	"context"
	"fmt"
	"log"
	"time"

	smtpsrv "github.com/alash3al/go-smtpsrv"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/burner"
	"github.com/haydenwoodhead/burner.kiwi/email"
	"github.com/jhillyerd/enmime"
)

var _ burner.EmailProvider = &SMTPMail{}

type SMTPMail struct {
	srv        *smtpsrv.Server
	listenAddr string
}

type handler struct {
	db            burner.Database
	isBlacklisted func(string) bool
}

func NewSMPTMailProvider(listenAddr string) *SMTPMail {
	return &SMTPMail{
		srv:        nil,
		listenAddr: listenAddr,
	}
}

func (s *SMTPMail) Start(websiteAddr string, db burner.Database, r *mux.Router, isBlacklisted func(string) bool) error {
	h := &handler{
		db:            db,
		isBlacklisted: isBlacklisted,
	}

	s.srv = &smtpsrv.Server{
		Name:        websiteAddr,
		Addr:        s.listenAddr,
		Handler:     h.handler,
		Addressable: h.addressable,
		MaxBodySize: 5 * 1024,
	}

	go func() {
		err := s.srv.ListenAndServe()
		log.Fatalf("SMTPMail: failed to start server: %v", err)
	}()

	return nil
}

func (h *handler) handler(req *smtpsrv.Request) error {
	envelope, err := enmime.ReadEnvelope(req.Message.Body)
	if err != nil {
		log.Printf("SMTP.handler: failed to parse envelope: %v", err)
		return fmt.Errorf("SMTP.handler: failed to parse envelope: %v", err)
	}

	partialMsg := burner.Message{
		ReceivedAt:      time.Now().Unix(),
		EmailProviderID: "smtp", // TODO: maybe a better id here? For logging purposes?
		Sender:          req.From,
		From:            envelope.GetHeader("From"),
		Subject:         envelope.GetHeader("Subject"),
		BodyPlain:       envelope.Text,
	}

	if envelope.HTML != "" {
		modifiedHTML, err := email.AddTargetBlank(envelope.HTML)
		if err != nil {
			log.Printf("SMTP.handler: failed to AddTargetBlank: %v", err)
			return fmt.Errorf("SMTP.handler: failed to AddTargetBlank: %v", err)
		}

		partialMsg.BodyHTML = modifiedHTML
	}

	for _, rcpt := range req.To {
		inbox, err := h.db.GetInboxByAddress(rcpt)
		if err != nil {
			log.Printf("SMTP.handler: failed to retrieve inbox: %v", err)
			return fmt.Errorf("SMTP.handler: failed to retrieve inbox: %v", err)
		}

		mID, err := uuid.NewRandom()
		if err != nil {
			log.Printf("SMTP.handler: failed to generate uuid for inbox: %v", err)
			return fmt.Errorf("SMTP.handler: failed to generate uuid for inbox: %v", err)
		}

		msg := partialMsg
		msg.ID = mID.String()
		msg.InboxID = inbox.ID
		msg.TTL = inbox.TTL

		err = h.db.SaveNewMessage(msg)
		if err != nil {
			log.Printf("SMTP.handler: failed to save message to db: %v", err)
			return fmt.Errorf("SMTP.handler: failed to save message to db: %v", err)
		}
	}

	return nil
}

func (h *handler) addressable(user, address string) bool {
	if h.isBlacklisted(address) {
		return false
	}

	exists, err := h.db.EmailAddressExists(address)
	if err != nil {
		log.Printf("SMTPMail: failed to query if email exists: %v", err)
		return false
	}

	return exists
}

func (s *SMTPMail) Stop() error {
	return s.srv.Shutdown(context.Background())
}

// RegisterRoute is redundant in this instance as we're not calling to an external service to register a callback
// instead we will receive all email and then be asking our db directly if we should accept this email or not.
func (s *SMTPMail) RegisterRoute(i burner.Inbox) (string, error) {
	return "smtp", nil
}
