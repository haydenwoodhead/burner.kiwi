package postgresql

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/haydenwoodhead/burner.kiwi/server"
	"github.com/jmoiron/sqlx"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"

	"github.com/haydenwoodhead/burner.kiwi/data"
)

var dbURL string

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.Run("postgres", "11.3", []string{"POSTGRES_PASSWORD=password"})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	dbURL = fmt.Sprintf("postgresql://postgres:password@localhost:%s/postgres?sslmode=disable", resource.GetPort("5432/tcp"))

	if err := pool.Retry(func() error {
		db, err := sqlx.Connect("postgres", dbURL)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	code := m.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestPostgreSQL(t *testing.T) {
	db := GetPostgreSQLDB(dbURL)

	// iterate over the testing suite and call the function
	for _, f := range data.TestingFuncs {
		f(t, db)
	}

	testTTLDelete(t, db)
}

func testTTLDelete(t *testing.T, db *PostgreSQL) {
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
