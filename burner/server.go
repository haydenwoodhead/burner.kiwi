package burner

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/haydenwoodhead/burner.kiwi/emailgenerator"
	"github.com/haydenwoodhead/burner.kiwi/notary"
	"github.com/justinas/alice"
)

// version number - this is also overridden at build time to inject the commit hash
var version = "dev"

// Server bundles several data types together for dependency injection into http handlers
type Server struct {
	sessionStore *sessions.CookieStore
	eg           EmailGenerator
	email        EmailProvider
	db           Database
	Router       *mux.Router
	notariser    *notary.Notary

	cfg Config
}

//Config contains key configuration parameters to be passed to New()
type Config struct {
	Key                string
	URL                string
	StaticURL          string
	Email              EmailProvider
	Domains            []string
	Developing         bool
	UsingLambda        bool
	RestoreRealIP      bool
	Database           Database
	BlacklistedDomains []string
}

// New returns a burner with the given settings
func New(cfg Config, db Database, email EmailProvider) (*Server, error) {
	s := Server{
		sessionStore: sessions.NewCookieStore([]byte(cfg.Key)),
		eg:           emailgenerator.New(cfg.Domains, 8),
		notariser:    notary.New(cfg.Key),
		cfg:          cfg,
		db:           db,
		email:        email,
	}

	if !s.cfg.Developing {
		s.getIndexTemplate()
		s.getDeleteTemplate()
		s.getEditTemplate()
	}

	s.sessionStore.MaxAge(86402) // set max cookie age to 24 hours + 2 seconds

	err := s.db.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start database: %w", err)
	}

	err = s.email.Start(cfg.URL, s.db, s.Router, s.isBlacklistedDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to start email provider: %w", err)
	}

	s.Router = mux.NewRouter()
	s.Router.StrictSlash(true) // means router will match both "/path" and "/path/"

	// HTML - trying to make middleware flow/handler declaration a little more readable
	s.Router.Handle("/",
		alice.New( //Middleware below
			Refresh(20),
			SetVersionHeader,
			s.SecurityHeaders(),
		).ThenFunc(s.Index),
	).Methods(http.MethodGet)

	s.Router.Handle("/messages/{messageID}/",
		alice.New(
			s.CheckSessionCookieExists,
			SetVersionHeader,
			s.SecurityHeaders(),
		).ThenFunc(s.IndividualMessage),
	).Methods(http.MethodGet)

	s.Router.Handle("/edit",
		alice.New(
			s.CheckSessionCookieExists,
			SetVersionHeader,
			s.SecurityHeaders(),
		).ThenFunc(s.EditInbox),
	).Methods(http.MethodGet)

	s.Router.Handle("/edit",
		alice.New(
			s.CheckSessionCookieExists,
			SetVersionHeader,
			s.SecurityHeaders(),
		).ThenFunc(s.NewNamedInbox),
	).Methods(http.MethodPost)

	s.Router.Handle("/delete",
		alice.New(
			s.CheckSessionCookieExists,
			SetVersionHeader,
			s.SecurityHeaders(),
		).ThenFunc(s.DeleteInbox),
	).Methods(http.MethodGet)

	s.Router.Handle("/delete",
		alice.New(
			s.CheckSessionCookieExists,
			SetVersionHeader,
			s.SecurityHeaders(),
		).ThenFunc(s.ConfirmDeleteInbox),
	).Methods(http.MethodPost)

	// JSON API
	s.Router.Handle("/api/v2/inbox", alice.New(JSONContentType).ThenFunc(s.NewInboxJSON)).Methods(http.MethodGet)
	s.Router.Handle("/api/v2/inbox/{inboxID}", alice.New(JSONContentType, s.CheckPermissionJSON).ThenFunc(s.GetInboxDetailsJSON)).Methods(http.MethodGet)
	s.Router.Handle("/api/v2/inbox/{inboxID}/messages", alice.New(JSONContentType, s.CheckPermissionJSON).ThenFunc(s.GetAllMessagesJSON)).Methods(http.MethodGet)

	// Static File Serving
	fs := http.StripPrefix("/static/", http.FileServer(s.getStaticFS()))

	if cfg.Developing {
		s.Router.PathPrefix("/static/").Handler(alice.New(CacheControl(0)).Then(fs))
	} else {
		s.Router.PathPrefix("/static/").Handler(alice.New(CacheControl(15778463)).Then(fs))
	}

	if cfg.RestoreRealIP {
		s.Router.Use(RestoreRealIP)
	}

	s.Router.HandleFunc("/ping", s.Ping)

	return &s, nil
}

// Ping returns PONG when called
func (s *Server) Ping(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("PONG"))
	if err != nil {
		log.Printf("ping - failed to write out response: %v", err)
	}
}
