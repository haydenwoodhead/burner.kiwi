package postgresql

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"

	"github.com/haydenwoodhead/burner.kiwi/data"
)

func TestPostgreSQL(t *testing.T) {
	dburl := os.Getenv("DATABASE_URL")
	if dburl == "" {
		t.Fatalf("PostgreSQL: no datbase url set")
	}

	db := GetPostgreSQLDB(dburl)

	// iterate over the testing suite and call the function
	for _, f := range data.TestingFuncs {
		f(t, db)
	}

	testTTLDelete(t, db)
}

func testTTLDelete(t *testing.T, db *PostgreSQL) {
	db.MustExec("DELETE FROM inbox")
	db.MustExec("DELETE FROM message")

	i1 := data.Inbox{
		ID:      uuid.Must(uuid.NewRandom()).String(),
		Address: "hayden@example.com",
		TTL:     time.Now().Add(-1 * time.Hour).Unix(),
	}
	err := db.SaveNewInbox(i1)
	assert.NoError(t, err)

	i2 := data.Inbox{
		ID:      uuid.Must(uuid.NewRandom()).String(),
		Address: "bobby@example.com",
		TTL:     time.Now().Add(1 * time.Hour).Unix(),
	}
	err = db.SaveNewInbox(i2)
	assert.NoError(t, err)

	count, err := db.RunTTLDelete()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	_, err = db.GetInboxByID(i1.ID)
	assert.Equal(t, sql.ErrNoRows, err)
}
