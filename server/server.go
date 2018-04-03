package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/haydenwoodhead/burnerkiwi/generateemail"
	"github.com/justinas/alice"
	"gopkg.in/mailgun/mailgun-go.v1"
)

type Server struct {
	store      *sessions.CookieStore
	websiteURL string
	eg         *generateemail.EmailGenerator
	mg         mailgun.Mailgun
	dynDB      *dynamodb.DynamoDB
	Router     *mux.Router
}

func NewServer(key, url, mgDomain, mgKey string, domains []string) (*Server, error) {
	s := Server{}

	s.store = sessions.NewCookieStore([]byte(key))
	s.store.MaxAge(86402) // set max cookie age to 24 hours + 2 seconds

	s.websiteURL = url

	s.mg = mailgun.NewMailgun(mgDomain, mgKey, "")

	s.eg = generateemail.NewEmailGenerator(domains, key, 8)

	awsSession := session.Must(session.NewSession())
	s.dynDB = dynamodb.New(awsSession)

	s.Router = mux.NewRouter()
	s.Router.HandleFunc("/", s.Index)
	s.Router.Handle("/.json", alice.New().ThenFunc(s.IndexJSON))
	//r.HandleFunc("/.json", )
	//r.HandleFunc("/inbox/{address}", InboxHTML)
	//r.HandleFunc("/inbox/{address}.json", InboxJSON)

	return &s, nil
}

type Email struct {
	Address        string `dynamodbav:"email_address"`
	ID             string `dynamodbav:"id"`
	CreatedAt      int64  `dynamodbav:"created_at"`
	TTL            int64  `dynamodbav:"ttl"`
	MGRouteID      string `dynamodbav:"mg_routeid"`
	FailedToCreate bool   `dynamodbav:"failed_to_create"`
}

// Index checks to see if a session already exists for the user. If so it redirects them to their page otherwise
// it generates a new email address for them and then redirects.
func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.store.Get(r, "session")

	if !sess.IsNew {
		id, ok := sess.Values["email_id"].(string)

		if !ok {
			log.Printf("Index: session id value not a string")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, s.websiteURL+"/inbox/"+id, 302)
	}

	var e Email

	addr, err := s.eg.NewRandom()

	if err != nil {
		log.Printf("Index: failed to generate new random email: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	id, err := uuid.NewRandom()

	if err != nil {
		log.Printf("Index: failed to generate new random id: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	e.Address = addr
	e.ID = id.String()
	e.CreatedAt = time.Now().Unix()
	e.TTL = time.Now().Add(time.Hour * 24).Unix()

	// Create route and save to dynamodb
	go func(e *Email) {
		err := s.createRoute(e)

		if err != nil {
			log.Println(err)
		}

		err = s.saveEmail(e)

		if err != nil {
			log.Println(err)
		}
	}(&e)

	sess.Values["email_id"] = e.ID
	sess.Values["email"] = e.Address
	sess.Values["ttl"] = e.TTL
	sess.Save(r, w)
	http.Redirect(w, r, s.websiteURL+"/inbox/"+id.String(), 302)
}

func (s *Server) IndexJSON(w http.ResponseWriter, r *http.Request) {
	var e Email

	addr, err := s.eg.NewRandom()

	if err != nil {
		log.Printf("Index: failed to generate new random email: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	id, err := uuid.NewRandom()

	if err != nil {
		log.Printf("Index: failed to generate new random id: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	e.Address = addr
	e.ID = id.String()
	e.CreatedAt = time.Now().Unix()
	e.TTL = time.Now().Add(time.Hour * 24).Unix()

	// Create route and save to dynamodb
	go func(e *Email) {
		err := s.createRoute(e)

		if err != nil {
			log.Println(err)
		}

		err = s.saveEmail(e)

		if err != nil {
			log.Println(err)
		}
	}(&e)
}

func InboxHTML(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("success"))
}

func InboxJSON(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("success json"))
}

// createRoute registers the email route with mailgun
func (s *Server) createRoute(e *Email) error {
	routeAddr := s.websiteURL + "/inbox/" + e.ID + "/"

	route, err := s.mg.CreateRoute(mailgun.Route{
		Priority:    1,
		Description: "Create route for email: " + e.Address,
		Expression:  "match_recipient(\"" + e.Address + "\")",
		Actions:     []string{"forward(\"" + routeAddr + "\")", "store()", "stop()"},
	})

	if err != nil {
		err := fmt.Errorf("createRoute: failed to create mailgun route: %v", err)
		e.FailedToCreate = true
		return err
	}

	e.MGRouteID = route.ID

	return nil
}

// saveEmail saves the passed in email to dynamodb
func (s *Server) saveEmail(e *Email) error {
	av, err := dynamodbattribute.MarshalMap(e)

	if err != nil {
		return fmt.Errorf("putEmailToDB: failed to marshal struct to attribute value: %v", err)

	}

	_, err = s.dynDB.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("emails"),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("putEmailToDB: failed to put to dynamodb: %v", err)
	}

	return nil
}
