package smtpmail

import (
	"io"
	"net"
	"net/mail"
	"strings"
	"time"

	"github.com/DusanKasan/parsemail"
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/burner"
	"github.com/haydenwoodhead/burner.kiwi/email"
	log "github.com/sirupsen/logrus"
)

var _ burner.EmailProvider = &SMTPMail{}

type SMTPMail struct {
	listenAddr string
	listener   *net.Listener
	server     *smtp.Server
}

type smtpBackend struct {
	handler *handler
}

type smtpSession struct {
	conState    *smtp.ConnectionState
	fromAddress string
	handler     *handler
}

type handler struct {
	db                  burner.Database
	isBlacklistedDomain func(string) bool
}

func NewMailProvider(listenAddr string) *SMTPMail {
	return &SMTPMail{
		listenAddr: listenAddr,
	}
}

func (s *SMTPMail) Start(websiteAddr string, db burner.Database, r *mux.Router, isBlacklistedDomain func(string) bool) error {
	h := &handler{
		db:                  db,
		isBlacklistedDomain: isBlacklistedDomain,
	}

	be := &smtpBackend{handler: h}

	server := smtp.NewServer(be)
	server.WriteTimeout = 20 * time.Second
	server.ReadTimeout = 20 * time.Second
	server.MaxMessageBytes = 5 * (1024 * 1024)
	server.Addr = s.listenAddr
	server.AllowInsecureAuth = true

	s.server = server

	log.Info("Starting smtp server")
	go func() {
		if s.listener != nil {
			err := s.server.Serve(*s.listener)
			if err != nil {
				log.WithError(err).Fatal("SMTP: failed to start server")
			}
		} else {
			err := s.server.ListenAndServe()
			if err != nil {
				log.WithError(err).Error("SMTP: failed to start server")
			}
		}
	}()

	return nil
}

func (b *smtpBackend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	return nil, smtp.ErrAuthUnsupported
}

func (b *smtpBackend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return &smtpSession{conState: state, handler: b.handler}, nil
}

func (s *smtpSession) Reset() {
}

func (s *smtpSession) Logout() error {
	return nil
}

func (s *smtpSession) Mail(from string, opts smtp.MailOptions) error {
	s.fromAddress = from
	return nil
}

const smtpMailBoxNotAvailableCode = 550

func (s *smtpSession) Rcpt(to string) error {
	parsedTo, err := mail.ParseAddress(to)
	if err != nil {
		log.WithError(err).WithField("to", to).Error("SMTP: failed to parse to field")
		return err
	}

	if !s.handler.emailAddressExists(parsedTo.Address) {
		return &smtp.SMTPError{
			Code:         smtpMailBoxNotAvailableCode,
			EnhancedCode: smtp.EnhancedCode{5, 1, 1},
			Message:      "Bad destination mailbox address",
		}
	}

	return nil
}

func (s *smtpSession) Data(r io.Reader) error {
	email, err := parsemail.Parse(r)
	if err != nil {
		log.WithError(err).Error("SMTP: failed to parse message body")
		return err
	}
	return s.handler.handleMessage(s.fromAddress, email)
}

func (h *handler) handleMessage(from string, parsedEmail parsemail.Email) error {
	partialMsg := burner.Message{
		ReceivedAt:      time.Now().Unix(),
		EmailProviderID: "smtp", // TODO: maybe a better id here? For logging purposes?
		Sender:          from,
		FromAddress:     getFirstFrom(parsedEmail.From).Address,
		FromName:        getFirstFrom(parsedEmail.From).Name,
		Subject:         parsedEmail.Subject,
	}

	partialMsg.BodyPlain = strings.TrimSpace(parsedEmail.TextBody)

	if parsedEmail.HTMLBody != "" {
		modifiedHTML, err := email.AddTargetBlank(strings.TrimSpace(parsedEmail.HTMLBody))
		if err != nil {
			log.WithError(err).Error("SMTP: failed to AddTargetBlank")
			return err
		}
		partialMsg.BodyHTML = modifiedHTML
	}

	for _, rcpt := range parsedEmail.To {
		inbox, err := h.db.GetInboxByAddress(rcpt.Address)
		if err != nil {
			log.WithError(err).Error("SMTP: failed to retrieve inbox")
			return err
		}

		msg := partialMsg
		msg.ID = uuid.Must(uuid.NewRandom()).String()
		msg.InboxID = inbox.ID
		msg.TTL = inbox.TTL
		err = h.db.SaveNewMessage(msg)
		if err != nil {
			log.WithError(err).Error("SMTP: failed to save message to db")
			return err
		}
	}

	return nil
}

func (h *handler) emailAddressExists(address string) bool {
	if h.isBlacklistedDomain(address) {
		return false
	}

	exists, err := h.db.EmailAddressExists(address)
	if err != nil {
		log.WithError(err).Error("SMTP: failed to query if email exists")
		return false
	}

	return exists
}

func (s *SMTPMail) Stop() error {
	return s.server.Close()
}

// RegisterRoute is redundant in this instance as we're not calling to an external service to register a callback
// instead we will receive all email and then be asking our db directly if we should accept this email or not.
func (s *SMTPMail) RegisterRoute(i burner.Inbox) (string, error) {
	return "smtp", nil
}

func getFirstFrom(from []*mail.Address) mail.Address {
	for _, f := range from {
		if f != nil {
			return *f
		}
	}
	return mail.Address{}
}
