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

// GetPostgreSQLDB returns a new postgres db or panics
func GetPostgreSQLDB(dbURL string) *PostgreSQL {
	db := sqlx.MustOpen("postgres", dbURL)
	return &PostgreSQL{db}
}

// SaveNewInbox saves a new inbox
func (p *PostgreSQL) SaveNewInbox(i data.Inbox) error {
	_, err := p.NamedExec(
		"INSERT INTO inbox VALUES (:id, :address, :created_at, :created_by, :mg_routeid, :ttl, :failed_to_create)",
		map[string]interface{}{
			"id":               i.ID,
			"address":          i.Address,
			"created_at":       i.CreatedAt,
			"created_by":       i.CreatedBy,
			"mg_routeid":       i.MGRouteID,
			"ttl":              i.TTL,
			"failed_to_create": i.FailedToCreate,
		},
	)
	return err
}

// GetInboxByID gets an inbox by id
func (p *PostgreSQL) GetInboxByID(id string) (data.Inbox, error) {
	var i data.Inbox
	err := p.Get(&i, "SELECT * FROM inbox WHERE id = $1", id)
	return i, err
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
