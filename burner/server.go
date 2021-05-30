package burner

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"log"

	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/haydenwoodhead/burner.kiwi/emailgenerator"
	"github.com/haydenwoodhead/burner.kiwi/token"
	"github.com/justinas/alice"
)

// Packr boxes for static templates and assets
var templates = packr.NewBox("../templates")
var staticFS = packr.NewBox("../static")

// Templates
var indexTemplate *template.Template
var editTemplate *template.Template
var deleteTemplate *template.Template

// Static asset vars - these are overridden at build time to inject a file w/ version info
var css = "styles.css"

// version number - this is also overridden at build time to inject the commit hash
var version = "dev"

// Server bundles several data types together for dependency injection into http handlers
type Server struct {
	sessionStore *sessions.CookieStore
	eg           EmailGenerator
	email        EmailProvider
	db           Database
	Router       *mux.Router
	tg           *token.Generator

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
		tg:           token.NewGenerator(cfg.Key, 24*time.Hour),
		cfg:          cfg,
		db:           db,
		email:        email,
	}

	// Setup Templates
	indexTemplate = mustParseTemplates(templates, "base.html", "inbox.html")
	// messageHTMLTemplate = mustParseTemplates(templates, "base.html", "message-html.html")
	// messagePlainTemplate = mustParseTemplates(templates, "base.html", "message-plain.html")
	// editTemplate = mustParseTemplates(templates, "base.html", "edit.html")
	// deleteTemplate = mustParseTemplates(templates, "base.html", "delete.html")

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
	s.Router.Handle("/api/v2/inbox/", alice.New(JSONContentType).ThenFunc(s.NewInboxJSON)).Methods(http.MethodGet)
	s.Router.Handle("/api/v2/inbox/{inboxID}/", alice.New(JSONContentType, s.CheckPermissionJSON).ThenFunc(s.GetInboxDetailsJSON)).Methods(http.MethodGet)
	s.Router.Handle("/api/v2/inbox/{inboxID}/messages/", alice.New(JSONContentType, s.CheckPermissionJSON).ThenFunc(s.GetAllMessagesJSON)).Methods(http.MethodGet)

	// Static File Serving w/ Packr
	fs := http.StripPrefix("/static/", http.FileServer(staticFS))

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

// mustParseTemplates parses string templates into one template
// Function modified from: https://stackoverflow.com/questions/41856021/how-to-parse-multiple-strings-into-a-template-with-go
func mustParseTemplates(box packr.Box, templs ...string) *template.Template {
	t := template.New("templ")

	for i, templ := range templs {
		templateString, err := box.FindString(templ)
		if err != nil {
			log.Fatalf("MustParseTemplates: failed to find template: %v", err)
		}

		_, err = t.New(fmt.Sprintf("%v", i)).Parse(templateString)
		if err != nil {
			log.Fatalf("MustParseTemplates: failed to parse template: %v", err)

		}
	}

	return t
}
