package burner

import "errors"

// ErrMessageDoesntExist is returned by GetMessagesByID when it cant find that specific message
var ErrMessageDoesntExist = errors.New("message doesn't exist")

// Database lists methods needed to implement a db
type Database interface {
	// Start is where you should do schema creation and launch gorountines for background operations
	Start() error
	SaveNewInbox(inbox Inbox) error
	GetInboxByID(id string) (Inbox, error)
	GetInboxByAddress(address string) (Inbox, error)
	EmailAddressExists(address string) (bool, error)
	SetInboxCreated(inbox Inbox) error
	SetInboxFailed(inbox Inbox) error
	SaveNewMessage(message Message) error
	GetMessagesByInboxID(id string) ([]Message, error)
	GetMessageByID(inboxID string, messageID string) (Message, error)
}
