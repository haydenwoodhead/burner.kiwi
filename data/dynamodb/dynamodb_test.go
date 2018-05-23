package dynamodb

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/haydenwoodhead/burner.kiwi/data"
)

func TestDynamoDB(t *testing.T) {
	sess, err := session.NewSession(&aws.Config{
		Region:   aws.String("us-west-2"),
		Endpoint: aws.String("http://localhost:8000")})

	if err != nil {
		t.Fatalf("DynamoDB: failed to setup db: %v", err)
	}

	dbSvc := dynamodb.New(sess)

	db := &DynamoDB{
		dynDB: dbSvc,
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
