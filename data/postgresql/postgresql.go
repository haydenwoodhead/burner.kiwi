package postgresql

import (
	"github.com/haydenwoodhead/burner.kiwi/data/sqldb"

	_ "github.com/lib/pq" // import lib pq here rather than main
)

// PostgreSQL implements the database interface for postgres
type PostgreSQL struct {
	*sqldb.SQLDatabase
}

// GetPostgreSQLDB returns a new postgres db or panics
func GetPostgreSQLDB(dbURL string) *PostgreSQL {
	return &PostgreSQL{sqldb.GetDatabase("postgres", dbURL)}
}
