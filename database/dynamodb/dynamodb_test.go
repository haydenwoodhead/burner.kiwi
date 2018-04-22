package dynamodb

import (
	"testing"

	"github.com/haydenwoodhead/burnerkiwi/database"
)

func TestDynamoDB(t *testing.T) {
	db := GetNewDynamoDB()

	// iterate over the testing suite and call the function
	for _, f := range database.TestingFuncs {
		f(t, db)
	}
}
