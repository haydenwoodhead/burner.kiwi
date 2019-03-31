package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/haydenwoodhead/burner.kiwi/data"
	"github.com/haydenwoodhead/burner.kiwi/email"
	"github.com/haydenwoodhead/burner.kiwi/generateemail"
	"github.com/haydenwoodhead/burner.kiwi/token"
	"github.com/justinas/alice"
)

// Packr boxes for static templates and assets
var templates = packr.NewBox("../templates")
var staticFS = packr.NewBox("../static")

// Templates
var indexTemplate *template.Template
var messageHTMLTemplate *template.Template
var messagePlainTemplate *template.Template
var deleteTemplate *template.Template

// Static asset vars - these are overridden at build time to inject a file w/ version info
var milligram = "milligram.css"
var logo = "roger-proportional.svg"
var normalize = "normalize.css"
var custom = "custom.css"

// version number - this is also overridden at build time to inject the commit hash
var version = "dev"

// Server bundles several data types together for dependency injection into http handlers
type Server struct {
	store              *sessions.CookieStore
	websiteURL         string
	staticURL          string
	eg                 *generateemail.EmailGenerator
	email              email.Provider
	db                 data.Database
	Router             *mux.Router
	tg                 *token.Generator
	developing         bool
	usingLambda        bool
	blacklistedDomains []string
}

//NewServerInput contains key configuration parameters to be passed to NewServer()
type NewServerInput struct {
	Key                string
	URL                string
	StaticURL          string
	Email              email.Provider
	Domains            []string
	Developing         bool
	UsingLambda        bool
	RestoreRealIP      bool
	Database           data.Database
	BlacklistedDomains []string
}

// NewServer returns a server with the given settings
func NewServer(n NewServerInput) (*Server, error) {
	s := Server{}

	// Setup Templates
	indexTemplate = MustParseTemplates(templates, "base.html", "index.html")
	messageHTMLTemplate = MustParseTemplates(templates, "base.html", "message-html.html")
	messagePlainTemplate = MustParseTemplates(templates, "base.html", "message-plain.html")
	deleteTemplate = MustParseTemplates(templates, "base.html", "delete.html")

	s.store = sessions.NewCookieStore([]byte(n.Key))
	s.store.MaxAge(86402) // set max cookie age to 24 hours + 2 seconds

	s.websiteURL = n.URL
	s.staticURL = n.StaticURL

	s.developing = n.Developing

	s.usingLambda = n.UsingLambda

	s.eg = generateemail.NewEmailGenerator(n.Domains, 8)

	s.tg = token.NewGenerator(n.Key, 24*time.Hour)

	s.db = n.Database

	s.blacklistedDomains = n.BlacklistedDomains

	s.Router = mux.NewRouter()
	s.Router.StrictSlash(true) // means router will match both "/path" and "/path/"

	s.email = n.Email
	s.email.Start(s.websiteURL, s.db, s.Router, s.isBlacklisted)

	// HTML - trying to make middleware flow/handler declaration a little more readable
	s.Router.Handle("/",
		alice.New( //Middleware below
			Refresh(20),
			CacheControl(14),
			SetVersionHeader,
			s.SecurityHeaders(false),
			s.IsNew(http.HandlerFunc(s.NewInbox)),
		).ThenFunc(s.Index),
	).Methods(http.MethodGet)

	s.Router.Handle("/messages/{messageID}/",
		alice.New(
			s.CheckCookieExists,
			CacheControl(3600),
			SetVersionHeader,
			s.SecurityHeaders(true),
		).ThenFunc(s.IndividualMessage),
	).Methods(http.MethodGet)

	s.Router.Handle("/delete",
		alice.New(
			s.CheckCookieExists,
			SetVersionHeader,
			s.SecurityHeaders(false),
		).ThenFunc(s.DeleteInbox),
	).Methods(http.MethodGet)

	s.Router.Handle("/delete",
		alice.New(
			s.CheckCookieExists,
			SetVersionHeader,
			s.SecurityHeaders(false),
		).ThenFunc(s.ConfirmDeleteInbox),
	).Methods(http.MethodPost)

	// JSON API
	s.Router.Handle("/api/v1/inbox/", alice.New(JSONContentType).ThenFunc(s.NewInboxJSON)).Methods(http.MethodGet)
	s.Router.Handle("/api/v1/inbox/{inboxID}/", alice.New(JSONContentType, s.CheckPermissionJSON).ThenFunc(s.GetInboxDetailsJSON)).Methods(http.MethodGet)
	s.Router.Handle("/api/v1/inbox/{inboxID}/messages/", alice.New(JSONContentType, s.CheckPermissionJSON).ThenFunc(s.GetAllMessagesJSON)).Methods(http.MethodGet)

	// Static File Serving w/ Packr
	fs := http.StripPrefix("/static/", http.FileServer(staticFS))

	if n.Developing {
		s.Router.PathPrefix("/static/").Handler(alice.New(CacheControl(0)).Then(fs))
	} else {
		s.Router.PathPrefix("/static/").Handler(alice.New(CacheControl(15778463)).Then(fs))
	}

	if n.RestoreRealIP {
		s.Router.Use(RestoreRealIP)
	}

	// Metrics
	s.Router.Handle("/metrics", promhttp.Handler())

	return &s, nil
}

// Session Related constants
const sessionStoreKey = "session"

type key int

const (
	sessionCTXKey key = iota
)

func (s *Server) isBlacklisted(email string) bool {
	emailDomain := strings.Split(email, "@")[1]
	for _, domain := range s.blacklistedDomains {
		if domain == emailDomain {
			return true
		}
	}
	return false
}

//createRouteAndUpdate is intended to be run in a goroutine. It creates a mailgun route and updates dynamodb with
//the result. Otherwise it fails silently and this failure is picked up in the next request.
func (s *Server) createRouteAndUpdate(i data.Inbox) {
	routeID, err := s.email.RegisterRoute(i)
	if err != nil {
		log.Printf("Index JSON: failed to create route: %v", err)
		return
	}

	i.MGRouteID = routeID
	i.FailedToCreate = false
	err = s.db.SetInboxCreated(i)
	if err != nil {
		log.Printf("Index JSON: failed to update that route is created: %v", err)
	}
}

//lambdaCreateRouteAndUpdate makes use of the waitgroup then calls createRouteAndUpdate. This is because lambda
//will exit as soon as we return the response so we must make it wait
func (s *Server) lambdaCreateRouteAndUpdate(wg *sync.WaitGroup, i data.Inbox) {
	defer wg.Done()
	s.createRouteAndUpdate(i)
}

//DeleteOldRoutes deletes routes older than 24 hours
func (s *Server) DeleteOldRoutes() error {
	return s.email.DeleteExpiredRoutes()
}

// MustParseTemplates parses string templates into one template
// Function modified from: https://stackoverflow.com/questions/41856021/how-to-parse-multiple-strings-into-a-template-with-go
func MustParseTemplates(box packr.Box, templs ...string) *template.Template {
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
