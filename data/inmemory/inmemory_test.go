package inmemory

import (
	"testing"
	"time"

	"github.com/haydenwoodhead/burner.kiwi/data"
	"github.com/haydenwoodhead/burner.kiwi/server"
)

func TestInMemoryDB(t *testing.T) {
	db := GetInMemoryDB()

	// iterate over the testing suite and call the function
	for _, f := range data.TestingFuncs {
		f(t, db)
	}
}

func TestInMemory_DeleteExpiredData(t *testing.T) {
	db := GetInMemoryDB()

	i1 := server.Inbox{
		ID:  "1234",
		TTL: time.Now().Add(-1 * time.Second).Unix(),
	}

	i2 := server.Inbox{
		ID:  "5678",
		TTL: time.Now().Add(1 * time.Hour).Unix(),
	}

	_ = db.SaveNewInbox(i1)
	_ = db.SaveNewInbox(i2)

	m1 := server.Message{
		InboxID: "1234",
		ID:      "1234",
		TTL:     time.Now().Add(-1 * time.Second).Unix(),
	}

	m2 := server.Message{
		InboxID: "5678",
		ID:      "5678",
		TTL:     time.Now().Add(1 * time.Hour).Unix(),
	}

	_ = db.SaveNewMessage(m1)
	_ = db.SaveNewMessage(m2)

	db.DeleteExpiredData()

	inboxTests := []struct {
		ID          string
		ExpectedErr error
	}{
		{
			ID:          "1234",
			ExpectedErr: errInboxDoesntExist,
		},
		{
			ID:          "5678",
			ExpectedErr: nil,
		},
	}

	for _, test := range inboxTests {
		_, err := db.GetInboxByID(test.ID)

		if err != test.ExpectedErr {
			t.Errorf("TestInMemory_DeleteExpiredData: inbox test failed. Expected error - %v, got %v", test.ExpectedErr, err)
		}
	}

	msgTests := []struct {
		ID          string
		ExpectedErr error
	}{
		{
			ID:          "1234",
			ExpectedErr: server.ErrMessageDoesntExist,
		},
		{
			ID:          "5678",
			ExpectedErr: nil,
		},
	}

	for _, test := range msgTests {
		_, err := db.GetMessageByID(test.ID, test.ID)

		if err != test.ExpectedErr {
			t.Errorf("TestInMemory_DeleteExpiredData: message test failed. Expected error - %v, got %v", test.ExpectedErr, err)
		}
	}
}
