package sqldb

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/haydenwoodhead/burner.kiwi/burner"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

// SQLDatabase implements the database interface for sqldb
type SQLDatabase struct {
	*sqlx.DB
	dbType string
}

// New returns a new db or panics
func New(dbType string, dbURL string) *SQLDatabase {
	s := &SQLDatabase{sqlx.MustOpen(dbType, dbURL), dbType}
	return s
}

func (s *SQLDatabase) Start() error {
	log.Infof("Starting %s database connection", s.dbType)

	err := s.createTables()
	if err != nil {
		return fmt.Errorf("%s - failed to create tables: %w", s.dbType, err)
	}

	go func() {
		t := time.Now().Unix()
		var active int
		err := s.Get(&active, "select count(*) from inbox WHERE ttl > $1", t)
		if err != nil {
			log.WithError(err).Error("Failed to get number of active inboxes")
		}
		log.WithField("active", active).Info("Got count of active inboxes")

		for {
			count, err := s.RunTTLDelete()
			if err != nil {
				log.WithError(err).Error("Failed to delete old rows from db")
				break
			}
			log.WithField("deleted", count).Info("Deleted old inboxes from db")
			time.Sleep(1 * time.Hour)
		}
	}()

	return nil
}

// createTables creates the databse tables or panics
func (s *SQLDatabase) createTables() error {
	_, err := s.Exec(`create table if not exists inbox (
		id uuid not null unique,
		address text not null unique,
		created_at numeric,
		created_by text,
		ep_routeid text,
		ttl numeric,
		failed_to_create bool,
		primary key (id)
	);
	
	create table if not exists message (
		inbox_id uuid references inbox(id) on delete cascade,
		message_id uuid not null unique,
		received_at numeric,
		ep_id text,
		sender text,
		from_name text,
		from_address text,
		subject text,
		body_html text,
		body_plain text,
		ttl numeric,
		primary key (message_id)
	);`)
	return err
}

// SaveNewInbox saves a new inbox
func (s *SQLDatabase) SaveNewInbox(i burner.Inbox) error {
	_, err := s.NamedExec(
		"INSERT INTO inbox (id, address, created_at, created_by, ep_routeid, ttl, failed_to_create) VALUES (:id, :address, :created_at, :created_by, :ep_routeid, :ttl, :failed_to_create)",
		map[string]interface{}{
			"id":               i.ID,
			"address":          i.Address,
			"created_at":       i.CreatedAt,
			"created_by":       i.CreatedBy,
			"ep_routeid":       i.EmailProviderRouteID,
			"ttl":              i.TTL,
			"failed_to_create": i.FailedToCreate,
		},
	)

	return err
}

// GetInboxByID gets an inbox by id
func (s *SQLDatabase) GetInboxByID(id string) (burner.Inbox, error) {
	var i burner.Inbox
	err := s.Get(&i, "SELECT id, address, created_at, created_by, ep_routeid, ttl, failed_to_create FROM inbox WHERE id = $1", id)
	return i, err
}

// GetInboxByAddress gets an inbox by address
func (s *SQLDatabase) GetInboxByAddress(address string) (burner.Inbox, error) {
	var i burner.Inbox
	err := s.Get(&i, "SELECT id, address, created_at, created_by, ep_routeid, ttl, failed_to_create FROM inbox WHERE address = $1", address)
	return i, err
}

// EmailAddressExists checks if an address already exists
func (s *SQLDatabase) EmailAddressExists(email string) (bool, error) {
	var count int
	err := s.Get(&count, "SELECT COUNT(*) FROM inbox WHERE address = $1", email)
	return count > 0, err
}

// SetInboxCreated creates a new inbox
func (s *SQLDatabase) SetInboxCreated(i burner.Inbox) error {
	_, err := s.Exec("UPDATE inbox SET failed_to_create = 'false', ep_routeid = $1 WHERE id = $2", i.EmailProviderRouteID, i.ID)
	return err
}

// SetInboxFailed sets a given inbox as having failed to register with the mail provider
func (s *SQLDatabase) SetInboxFailed(i burner.Inbox) error {
	_, err := s.Exec("UPDATE inbox SET failed_to_create = 'true' WHERE id = $1", i.ID)
	return err
}

// SaveNewMessage saves a new message to the db
func (s *SQLDatabase) SaveNewMessage(m burner.Message) error {
	_, err := s.NamedExec("INSERT INTO message (inbox_id, message_id, received_at, ep_id, sender, from_name, from_address, subject, body_html, body_plain, ttl) VALUES (:inbox_id, :message_id, :received_at, :ep_id, :sender, :from_name, :from_address, :subject, :body_html, :body_plain, :ttl)",
		map[string]interface{}{
			"inbox_id":     m.InboxID,
			"message_id":   m.ID,
			"received_at":  m.ReceivedAt,
			"ep_id":        m.EmailProviderID,
			"sender":       m.Sender,
			"from_name":    m.FromName,
			"from_address": m.FromAddress,
			"subject":      m.Subject,
			"body_html":    m.BodyHTML,
			"body_plain":   m.BodyPlain,
			"ttl":          m.TTL,
		},
	)
	return err
}

// GetMessagesByInboxID gets all messages for an inbox
func (s *SQLDatabase) GetMessagesByInboxID(id string) ([]burner.Message, error) {
	var msgs []burner.Message
	err := s.Select(&msgs, "SELECT * FROM message WHERE inbox_id = $1", id)
	return msgs, err
}

// GetMessageByID gets a single message
func (s *SQLDatabase) GetMessageByID(i, m string) (burner.Message, error) {
	var msg burner.Message
	err := s.Get(&msg, "SELECT * FROM message WHERE inbox_id = $1 and message_id = $2", i, m)
	if err == sql.ErrNoRows {
		return msg, burner.ErrMessageDoesntExist
	}

	return msg, err
}

// RunTTLDelete runs the TTL delete process
func (s *SQLDatabase) RunTTLDelete() (int, error) {
	t := time.Now().Unix()
	res, err := s.Exec("DELETE from inbox WHERE ttl < $1", t)
	if err != nil {
		return -1, fmt.Errorf("%s - failed to delete expired inboxes: %w", s.dbType, err)
	}
	count, err := res.RowsAffected()
	return int(count), err
}
