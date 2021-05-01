package dynamodb

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/haydenwoodhead/burner.kiwi/data"
	"github.com/ory/dockertest"
)

var dynamoDBAddress string

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.Run("amazon/dynamodb-local", "1.11.477", []string{})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	dynamoDBAddress = fmt.Sprintf("http://localhost:%s", resource.GetPort("8000/tcp"))

	if err := pool.Retry(func() error {
		client := http.Client{
			Timeout: 10 * time.Second,
		}
		resp, err := client.Get(dynamoDBAddress)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	code := m.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestDynamoDB(t *testing.T) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials("id", "secret", "token"),
		Region:      aws.String("us-west-2"),
		Endpoint:    aws.String(dynamoDBAddress)},
	)
	if err != nil {
		t.Fatalf("DynamoDB: failed to setup db: %v", err)
	}

	dbSvc := dynamodb.New(sess)

	db := &DynamoDB{
		dynDB:                 dbSvc,
		emailAddressIndexName: "email_address-index",
		emailsTableName:       "emails",
	}

	err = db.createDatabase()
	if err != nil {
		t.Fatalf("DynamoDB: failed to setup db: %v", err)
	}

	// iterate over the testing suite and call the function
	for _, f := range data.TestingFuncs {
		f(t, db)
	}
}
