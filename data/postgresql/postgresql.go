package postgresql

import (
	"database/sql"
	"fmt"
	"log"
	"time"

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
	p := &PostgreSQL{sqlx.MustOpen("postgres", dbURL)}
	go func() {
		for {
			time.Sleep(30 * time.Minute)
			count, err := p.runTTLDelete()
			if err != nil {
				log.Println("Failed to delete old rows from db")
				continue
			}
			log.Printf("Deleted %v old inboxes from db\n", count)
		}
	}()
	return p
}

// SaveNewInbox saves a new inbox
func (p *PostgreSQL) SaveNewInbox(i data.Inbox) error {
	_, err := p.NamedExec(
		"INSERT INTO inbox (id, address, created_at, created_by, mg_routeid, ttl, failed_to_create) VALUES (:id, :address, :created_at, :created_by, :mg_routeid, :ttl, :failed_to_create)",
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
	err := p.Get(&i, "SELECT id, address, created_at, created_by, mg_routeid, ttl, failed_to_create FROM inbox WHERE id = $1", id)
	return i, err
}

// EmailAddressExists checks if an address already exists
func (p *PostgreSQL) EmailAddressExists(email string) (bool, error) {
	var count int
	err := p.Get(&count, "SELECT COUNT(*) FROM inbox WHERE address = $1", email)
	return count == 1, err
}

// SetInboxCreated creates a new inbox
func (p *PostgreSQL) SetInboxCreated(i data.Inbox) error {
	_, err := p.Exec("UPDATE inbox SET failed_to_create = 'false', mg_routeid = $1 WHERE id = $2", i.MGRouteID, i.ID)
	return err
}

// SaveNewMessage saves a new message to the db
func (p *PostgreSQL) SaveNewMessage(m data.Message) error {
	_, err := p.NamedExec("INSERT INTO message (inbox_id, message_id, received_at, mg_id, sender, from_address, subject, body_html, body_plain, ttl) VALUES (:inbox_id, :message_id, :received_at, :mg_id, :sender, :from_address, :subject, :body_html, :body_plain, :ttl)",
		map[string]interface{}{
			"inbox_id":     m.InboxID,
			"message_id":   m.ID,
			"received_at":  m.ReceivedAt,
			"mg_id":        m.MGID,
			"sender":       m.Sender,
			"from_address": m.From,
			"subject":      m.Subject,
			"body_html":    m.BodyHTML,
			"body_plain":   m.BodyPlain,
			"ttl":          m.TTL,
		},
	)
	return err
}

// GetMessagesByInboxID gets all messages for an inbox
func (p *PostgreSQL) GetMessagesByInboxID(id string) ([]data.Message, error) {
	var msgs []data.Message
	err := p.Select(&msgs, "SELECT inbox_id, message_id, received_at, mg_id, sender, from_address, subject, body_html, body_plain, ttl FROM message WHERE inbox_id = $1", id)
	return msgs, err
}

// GetMessageByID gets a single message
func (p *PostgreSQL) GetMessageByID(i, m string) (data.Message, error) {
	var msg data.Message
	err := p.Get(&msg, "SELECT inbox_id, message_id, received_at, mg_id, sender, from_address, subject, body_html, body_plain, ttl FROM message WHERE inbox_id = $1 and message_id = $2", i, m)
	if err == sql.ErrNoRows {
		return msg, data.ErrMessageDoesntExist
	}

	return msg, err
}

func (p *PostgreSQL) runTTLDelete() (int, error) {
	t := time.Now().Unix()
	res, err := p.Exec("DELETE from inbox WHERE ttl < $1", t)
	if err != nil {
		return -1, fmt.Errorf("Postgres.runTTLDelete failed with err=%v", err)
	}
	count, err := res.RowsAffected()
	return int(count), err
}
