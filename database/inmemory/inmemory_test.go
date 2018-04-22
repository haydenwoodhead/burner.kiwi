package inmemory

import (
	"testing"

	"github.com/haydenwoodhead/burnerkiwi/database"
)

func TestInMemoryDB(t *testing.T) {
	db := GetInMemoryDB()

	// iterate over the testing suite and call the function
	for _, f := range database.TestingFuncs {
		f(t, db)
	}
}
