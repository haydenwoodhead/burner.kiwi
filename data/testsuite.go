package data

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestFunction is the signature for a testing function
type TestFunction = func(t *testing.T, db Database)

// TestingFuncs contain the suite of funcs that a db implementation should be tested against
var TestingFuncs = []TestFunction{
	TestSaveNewInbox,
	TestGetInboxByID,
	TestEmailAddressExists,
	TestSetInboxCreated,
	TestSaveNewMessage,
	TestGetMessageByID,
	TestGetMessagesByInboxID,
}

// TestSaveNewInbox verifies that SaveNewInbox works
func TestSaveNewInbox(t *testing.T, db Database) {
	i := Inbox{
		Address:        "test@example.com",
		ID:             "1234",
		CreatedAt:      time.Now().Unix(),
		TTL:            time.Now().Add(5 * time.Minute).Unix(),
		MGRouteID:      "-",
		FailedToCreate: true,
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
func TestGetInboxByID(t *testing.T, db Database) {
	i := Inbox{
		Address:        "test@example.com",
		ID:             "1234",
		CreatedAt:      time.Now().Unix(),
		TTL:            time.Now().Add(5 * time.Minute).Unix(),
		MGRouteID:      "-",
		FailedToCreate: true,
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

// TestEmailAddressExists verifies that EmailAddressExists works
func TestEmailAddressExists(t *testing.T, db Database) {
	i := Inbox{
		Address:        "test.1@example.com",
		ID:             "5678",
		CreatedAt:      time.Now().Unix(),
		TTL:            time.Now().Add(5 * time.Minute).Unix(),
		MGRouteID:      "-",
		FailedToCreate: true,
	}

	err := db.SaveNewInbox(i)

	if err != nil {
		t.Errorf("%v - emailAddressExists: failed to save: %v", reflect.TypeOf(db), err)
	}

	tests := []struct {
		Email  string
		Expect bool
	}{
		{"test.1@example.com", true},
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
func TestSetInboxCreated(t *testing.T, db Database) {
	i := Inbox{
		Address:        "test2@example.com",
		ID:             "9101112",
		CreatedAt:      time.Now().Unix(),
		TTL:            time.Now().Add(5 * time.Minute).Unix(),
		MGRouteID:      "-",
		FailedToCreate: true,
	}

	err := db.SaveNewInbox(i)

	if err != nil {
		t.Errorf("%v - TestSetInboxCreated: failed to save: %v", reflect.TypeOf(db), err)
	}

	i.MGRouteID = "mg12345"

	err = db.SetInboxCreated(i)

	if err != nil {
		t.Errorf("%v - TestSetInboxCreated: failed to set inbox created: %v", reflect.TypeOf(db), err)
	}

	ret, err := db.GetInboxByID(i.ID)

	if err != nil {
		t.Errorf("%v - TestSetInboxCreated: failed to get inbox back: %v", reflect.TypeOf(db), err)
	}

	if strings.Compare(ret.MGRouteID, i.MGRouteID) != 0 {
		t.Errorf("%v - TestSetInboxCreated: mg route id not same. Expected %v, got %v", reflect.TypeOf(db), i.MGRouteID, ret.MGRouteID)
	}

	if ret.FailedToCreate {
		t.Errorf("%v - TestSetInboxCreated: failed to set failedtocreate to false", reflect.TypeOf(db))
	}
}

//TestSaveNewMessage verifies that SaveNewMessage works
func TestSaveNewMessage(t *testing.T, db Database) {
	m := Message{
		InboxID:    "1234",
		ID:         "5678",
		ReceivedAt: time.Now().Unix(),
		MGID:       "56789",
		Sender:     "bob@example.com",
		From:       "Bobby Tables <bob@example.com>",
		Subject:    "DELETE FROM MESSAGES;",
		BodyPlain:  "Hello there how are you!",
		BodyHTML:   "<html><body><p>Hello there how are you!</p></body></html>",
		TTL:        time.Now().Add(5 * time.Minute).Unix(),
	}

	err := db.SaveNewMessage(m)

	if err != nil {
		t.Errorf("%v - TestSaveNewMessage: failed to save new message: %v", reflect.TypeOf(db), err)
	}

	ret, err := db.GetMessageByID(m.InboxID, m.ID)

	if err != nil {
		t.Errorf("%v - TestSaveNewMessage: failed to get back new message: %v", reflect.TypeOf(db), err)
	}

	if !reflect.DeepEqual(ret, m) {
		t.Errorf("%v - TestSaveNewMessage: saved message not the same as returned. Expected %v, got %v", reflect.TypeOf(db), m, ret)
	}
}

//TestGetMessageByID verifies that GetMessageByID works
func TestGetMessageByID(t *testing.T, db Database) {
	m := Message{
		InboxID:    "9101112",
		ID:         "5678",
		ReceivedAt: time.Now().Unix(),
		MGID:       "56789",
		Sender:     "bob@example.com",
		From:       "Bobby Tables <bob@example.com>",
		Subject:    "DELETE FROM MESSAGES;",
		BodyPlain:  "Hello there how are you!",
		BodyHTML:   "<html><body><p>Hello there how are you!</p></body></html>",
		TTL:        time.Now().Add(5 * time.Minute).Unix(),
	}

	err := db.SaveNewMessage(m)

	if err != nil {
		t.Errorf("%v - TestGetMessageByID: failed to save new message: %v", reflect.TypeOf(db), err)
	}

	tests := []struct {
		InboxID     string
		MessageID   string
		ExpectedRes Message
		ExpectedErr error
	}{
		{
			InboxID:     m.InboxID,
			MessageID:   m.ID,
			ExpectedRes: m,
			ExpectedErr: nil,
		},
		{
			InboxID:     "000000",
			MessageID:   "00000",
			ExpectedRes: Message{},
			ExpectedErr: ErrMessageDoesntExist,
		},
	}

	for i, test := range tests {
		ret, err := db.GetMessageByID(test.InboxID, test.MessageID)

		if err != test.ExpectedErr {
			t.Errorf("%v - TestGetMessageByID - %v: error not expected. Expected %v, got %v", reflect.TypeOf(db), i, err, test.ExpectedErr)
		}

		if !reflect.DeepEqual(ret, test.ExpectedRes) {
			t.Errorf("%v - TestGetMessageByID - %v: expected not as same as returned. Expected %v, got %v", reflect.TypeOf(db), i, test.ExpectedRes, ret)
		}
	}
}

//TestGetMessagesByInboxID verifies that GetMessagesByInboxID works
//nolint
func TestGetMessagesByInboxID(t *testing.T, db Database) {
	i := Inbox{
		Address:        "ddb9ec88-2c11-4731-a433-36a04661de83@example.com",
		ID:             "ddb9ec88-2c11-4731-a433-36a04661de83",
		CreatedAt:      time.Now().Unix(),
		TTL:            time.Now().Add(5 * time.Minute).Unix(),
		MGRouteID:      "ddb9ec88-2c11-4731-a433-36a04661de83",
		FailedToCreate: false,
	}

	err := db.SaveNewInbox(i)

	if err != nil {
		t.Errorf("%v - TestGetMessagesByInboxID: failed to save: %v", reflect.TypeOf(db), err)
	}

	m1 := Message{
		InboxID:    "ddb9ec88-2c11-4731-a433-36a04661de83",
		ID:         "5678",
		ReceivedAt: time.Now().Unix(),
		MGID:       "56789",
		Sender:     "bob@example.com",
		From:       "Bobby Tables <bob@example.com>",
		Subject:    "DELETE FROM MESSAGES;",
		BodyPlain:  "Hello there how are you!",
		BodyHTML:   "<html><body><p>Hello there how are you!</p></body></html>",
		TTL:        time.Now().Add(5 * time.Minute).Unix(),
	}

	m2 := Message{
		InboxID:    "ddb9ec88-2c11-4731-a433-36a04661de83",
		ID:         "9999",
		ReceivedAt: time.Now().Unix(),
		MGID:       "56789",
		Sender:     "bob@example.com",
		From:       "Bobby Tables <bob@example.com>",
		Subject:    "DELETE FROM MESSAGES;",
		BodyPlain:  "Hello there how are you!",
		BodyHTML:   "<html><body><p>Hello there how are you!</p></body></html>",
		TTL:        time.Now().Add(5 * time.Minute).Unix(),
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

	if len(messages) != 2 {
		t.Errorf("%v - TestGetMessagesByInboxID: got back a different number of messages than saved: Expected 2, got %v", reflect.TypeOf(db), len(messages))
	}

	for _, m := range messages {
		if !reflect.DeepEqual(m, m1) {
			if !reflect.DeepEqual(m, m2) {
				t.Errorf("%v - TestGetMessagesByInboxID: Got back a different message than saved", reflect.TypeOf(db))
			}
		}
	}

	// Test that it returns an empty messages slice if there are no messages
	empty, err := db.GetMessagesByInboxID("doesntexist")

	if err != nil {
		t.Errorf("%v - TestGetMessagesByInboxID: get empty inbox: %v", reflect.TypeOf(db), err)
	}

	if len(empty) != 0 {
		t.Errorf("%v - TestGetMessagesByInboxID: returned messages for a non existent key", reflect.TypeOf(db))
	}
}
