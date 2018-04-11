package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/haydenwoodhead/burnerkiwi/generateemail"
	"github.com/haydenwoodhead/burnerkiwi/token"
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
	tg         *token.Generator
}

func NewServer(key, url, mgDomain, mgKey string, domains []string) (*Server, error) {
	s := Server{}

	s.store = sessions.NewCookieStore([]byte(key))
	s.store.MaxAge(86402) // set max cookie age to 24 hours + 2 seconds

	s.websiteURL = url

	s.mg = mailgun.NewMailgun(mgDomain, mgKey, "")

	s.eg = generateemail.NewEmailGenerator(domains, 8)

	s.tg = token.NewGenerator(key, 24*time.Hour)

	awsSession := session.Must(session.NewSession())
	s.dynDB = dynamodb.New(awsSession)

	s.Router = mux.NewRouter()

	// HTML
	s.Router.Handle("/", alice.New(s.IsNew(http.HandlerFunc(s.NewEmail))).ThenFunc(s.Index)).Methods(http.MethodGet)

	// JSON API
	s.Router.Handle("/api/v1/inbox", alice.New(JSONContentType).ThenFunc(s.NewEmailJSON)).Methods(http.MethodGet)
	s.Router.Handle("/api/v1/inbox/{emailID}", alice.New(JSONContentType, s.CheckPermissionJSON).ThenFunc(s.GetEmailDetailsJSON)).Methods(http.MethodGet)

	// Mailgun Incoming
	s.Router.HandleFunc("/mg/incoming/{emailID}/", s.MailgunIncoming).Methods(http.MethodPost)

	return &s, nil
}

type Email struct {
	Address        string `dynamodbav:"email_address" json:"address"`
	ID             string `dynamodbav:"id" json:"id"`
	CreatedAt      int64  `dynamodbav:"created_at" json:"created_at"`
	TTL            int64  `dynamodbav:"ttl" json:"ttl"`
	MGRouteID      string `dynamodbav:"mg_routeid" json:"-"`
	FailedToCreate bool   `dynamodbav:"failed_to_create" json:"-"`
}

// NewEmail returns an email with failed to create and route id set. Upon successful registration of the mailun
// route we set these as true.
func NewEmail() Email {
	return Email{
		FailedToCreate: true,
		MGRouteID:      "-",
	}
}

type Message struct {
	EmailID    string `dynamodbav:"email_id"`
	ID         string `dynamodbav:"message_id"`
	ReceivedAt int64  `dynamodbav:"received_at"`
	MGID       string `dynamodbav:"mg_id"`
	Sender     string `dynamodbav:"sender"`
	From       string `dynamodbav:"from"`
	Subject    string `dynamodbav:"subject"`
	BodyHTML   string `dynamodbav:"body_html"`
	BodyPlain  string `dynamodbav:"body_plain"`
	TTL        int64  `dynamodbav:"ttl"`
}

// createRoute registers the email route with mailgun
func (s *Server) createRoute(e *Email) error {
	routeAddr := s.websiteURL + "/mg/incoming/" + e.ID + "/"

	route, err := s.mg.CreateRoute(mailgun.Route{
		Priority:    1,
		Description: strconv.FormatInt(e.TTL, 10),
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

// saveNewEmail saves the passed in email to dynamodb
func (s *Server) saveNewEmail(e Email) error {
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

// getEmailByID gets an email by id
func (s *Server) getEmailByID(id string) (Email, error) {
	var e Email

	o, err := s.dynDB.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
		TableName: aws.String("emails"),
	})

	if err != nil {
		return Email{}, err
	}

	err = dynamodbattribute.UnmarshalMap(o.Item, &e)

	if err != nil {
		return Email{}, err
	}

	return e, nil
}

// emailExists checks to see if the given email address already exists in our db. It will only return
// false if we can explicitly verify the email doesn't exist.
func (s *Server) emailExists(a string) (bool, error) {
	q := &dynamodb.QueryInput{
		KeyConditionExpression: aws.String("email_address = :e"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":e": {
				S: aws.String(a),
			},
		},
		IndexName: aws.String("email_address-index"),
		TableName: aws.String("emails"),
	}

	res, err := s.dynDB.Query(q)

	if err != nil {
		return false, err
	}

	if len(res.Items) == 0 {
		return false, nil
	}

	return true, nil
}

// setEmailCreated updates dynamodb and sets the email as created and adds a mailgun route
func (s *Server) setEmailCreated(e Email) error {
	u := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#F": aws.String("failed_to_create"),
			"#M": aws.String("mg_routeid"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":f": {
				BOOL: aws.Bool(false),
			},
			":m": {
				S: aws.String(e.MGRouteID),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(e.ID),
			},
		},
		TableName:        aws.String("emails"),
		UpdateExpression: aws.String("SET #F = :f, #M = :m"),
	}

	_, err := s.dynDB.UpdateItem(u)

	if err != nil {
		return fmt.Errorf("setEmailCreated: failed to mark email as created: %v", err)
	}

	return err
}

//createRouteAndUpdate is intended to be run in a goroutine. It creates a mailgun route and updates dynamodb with
//the result. Otherwise it fails silently and this failure is picked up in the next request.
func (s *Server) createRouteAndUpdate(e Email) {
	err := s.createRoute(&e)

	if err != nil {
		log.Printf("Index JSON: failed to create route: %v", err)

		return
	}

	err = s.setEmailCreated(e)

	if err != nil {
		log.Printf("Index JSON: failed to update that route is created: %v", err)
	}
}

// saveMessage saves a given message to dynamodb
func (s *Server) saveNewMessage(m Message) error {
	mv, err := dynamodbattribute.MarshalMap(m)

	if err != nil {
		return fmt.Errorf("saveMessage: failed to marshal struct to attribute value: %v", err)
	}

	_, err = s.dynDB.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("messages"),
		Item:      mv,
	})

	if err != nil {
		return fmt.Errorf("saveMessage: failed to put to dynamodb: %v", err)
	}

	return nil
}
