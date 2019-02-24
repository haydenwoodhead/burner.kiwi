package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/haydenwoodhead/burner.kiwi/data"
	"github.com/haydenwoodhead/burner.kiwi/generateemail"
	"github.com/haydenwoodhead/burner.kiwi/token"
	"github.com/justinas/alice"
	mailgun "gopkg.in/mailgun/mailgun-go.v1"
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
	mg                 mailgun.Mailgun
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
	MGDomain           string
	MGKey              string
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

	s.mg = mailgun.NewMailgun(n.MGDomain, n.MGKey, "")

	s.eg = generateemail.NewEmailGenerator(n.Domains, 8)

	s.tg = token.NewGenerator(n.Key, 24*time.Hour)

	s.db = n.Database

	s.blacklistedDomains = n.BlacklistedDomains

	s.Router = mux.NewRouter()
	s.Router.StrictSlash(true) // means router will match both "/path" and "/path/"

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

	// Mailgun Incoming
	s.Router.HandleFunc("/mg/incoming/{inboxID}/", s.MailgunIncoming).Methods(http.MethodPost)

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

// createRoute registers the email route with mailgun
func (s *Server) createRoute(i *data.Inbox) error {
	routeAddr := s.websiteURL + "/mg/incoming/" + i.ID + "/"

	route, err := s.mg.CreateRoute(mailgun.Route{
		Priority:    1,
		Description: strconv.FormatInt(i.TTL, 10),
		Expression:  "match_recipient(\"" + i.Address + "\")",
		Actions:     []string{"forward(\"" + routeAddr + "\")", "store()", "stop()"},
	})

	if err != nil {
		i.FailedToCreate = true
		return fmt.Errorf("createRoute: failed to create mailgun route: %v", err)
	}

	i.MGRouteID = route.ID

	return nil
}

//createRouteAndUpdate is intended to be run in a goroutine. It creates a mailgun route and updates dynamodb with
//the result. Otherwise it fails silently and this failure is picked up in the next request.
func (s *Server) createRouteAndUpdate(i data.Inbox) {
	err := s.createRoute(&i)

	if err != nil {
		log.Printf("Index JSON: failed to create route: %v", err)
		return
	}

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
func (s *Server) DeleteOldRoutes() ([]mailgun.Route, error) {
	_, rs, err := s.mg.GetRoutes(1000, 0)

	if err != nil {
		return []mailgun.Route{}, fmt.Errorf("deleteOldRoutes: failed to get routes: %v", err)
	}

	var failed []mailgun.Route

	for _, r := range rs {
		tInt, err := strconv.ParseInt(r.Description, 10, 64)

		if err != nil {
			failed = append(failed, r)
			continue
		}

		t := time.Unix(tInt, 0)

		// if our route's ttl (expiration time) is before now... then delete it
		if t.Before(time.Now()) {
			err := s.mg.DeleteRoute(r.ID)

			if err != nil {
				failed = append(failed, r)
				continue
			}
		}
	}

	return failed, nil
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
