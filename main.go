package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/apex/gateway"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/haydenwoodhead/burnerkiwi/generateEmail"
	"gopkg.in/mailgun/mailgun-go.v1"
)

var store sessions.Store
var websiteURL string
var eg *generateEmail.EmailGenerator
var mg mailgun.Mailgun
var dynDB *dynamodb.DynamoDB

type Email struct {
	Address        string `dynamodbav:"email_address"`
	ID             string `dynamodbav:"id"`
	CreatedAt      int64  `dynamodbav:"created_at"`
	TTL            int64  `dynamodbav:"ttl"`
	MGRouteID      string `dynamodbav:"mg_routeid"`
	FailedToCreate bool   `dynamodbav:"failed_to_create"`
}

func main() {
	useLambda, err := strconv.ParseBool(os.Getenv("LAMBDA"))

	if err != nil {
		log.Fatalf("Failed to parse LAMBDA env var. Err = %v", err)
	}

	key := os.Getenv("KEY")

	if strings.Compare(key, "") == 0 {
		log.Fatalf("Env var key cannot be empty")
	}

	store = sessions.NewCookieStore([]byte(key))

	websiteURL = os.Getenv("WEBSITE_URL")

	if strings.Compare(websiteURL, "") == 0 {
		log.Fatalf("Env var WEBSITE_URL cannot be empty")
	}

	mgKey := os.Getenv("MG_KEY")

	if strings.Compare(mgKey, "") == 0 {
		log.Fatalf("Env var MG_KEY cannot be empty")
	}

	mgDomain := os.Getenv("MG_DOMAIN")

	if strings.Compare(mgKey, "") == 0 {
		log.Fatalf("Env var MG_KEY cannot be empty")
	}

	mg = mailgun.NewMailgun(mgDomain, mgKey, "")

	eg = generateEmail.NewEmailGenerator([]string{"example.com"}, key, 8)

	awsSession := session.Must(session.NewSession())
	dynDB = dynamodb.New(awsSession)

	r := mux.NewRouter()
	r.HandleFunc("/", Index)
	//r.HandleFunc("/.json", )
	//r.HandleFunc("/inbox/{address}", InboxHTML)
	//r.HandleFunc("/inbox/{address}.json", InboxJSON)

	if useLambda {
		log.Fatal(gateway.ListenAndServe("", context.ClearHandler(r))) // wrap mux in ClearHandler as per docs to prevent leaking memory
	} else {
		log.Fatal(http.ListenAndServe(":8080", context.ClearHandler(r)))
	}
}

// Index checks to see if a session already exists for the user. If so it redirects them to their page otherwise
// it generates a new email address for them and then redirects.
func Index(w http.ResponseWriter, r *http.Request) {
	sess, _ := store.Get(r, "session")

	if !sess.IsNew {
		id, ok := sess.Values["email_id"].(string)

		if !ok {
			log.Printf("Index: session id value not a string")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, websiteURL+"/inbox/"+id, 302)
	}

	var e Email

	addr, err := eg.NewRandom()

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
	go func(e Email) {
		err := e.createRoute()

		if err != nil {
			log.Println(err)
		}

		err = e.putToDB()

		if err != nil {
			log.Println(err)
		}
	}(e)

	sess.Values["email_id"] = e.ID
	sess.Values["email"] = e.Address
	sess.Values["ttl"] = e.TTL
	sess.Save(r, w)
	http.Redirect(w, r, websiteURL+"/inbox/"+id.String(), 302)
}

func InboxHTML(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("success"))
}

func InboxJSON(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("success json"))
}

func (e *Email) createRoute() error {
	routeAddr := websiteURL + "/inbox/" + e.ID + "/"

	route, err := mg.CreateRoute(mailgun.Route{
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

func (e *Email) putToDB() error {
	av, err := dynamodbattribute.MarshalMap(e)

	if err != nil {
		return fmt.Errorf("putToDB: failed to marshal struct to attribute value: %v", err)

	}

	_, err = dynDB.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("emails"),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("putToDB: failed to put to dynamodb: %v", err)
	}

	return nil
}
