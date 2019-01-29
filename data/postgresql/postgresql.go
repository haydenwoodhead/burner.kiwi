package postgresql

import (
	"github.com/haydenwoodhead/burner.kiwi/data"
	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq" // import lib pq here rather than main
)

// PostgreSQL implements the database interface for postgres
type PostgreSQL struct {
	*sqlx.DB
}

// SaveNewInbox saves a new inbox
func (p *PostgreSQL) SaveNewInbox(data.Inbox) error {
	panic("implement me")
}

// GetInboxByID gets an inbox by id
func (p *PostgreSQL) GetInboxByID(string) (data.Inbox, error) {
	panic("implement me")
}

// EmailAddressExists checks if an address already exists
func (p *PostgreSQL) EmailAddressExists(string) (bool, error) {
	panic("implement me")
}

// SetInboxCreated creates a new inbox
func (p *PostgreSQL) SetInboxCreated(data.Inbox) error {
	panic("implement me")
}

// SaveNewMessage saves a new message to the db
func (p *PostgreSQL) SaveNewMessage(data.Message) error {
	panic("implement me")
}

// GetMessagesByInboxID gets all messages for an inbox
func (p *PostgreSQL) GetMessagesByInboxID(string) ([]data.Message, error) {
	panic("implement me")
}

// GetMessageByID gets a single message
func (p *PostgreSQL) GetMessageByID(string, string) (data.Message, error) {
	panic("implement me")
}
