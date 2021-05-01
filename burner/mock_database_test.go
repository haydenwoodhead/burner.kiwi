package burner

import mock "github.com/stretchr/testify/mock"

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

func (m *MockDatabase) GetInboxByAddress(address string) (Inbox, error) {
	args := m.Called(address)
	return args.Get(0).(Inbox), args.Error(1)
}

func (m *MockDatabase) GetInboxByID(id string) (Inbox, error) {
	args := m.Called(id)
	return args.Get(0).(Inbox), args.Error(1)
}

func (m *MockDatabase) GetMessageByID(inboxID string, messageID string) (Message, error) {
	args := m.Called(inboxID, messageID)
	return args.Get(0).(Message), args.Error(1)
}

func (m *MockDatabase) GetMessagesByInboxID(id string) ([]Message, error) {
	args := m.Called(id)
	return args.Get(0).([]Message), args.Error(1)
}

func (m *MockDatabase) SaveNewInbox(inbox Inbox) error {
	args := m.Called(inbox)
	return args.Error(0)
}

func (m *MockDatabase) SaveNewMessage(message Message) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockDatabase) SetInboxCreated(inbox Inbox) error {
	args := m.Called(inbox)
	return args.Error(0)
}

func (m *MockDatabase) SetInboxFailed(inbox Inbox) error {
	args := m.Called(inbox)
	return args.Error(0)
}
