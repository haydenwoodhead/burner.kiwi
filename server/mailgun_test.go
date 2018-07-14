package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/data"
	"github.com/haydenwoodhead/burner.kiwi/data/inmemory"
)

func TestServer_MailgunIncoming_Verified(t *testing.T) {
	s := Server{
		mg: FakeMG{Verify: true},
		db: inmemory.GetInMemoryDB(),
	}

	s.db.SaveNewInbox(data.Inbox{
		Address:        "bobby@example.com",
		ID:             "17b79467-f409-4e7d-86a9-0dc79b77f7c3",
		CreatedAt:      time.Now().Unix(),
		TTL:            time.Now().Add(1 * time.Hour).Unix(),
		FailedToCreate: false,
		MGRouteID:      "1234",
	})

	router := mux.NewRouter()
	router.HandleFunc("/mg/incoming/{inboxID}/", s.MailgunIncoming)

	httpServer := httptest.NewServer(router)

	resp, err := http.PostForm(httpServer.URL+"/mg/incoming/17b79467-f409-4e7d-86a9-0dc79b77f7c3/", url.Values{
		"message-id": {"1234"},
		"sender":     {"hayden@example.com"},
		"from":       {"hayden@example.com"},
		"subject":    {"Hello there"},
		"body-plain": {"Hello there"},
		"body-html":  {`<html><body><a href="https://example.com">Hello there</a></body></html>`},
	})

	if err != nil {
		t.Fatalf("TestServer_MailgunIncoming_Verified: failed to post data: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("TestServer_MailgunIncoming_Verified: returned status not 200 got: %v", resp.StatusCode)
	}

	msgs, _ := s.db.GetMessagesByInboxID("17b79467-f409-4e7d-86a9-0dc79b77f7c3")

	if len(msgs) == 0 {
		t.Fatalf("TestServer_MailgunIncoming_Verified: failed to save incoming message to database")
	}

	msg := msgs[0]

	if msg.MGID != "1234" {
		t.Fatalf("TestServer_MailgunIncoming_Verified: mailgun message id not set to 1234. Actually %v", msg.ID)
	}

	if msg.Sender != "hayden@example.com" || msg.From != "hayden@example.com" {
		t.Fatalf("TestServer_MailgunIncoming_Verified: sender or from not correct. Should be hayden@example.com. Sender: %v, from %v", msg.Sender, msg.From)
	}

	if msg.Subject != "Hello there" {
		t.Fatalf("TestServer_MailgunIncoming_Verfified: subject not 'Hello There', actually %v", msg.Subject)
	}

	if msg.BodyPlain != "Hello there" {
		t.Fatalf("TestServer_MailgunIncoming_Verfified: BodyPlain not 'Hello There', actually %v", msg.BodyPlain)
	}

	const expectedHTML = `<html><head></head><body><a href="https://example.com" target="_blank">Hello there</a></body></html>`

	if msg.BodyHTML != expectedHTML {
		t.Fatalf("TestServer_MailgunIncoming_Verfified: html body different than expected. \nExpected: %v\nGot: %v", expectedHTML, msg.BodyHTML)
	}
}

func TestServer_MailgunIncoming_UnVerified(t *testing.T) {
	s := Server{
		mg: FakeMG{Verify: false},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	s.MailgunIncoming(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("TestServer_MailgunIncoming_UnVerified: expected status code: %v, got %v", http.StatusUnauthorized, rr.Code)
	}
}
