package server

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
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
