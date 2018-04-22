package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/haydenwoodhead/burnerkiwi/generateemail"
	"github.com/haydenwoodhead/burnerkiwi/token"
	"github.com/justinas/alice"
	"gopkg.in/mailgun/mailgun-go.v1"
)

// Packr boxes for static templates and assets
var templates = packr.NewBox("../templates")
var staticFS = packr.NewBox("../static")

// Templates
var indexTemplate = MustParseTemplates(templates.String("base.html"), templates.String("index.html"))
var messageHTMLTemplate = MustParseTemplates(templates.String("base.html"), templates.String("message-html.html"))
var messagePlainTemplate = MustParseTemplates(templates.String("base.html"), templates.String("message-plain.html"))
var deleteTemplate = MustParseTemplates(templates.String("base.html"), templates.String("delete.html"))

// Static asset vars - these are overridden at build time to inject a file w/ version info
const milligram = "milligram.css"
const logo = "logo-placeholder.png"
const normalize = "normalize.css"
const custom = "custom.css"

// version number - this is also overridden at build time to inject the commit hash
const version = "dev"

// Server bundles several data types together for dependency injection into http handlers
type Server struct {
	store      *sessions.CookieStore
	websiteURL string
	staticURL  string
	eg         *generateemail.EmailGenerator
	mg         mailgun.Mailgun
	db         Database
	Router     *mux.Router
	tg         *token.Generator
	developing bool
	deleteKey  string
}

//NewServerInput contains key configuration parameters to be passed to NewServer()
type NewServerInput struct {
	Key        string
	URL        string
	StaticURL  string
	MGDomain   string
	MGKey      string
	DeleteKey  string
	Domains    []string
	Developing bool
	Database   Database
}

// NewServer returns a server with the given settings
func NewServer(n NewServerInput) (*Server, error) {
	s := Server{}

	s.store = sessions.NewCookieStore([]byte(n.Key))
	s.store.MaxAge(86402) // set max cookie age to 24 hours + 2 seconds

	s.websiteURL = n.URL
	s.staticURL = n.StaticURL

	s.developing = n.Developing

	s.mg = mailgun.NewMailgun(n.MGDomain, n.MGKey, "")

	s.eg = generateemail.NewEmailGenerator(n.Domains, 8)

	s.tg = token.NewGenerator(n.Key, 24*time.Hour)

	s.deleteKey = n.DeleteKey

	s.db = n.Database

	s.Router = mux.NewRouter()
	s.Router.StrictSlash(true) // means router will match both "/path" and "/path/"

	// HTML - trying to make middleware flow/handler declaration a little more readable
	s.Router.Handle("/",
		alice.New( //Middleware below
			s.IsNew(http.HandlerFunc(s.NewInbox)),
			Refresh(20),
			CacheControl(14),
			SetVersionHeader,
			s.SecurityHeaders,
		).ThenFunc(s.Index),
	).Methods(http.MethodGet)

	s.Router.Handle("/messages/{messageID}/",
		alice.New(
			s.CheckCookieExists(returnHTMLError),
			CacheControl(3600),
			SetVersionHeader,
			s.SecurityHeaders,
		).ThenFunc(s.IndividualMessage),
	).Methods(http.MethodGet)

	s.Router.Handle("/delete",
		alice.New(
			s.CheckCookieExists(returnHTMLError),
			SetVersionHeader,
			s.SecurityHeaders,
		).ThenFunc(s.DeleteInbox),
	).Methods(http.MethodGet)

	s.Router.Handle("/delete",
		alice.New(
			s.CheckCookieExists(returnHTMLError),
			SetVersionHeader,
			s.SecurityHeaders,
		).ThenFunc(s.ConfirmDeleteInbox),
	).Methods(http.MethodPost)

	// JSON API
	s.Router.Handle("/api/v1/inbox/", alice.New(JSONContentType).ThenFunc(s.NewInboxJSON)).Methods(http.MethodGet)
	s.Router.Handle("/api/v1/inbox/{inboxID}/", alice.New(JSONContentType, s.CheckPermissionJSON).ThenFunc(s.GetInboxDetailsJSON)).Methods(http.MethodGet)
	s.Router.Handle("/api/v1/inbox/{inboxID}/messages/", alice.New(JSONContentType, s.CheckPermissionJSON).ThenFunc(s.GetAllMessagesJSON)).Methods(http.MethodGet)

	// Mailgun Incoming
	s.Router.HandleFunc("/mg/incoming/{inboxID}/", s.MailgunIncoming).Methods(http.MethodPost)

	// Delete Old Routes Endpoint
	s.Router.HandleFunc("/bk/deleteoldroutes/", s.DeleteOldRoutesEndpoint)

	// Static File Serving w/ Packr
	fs := http.StripPrefix("/static/", http.FileServer(staticFS))

	if n.Developing {
		s.Router.PathPrefix("/static/").Handler(alice.New(CacheControl(0)).Then(fs))
	} else {
		s.Router.PathPrefix("/static/").Handler(alice.New(CacheControl(15778463)).Then(fs))
	}

	return &s, nil
}

// Session Related constants
const sessionStoreKey = "session"

type key int

const (
	sessionCTXKey key = iota
)

// createRoute registers the email route with mailgun
func (s *Server) createRoute(i *Inbox) error {
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
func (s *Server) createRouteAndUpdate(i Inbox) {
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

//MustParseTemplates parses string templates into one template
// Function modified from: https://stackoverflow.com/questions/41856021/how-to-parse-multiple-strings-into-a-template-with-go
func MustParseTemplates(templs ...string) *template.Template {
	t := template.New("templ")

	for i, templ := range templs {
		_, err := t.New(fmt.Sprintf("%v", i)).Parse(templ)

		if err != nil {
			panic(err)
		}

	}

	return t
}
