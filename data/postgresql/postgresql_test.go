package postgresql

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/haydenwoodhead/burner.kiwi/data"
)

func TestPostgreSQL(t *testing.T) {
	dburl := os.Getenv("DATABASE_URL")
	if dburl == "" {
		t.Fatalf("PostgreSQL: no datbase url set")
	}

	db := GetPostgreSQLDB(dburl)

	fb, err := ioutil.ReadFile("schema.sql")
	if err != nil {
		t.Fatalf("PostgreSQL: failed to read schema file")
	}

	db.MustExec(string(fb))

	// iterate over the testing suite and call the function
	for _, f := range data.TestingFuncs {
		f(t, db)
	}
}
