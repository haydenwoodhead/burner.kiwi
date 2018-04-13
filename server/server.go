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

// Server bundles several data types together for dependency injection into http handlers
type Server struct {
	store      *sessions.CookieStore
	websiteURL string
	eg         *generateemail.EmailGenerator
	mg         mailgun.Mailgun
	dynDB      *dynamodb.DynamoDB
	Router     *mux.Router
	tg         *token.Generator
}

const sessionStoreKey = "session"

type key int

const (
	sessionCTXKey key = iota
)

// NewServer returns a server with the given settings
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
	s.Router.StrictSlash(true) // means router will match both "/path" and "/path/"

	// HTML
	s.Router.Handle("/", alice.New(s.IsNew(http.HandlerFunc(s.NewInbox))).ThenFunc(s.Index)).Methods(http.MethodGet)

	// JSON API
	s.Router.Handle("/api/v1/inbox/", alice.New(JSONContentType).ThenFunc(s.NewInboxJSON)).Methods(http.MethodGet)
	s.Router.Handle("/api/v1/inbox/{inboxID}/", alice.New(JSONContentType, s.CheckPermissionJSON).ThenFunc(s.GetInboxDetailsJSON)).Methods(http.MethodGet)
	s.Router.Handle("/api/v1/inbox/{inboxID}/messages/", alice.New(JSONContentType, s.CheckPermissionJSON).ThenFunc(s.GetAllMessagesJSON)).Methods(http.MethodGet)

	// Mailgun Incoming
	s.Router.HandleFunc("/mg/incoming/{inboxID}/", s.MailgunIncoming).Methods(http.MethodPost)

	return &s, nil
}

// Inbox contains data on a temporary inbox including its address and ttl
type Inbox struct {
	Address        string `dynamodbav:"email_address" json:"address"`
	ID             string `dynamodbav:"id" json:"id"`
	CreatedAt      int64  `dynamodbav:"created_at" json:"created_at"`
	TTL            int64  `dynamodbav:"ttl" json:"ttl"`
	MGRouteID      string `dynamodbav:"mg_routeid" json:"-"`
	FailedToCreate bool   `dynamodbav:"failed_to_create" json:"-"`
}

// NewInbox returns an inbox with failed to create and route id set. Upon successful registration of the mailgun route we set these as true.
func NewInbox() Inbox {
	return Inbox{
		FailedToCreate: true,
		MGRouteID:      "-",
	}
}

// Message contains details of an individual email message received by the server
type Message struct {
	InboxID    string `dynamodbav:"inbox_id" json:"-"`
	ID         string `dynamodbav:"message_id" json:"id"`
	ReceivedAt int64  `dynamodbav:"received_at" json:"received_at"`
	MGID       string `dynamodbav:"mg_id" json:"-"`
	Sender     string `dynamodbav:"sender" json:"sender"`
	From       string `dynamodbav:"from" json:"from"`
	Subject    string `dynamodbav:"subject" json:"subject"`
	BodyHTML   string `dynamodbav:"body_html" json:"body_html"`
	BodyPlain  string `dynamodbav:"body_plain" json:"body_plain"`
	TTL        int64  `dynamodbav:"ttl" json:"ttl"`
}

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

// saveNewInbox saves the passed in inbox to dynamodb
func (s *Server) saveNewInbox(i Inbox) error {
	av, err := dynamodbattribute.MarshalMap(i)

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

// getInboxByID gets an email by id
func (s *Server) getInboxByID(id string) (Inbox, error) {
	var i Inbox

	o, err := s.dynDB.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
		TableName: aws.String("emails"),
	})

	if err != nil {
		return Inbox{}, err
	}

	err = dynamodbattribute.UnmarshalMap(o.Item, &i)

	if err != nil {
		return Inbox{}, err
	}

	return i, nil
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

// setInboxCreated updates dynamodb and sets the email as created and adds a mailgun route
func (s *Server) setInboxCreated(i Inbox) error {
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
				S: aws.String(i.MGRouteID),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(i.ID),
			},
		},
		TableName:        aws.String("emails"),
		UpdateExpression: aws.String("SET #F = :f, #M = :m"),
	}

	_, err := s.dynDB.UpdateItem(u)

	if err != nil {
		return fmt.Errorf("setInboxCreated: failed to mark email as created: %v", err)
	}

	return err
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

//getAllMessagesByInboxID gets all messages in the given inbox
func (s *Server) getAllMessagesByInboxID(i string) ([]Message, error) {
	var m []Message

	qi := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":id": {
				S: aws.String(i),
			},
		},
		KeyConditionExpression: aws.String("inbox_id = :id"),
		TableName:              aws.String("messages"),
	}

	res, err := s.dynDB.Query(qi)

	if err != nil {
		return []Message{}, fmt.Errorf("getAllMessagesByInboxID: failed to query dynamodb: %v", err)
	}

	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &m)

	if err != nil {
		return []Message{}, fmt.Errorf("getAllMessagesByInboxID: failed to unmarshal result: %v", err)
	}

	return m, nil
}
