package inmemory

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/haydenwoodhead/burner.kiwi/server"
)

var _ server.Database = &InMemory{}

var errInboxDoesntExist = errors.New("failed to get inbox. It doesn't exist")

// InMemory implements an in memory database
type InMemory struct {
	emails   map[string]server.Inbox
	messages map[string]map[string]server.Message
	m        sync.Mutex
}

// GetInMemoryDB returns a new InMemoryDB to use
func GetInMemoryDB() *InMemory {
	im := &InMemory{}

	im.messages = make(map[string]map[string]server.Message)
	im.emails = make(map[string]server.Inbox)

	// launch a func which deletes expired objects
	go func(im *InMemory) {
		for {
			time.Sleep(6 * time.Hour)
			im.DeleteExpiredData()
		}
	}(im)

	return im
}

// DeleteExpiredData deletes data that has expired according to its TTL
func (im *InMemory) DeleteExpiredData() {
	im.m.Lock()

	for k, v := range im.emails {
		t := time.Unix(v.TTL, 0)

		// if our emails ttl is before now then delete it
		if t.Before(time.Now()) {
			delete(im.emails, k)
		}
	}

	for iK, iV := range im.messages {
		for k, v := range iV {
			t := time.Unix(v.TTL, 0)

			if t.Before(time.Now()) {
				delete(im.messages[iK], k)
			}
		}

		if len(iV) == 0 {
			delete(im.messages, iK)
		}
	}

	im.m.Unlock()
}

// SaveNewInbox saves a given inbox to memory
func (im *InMemory) SaveNewInbox(i server.Inbox) error {
	im.m.Lock()
	defer im.m.Unlock()

	im.emails[i.ID] = i

	if im.messages[i.ID] == nil {
		im.messages[i.ID] = make(map[string]server.Message)
	}

	return nil
}

//GetInboxByID gets an inbox by the given inbox id
func (im *InMemory) GetInboxByID(id string) (server.Inbox, error) {
	im.m.Lock()
	defer im.m.Unlock()

	i, ok := im.emails[id]

	if !ok {
		return server.Inbox{}, errInboxDoesntExist
	}

	return i, nil
}

//EmailAddressExists returns a bool depending on whether or not the given email address
// is already assigned to an inbox
func (im *InMemory) EmailAddressExists(a string) (bool, error) {
	im.m.Lock()
	defer im.m.Unlock()

	for _, v := range im.emails {
		if strings.Compare(a, v.Address) == 0 {
			return true, nil
		}
	}

	return false, nil
}

// SetInboxCreated updates the given inbox to reflect its created status
func (im *InMemory) SetInboxCreated(i server.Inbox) error {
	im.m.Lock()
	defer im.m.Unlock()

	i.FailedToCreate = false
	im.emails[i.ID] = i

	return nil
}

// SetInboxFailed sets a given inbox as having failed to register with the mail provider
func (im *InMemory) SetInboxFailed(i server.Inbox) error {
	im.m.Lock()
	defer im.m.Unlock()

	i.FailedToCreate = true
	im.emails[i.ID] = i

	return nil
}

//SaveNewMessage saves a given message to memory
func (im *InMemory) SaveNewMessage(m server.Message) error {
	im.m.Lock()
	defer im.m.Unlock()

	if im.messages[m.InboxID] == nil {
		im.messages[m.InboxID] = make(map[string]server.Message)
	}

	im.messages[m.InboxID][m.ID] = m

	return nil
}

//GetMessagesByInboxID returns all messages in a given inbox
func (im *InMemory) GetMessagesByInboxID(id string) ([]server.Message, error) {
	im.m.Lock()
	defer im.m.Unlock()

	msgs, ok := im.messages[id]

	if !ok {
		return []server.Message{}, nil
	}

	var msgsSlice []server.Message

	for _, v := range msgs {
		msgsSlice = append(msgsSlice, v)
	}

	return msgsSlice, nil
}

//GetMessageByID gets a single message by the given inbox and message id
func (im *InMemory) GetMessageByID(i, m string) (server.Message, error) {
	im.m.Lock()
	defer im.m.Unlock()

	msg, ok := im.messages[i][m]

	if !ok {
		return server.Message{}, server.ErrMessageDoesntExist
	}

	return msg, nil
}
