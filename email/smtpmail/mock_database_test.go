package smtpmail

import (
	"github.com/haydenwoodhead/burner.kiwi/burner"
	mock "github.com/stretchr/testify/mock"
)

type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDatabase) EmailAddressExists(address string) (bool, error) {
	args := m.Called(address)
	return args.Bool(0), args.Error(1)
}

func (m *MockDatabase) GetInboxByAddress(address string) (burner.Inbox, error) {
	args := m.Called(address)
	return args.Get(0).(burner.Inbox), args.Error(1)
}

func (m *MockDatabase) GetInboxByID(id string) (burner.Inbox, error) {
	args := m.Called(id)
	return args.Get(0).(burner.Inbox), args.Error(1)
}

func (m *MockDatabase) GetMessageByID(inboxID string, messageID string) (burner.Message, error) {
	args := m.Called(inboxID, messageID)
	return args.Get(0).(burner.Message), args.Error(1)
}

func (m *MockDatabase) GetMessagesByInboxID(id string) ([]burner.Message, error) {
	args := m.Called(id)
	return args.Get(0).([]burner.Message), args.Error(1)
}

func (m *MockDatabase) SaveNewInbox(inbox burner.Inbox) error {
	args := m.Called(inbox)
	return args.Error(0)
}

func (m *MockDatabase) SaveNewMessage(message burner.Message) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockDatabase) SetInboxCreated(inbox burner.Inbox) error {
	args := m.Called(inbox)
	return args.Error(0)
}

func (m *MockDatabase) SetInboxFailed(inbox burner.Inbox) error {
	args := m.Called(inbox)
	return args.Error(0)
}
