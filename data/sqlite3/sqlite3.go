package sqlite3

import (
	"github.com/haydenwoodhead/burner.kiwi/data/sqldb"

	_ "github.com/mattn/go-sqlite3" // import go-sqlite3 here rather than main
)

// SQLite3 implements the database interface for sqlite3
type SQLite3 struct {
	*sqldb.SQLDatabase
}

// GetSQLite3DB returns a new postgres db or panics
func GetSQLite3DB(dbURL string) *SQLite3 {
	return &SQLite3{sqldb.New("sqlite3", dbURL)}
}
