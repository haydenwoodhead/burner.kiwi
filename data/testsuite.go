package data

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/haydenwoodhead/burner.kiwi/burner"
	"github.com/stretchr/testify/assert"
)

// TestFunction is the signature for a testing function
type TestFunction = func(t *testing.T, db burner.Database)

// TestingFuncs contain the suite of funcs that a db implementation should be tested against
var TestingFuncs = []TestFunction{
	TestSaveNewInbox,
	TestGetInboxByID,
	TestGetInboxByAddress,
	TestEmailAddressExists,
	TestSetInboxCreated,
	TestSaveNewMessage,
	TestGetMessageByID,
	TestGetMessagesByInboxID,
}

// TestSaveNewInbox verifies that SaveNewInbox works
func TestSaveNewInbox(t *testing.T, db burner.Database) {
	i := burner.Inbox{
		Address:              "test.1@example.com",
		ID:                   uuid.Must(uuid.NewRandom()).String(),
		CreatedBy:            "192.168.1.1",
		CreatedAt:            time.Now().Unix(),
		TTL:                  time.Now().Add(5 * time.Minute).Unix(),
		EmailProviderRouteID: "-",
		FailedToCreate:       true,
	}

	err := db.SaveNewInbox(i)

	if err != nil {
		t.Errorf("%v - TestSaveNewInbox: failed to save: %v", reflect.TypeOf(db), err)
	}

	ri, err := db.GetInboxByID(i.ID)

	if err != nil {
		t.Errorf("%v - TestSaveNewInbox: failed to get inbox back: %v", reflect.TypeOf(db), err)
	}

	if !reflect.DeepEqual(i, ri) {
		t.Errorf("%v - TestSaveNewInbox: inbox not the same after retrieve. Expected %v got %v", reflect.TypeOf(db), i, ri)
	}
}

// TestGetInboxByID verifies that GetInboxByID works
func TestGetInboxByID(t *testing.T, db burner.Database) {
	i := burner.Inbox{
		Address:              "test.2@example.com",
		ID:                   uuid.Must(uuid.NewRandom()).String(),
		CreatedBy:            "192.168.1.1",
		CreatedAt:            time.Now().Unix(),
		TTL:                  time.Now().Add(5 * time.Minute).Unix(),
		EmailProviderRouteID: "-",
		FailedToCreate:       true,
	}

	err := db.SaveNewInbox(i)

	if err != nil {
		t.Errorf("%v - TestGetInboxByID: failed to save: %v", reflect.TypeOf(db), err)
	}

	ri, err := db.GetInboxByID(i.ID)

	if err != nil {
		t.Errorf("%v - TestGetInboxByID: failed to get inbox back: %v", reflect.TypeOf(db), err)
	}

	if !reflect.DeepEqual(i, ri) {
		t.Errorf("%v - TestGetInboxByID: inbox not the same after retrieve. Expected %v got %v", reflect.TypeOf(db), i, ri)
	}
}

// TestGetInboxByID verifies that GetInboxByID works
func TestGetInboxByAddress(t *testing.T, db burner.Database) {
	i := burner.Inbox{
		Address:              "test.8@example.com",
		ID:                   uuid.Must(uuid.NewRandom()).String(),
		CreatedBy:            "192.168.1.1",
		CreatedAt:            time.Now().Unix(),
		TTL:                  time.Now().Add(5 * time.Minute).Unix(),
		EmailProviderRouteID: "-",
		FailedToCreate:       true,
	}

	err := db.SaveNewInbox(i)

	if err != nil {
		t.Errorf("%v - TestGetInboxByAddress: failed to save: %v", reflect.TypeOf(db), err)
	}

	ri, err := db.GetInboxByAddress(i.Address)

	if err != nil {
		t.Errorf("%v - TestGetInboxByAddress: failed to get inbox back: %v", reflect.TypeOf(db), err)
	}

	if !reflect.DeepEqual(i, ri) {
		t.Errorf("%v - TestGetInboxByAddress: inbox not the same after retrieve. Expected %v got %v", reflect.TypeOf(db), i, ri)
	}
}

// TestEmailAddressExists verifies that EmailAddressExists works
func TestEmailAddressExists(t *testing.T, db burner.Database) {
	i := burner.Inbox{
		Address:              "test.3@example.com",
		ID:                   uuid.Must(uuid.NewRandom()).String(),
		CreatedAt:            time.Now().Unix(),
		CreatedBy:            "192.168.1.1",
		TTL:                  time.Now().Add(5 * time.Minute).Unix(),
		EmailProviderRouteID: "-",
		FailedToCreate:       true,
	}

	err := db.SaveNewInbox(i)

	if err != nil {
		t.Errorf("%v - emailAddressExists: failed to save: %v", reflect.TypeOf(db), err)
	}

	tests := []struct {
		Email  string
		Expect bool
	}{
		{"test.3@example.com", true},
		{"doesntexist@example.com", false},
	}

	for i, test := range tests {
		exists, err := db.EmailAddressExists(test.Email)

		if err != nil {
			t.Errorf("%v - TestEmailAddressExists - %v: failed to check if address exists: %v", reflect.TypeOf(db), i, err)
		}

		if exists != test.Expect {
			t.Errorf("%v - TestEmailAddressExists - %v: Check not correct. Expected %v, got %v", reflect.TypeOf(db), i, test.Expect, exists)
		}
	}
}

//TestSetInboxCreated verifies that SetInboxCreated works
func TestSetInboxCreated(t *testing.T, db burner.Database) {
	i := burner.Inbox{
		Address:              "test.4@example.com",
		ID:                   uuid.Must(uuid.NewRandom()).String(),
		CreatedAt:            time.Now().Unix(),
		CreatedBy:            "192.168.1.1",
		TTL:                  time.Now().Add(5 * time.Minute).Unix(),
		EmailProviderRouteID: "-",
		FailedToCreate:       true,
	}

	err := db.SaveNewInbox(i)

	if err != nil {
		t.Errorf("%v - TestSetInboxCreated: failed to save: %v", reflect.TypeOf(db), err)
	}

	i.EmailProviderRouteID = "mg12345"

	err = db.SetInboxCreated(i)

	if err != nil {
		t.Errorf("%v - TestSetInboxCreated: failed to set inbox created: %v", reflect.TypeOf(db), err)
	}

	ret, err := db.GetInboxByID(i.ID)

	if err != nil {
		t.Errorf("%v - TestSetInboxCreated: failed to get inbox back: %v", reflect.TypeOf(db), err)
	}

	if ret.EmailProviderRouteID != i.EmailProviderRouteID {
		t.Errorf("%v - TestSetInboxCreated: mg route id not same. Expected %v, got %v", reflect.TypeOf(db), i.EmailProviderRouteID, ret.EmailProviderRouteID)
	}

	if ret.FailedToCreate {
		t.Errorf("%v - TestSetInboxCreated: failed to set failedtocreate to false", reflect.TypeOf(db))
	}
}

//TestSaveNewMessage verifies that SaveNewMessage works
func TestSaveNewMessage(t *testing.T, db burner.Database) {
	i := burner.Inbox{
		Address:              "test.5@example.com",
		ID:                   uuid.Must(uuid.NewRandom()).String(),
		CreatedAt:            time.Now().Unix(),
		CreatedBy:            "192.168.1.1",
		TTL:                  time.Now().Add(5 * time.Minute).Unix(),
		EmailProviderRouteID: "-",
		FailedToCreate:       true,
	}

	err := db.SaveNewInbox(i)
	if err != nil {
		t.Fatalf("%v - TestSaveNewMessage: failed to insert new db: %v", reflect.TypeOf(db), err)
	}

	m := burner.Message{
		InboxID:         i.ID,
		ID:              uuid.Must(uuid.NewRandom()).String(),
		ReceivedAt:      time.Now().Unix(),
		EmailProviderID: "56789",
		Sender:          "bob@example.com",
		FromName:        "Bobby Tables",
		FromAddress:     "bob@example.com",
		Subject:         "DELETE FROM MESSAGES;",
		BodyPlain:       "Hello there how are you!",
		BodyHTML:        "<html><body><p>Hello there how are you!</p></body></html>",
		TTL:             time.Now().Add(5 * time.Minute).Unix(),
	}

	err = db.SaveNewMessage(m)

	if err != nil {
		t.Errorf("%v - TestSaveNewMessage: failed to save new message: %v", reflect.TypeOf(db), err)
	}

	ret, err := db.GetMessageByID(m.InboxID, m.ID)
	if err != nil {
		t.Errorf("%v - TestSaveNewMessage: failed to get back new message: %v", reflect.TypeOf(db), err)
	}

	assert.Equal(t, m, ret, "%v - TestSaveNewMessage: saved message not the same as returned.", reflect.TypeOf(db))
}

//TestGetMessageByID verifies that GetMessageByID works
func TestGetMessageByID(t *testing.T, db burner.Database) {
	i := burner.Inbox{
		ID:      uuid.Must(uuid.NewRandom()).String(),
		Address: "test.6@example.com",
	}

	err := db.SaveNewInbox(i)
	if err != nil {
		t.Fatalf("%v - TestGetMessageByID: failed to insert new db: %v", reflect.TypeOf(db), err)
	}

	m := burner.Message{
		InboxID:         i.ID,
		ID:              uuid.Must(uuid.NewRandom()).String(),
		ReceivedAt:      time.Now().Unix(),
		EmailProviderID: "56789",
		Sender:          "bob@example.com",
		FromName:        "Bobby Tables",
		FromAddress:     "bob@example.com",
		Subject:         "DELETE FROM MESSAGES;",
		BodyPlain:       "Hello there how are you!",
		BodyHTML:        "<html><body><p>Hello there how are you!</p></body></html>",
		TTL:             time.Now().Add(5 * time.Minute).Unix(),
	}

	err = db.SaveNewMessage(m)

	if err != nil {
		t.Errorf("%v - TestGetMessageByID: failed to save new message: %v", reflect.TypeOf(db), err)
	}

	tests := []struct {
		InboxID     string
		MessageID   string
		ExpectedRes burner.Message
		ExpectedErr error
	}{
		{
			InboxID:     m.InboxID,
			MessageID:   m.ID,
			ExpectedRes: m,
			ExpectedErr: nil,
		},
		{
			InboxID:     uuid.Must(uuid.NewRandom()).String(), // doesn't exist
			MessageID:   uuid.Must(uuid.NewRandom()).String(),
			ExpectedRes: burner.Message{},
			ExpectedErr: burner.ErrMessageDoesntExist,
		},
	}

	for i, test := range tests {
		ret, err := db.GetMessageByID(test.InboxID, test.MessageID)

		if err != test.ExpectedErr {
			t.Errorf("%v - TestGetMessageByID - %v: error not expected. Expected %v, got %v", reflect.TypeOf(db), i, err, test.ExpectedErr)
		}

		assert.Equalf(t, test.ExpectedRes, ret, "%v - TestGetMessageByID - %v: expected not as same as returned.", reflect.TypeOf(db), i)
	}
}

//TestGetMessagesByInboxID verifies that GetMessagesByInboxID works
//nolint
func TestGetMessagesByInboxID(t *testing.T, db burner.Database) {
	i := burner.Inbox{
		Address:              "test.7@example.com",
		ID:                   "ddb9ec88-2c11-4731-a433-36a04661de83",
		CreatedAt:            time.Now().Unix(),
		CreatedBy:            "192.168.1.1",
		TTL:                  time.Now().Add(5 * time.Minute).Unix(),
		EmailProviderRouteID: "ddb9ec88-2c11-4731-a433-36a04661de83",
		FailedToCreate:       false,
	}

	err := db.SaveNewInbox(i)

	if err != nil {
		t.Errorf("%v - TestGetMessagesByInboxID: failed to save: %v", reflect.TypeOf(db), err)
	}

	m1 := burner.Message{
		InboxID:         "ddb9ec88-2c11-4731-a433-36a04661de83",
		ID:              uuid.Must(uuid.NewRandom()).String(),
		ReceivedAt:      time.Now().Unix(),
		EmailProviderID: "56789",
		FromName:        "Bobby Tables",
		FromAddress:     "bob@example.com",
		Subject:         "DELETE FROM MESSAGES;",
		BodyPlain:       "Hello there how are you!",
		BodyHTML:        "<html><body><p>Hello there how are you!</p></body></html>",
		TTL:             time.Now().Add(5 * time.Minute).Unix(),
	}

	m2 := burner.Message{
		InboxID:         "ddb9ec88-2c11-4731-a433-36a04661de83",
		ID:              uuid.Must(uuid.NewRandom()).String(),
		ReceivedAt:      time.Now().Unix(),
		EmailProviderID: "56789",
		FromName:        "Bobby Tables",
		FromAddress:     "bob@example.com",
		Subject:         "DELETE FROM MESSAGES;",
		BodyPlain:       "Hello there how are you!",
		BodyHTML:        "<html><body><p>Hello there how are you!</p></body></html>",
		TTL:             time.Now().Add(5 * time.Minute).Unix(),
	}

	err = db.SaveNewMessage(m1)

	if err != nil {
		t.Errorf("%v - TestGetMessagesByInboxID: failed to save message 1: %v", reflect.TypeOf(db), err)
	}

	err = db.SaveNewMessage(m2)
	if err != nil {
		t.Errorf("%v - TestGetMessagesByInboxID: failed to save message 2: %v", reflect.TypeOf(db), err)
	}

	messages, err := db.GetMessagesByInboxID(m1.InboxID)
	if err != nil {
		t.Errorf("%v - TestGetMessagesByInboxID: failed to retrieve messages: %v", reflect.TypeOf(db), err)
	}

	assert.ElementsMatch(t, []burner.Message{m1, m2}, messages, "%v - TestGetMessagesByInboxID: Got back a different message than saved", reflect.TypeOf(db))

	// Test that it returns an empty messages slice if there are no messages
	empty, err := db.GetMessagesByInboxID(uuid.Must(uuid.NewRandom()).String())
	if err != nil {
		t.Errorf("%v - TestGetMessagesByInboxID: get empty inbox: %v", reflect.TypeOf(db), err)
	}

	if len(empty) != 0 {
		t.Errorf("%v - TestGetMessagesByInboxID: returned messages for a non existent key", reflect.TypeOf(db))
	}
}
