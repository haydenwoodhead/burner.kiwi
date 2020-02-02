package sqlite3

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/haydenwoodhead/burner.kiwi/server"
	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"

	"github.com/haydenwoodhead/burner.kiwi/data"
)

func TestSQLite3(t *testing.T) {
	db := GetSQLite3DB("test.sqlite3")

	// iterate over the testing suite and call the function
	for _, f := range data.TestingFuncs {
		f(t, db)
	}

	testTTLDelete(t, db)

	// remove test database
	err := os.Remove("test.sqlite3")
	if err != nil {
		t.Fatalf("SQLite3: failed to delete test database file")
	}
}

func testTTLDelete(t *testing.T, db *SQLite3) {
	db.MustExec("DELETE FROM inbox")
	db.MustExec("DELETE FROM message")

	i1 := server.Inbox{
		ID:      uuid.Must(uuid.NewRandom()).String(),
		Address: "hayden@example.com",
		TTL:     time.Now().Add(-1 * time.Hour).Unix(),
	}
	err := db.SaveNewInbox(i1)
	assert.NoError(t, err)

	i2 := server.Inbox{
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
