package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/haydenwoodhead/burnerkiwi/generateemail"
	"github.com/haydenwoodhead/burnerkiwi/token"
	"github.com/justinas/alice"
	"gopkg.in/mailgun/mailgun-go.v1"
)

// version number - this is also overridden at build time to inject the commit hash
const version = "dev"

// Server bundles several data types together for dependency injection into http handlers
type Server struct {
	store        *sessions.CookieStore
	websiteURL   string
	staticURL    string
	eg           *generateemail.EmailGenerator
	mg           mailgun.Mailgun
	dynDB        *dynamodb.DynamoDB
	Router       *mux.Router
	tg           *token.Generator
	disableHTTPS bool
}

// NewServer returns a server with the given settings
func NewServer(key, url, static, mgDomain, mgKey string, domains []string, disableHTTPS bool) (*Server, error) {
	s := Server{}

	s.store = sessions.NewCookieStore([]byte(key))
	s.store.MaxAge(86402) // set max cookie age to 24 hours + 2 seconds

	s.websiteURL = url
	s.staticURL = static

	s.disableHTTPS = disableHTTPS

	s.mg = mailgun.NewMailgun(mgDomain, mgKey, "")

	s.eg = generateemail.NewEmailGenerator(domains, 8)

	s.tg = token.NewGenerator(key, 24*time.Hour)

	awsSession := session.Must(session.NewSession())
	s.dynDB = dynamodb.New(awsSession)

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

	// Temp Static
	fs := http.StripPrefix("/static/", http.FileServer(http.Dir("static/")))
	s.Router.PathPrefix("/static/").Handler(fs)

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

	err = s.setInboxCreated(i)

	if err != nil {
		log.Printf("Index JSON: failed to update that route is created: %v", err)
	}
}
