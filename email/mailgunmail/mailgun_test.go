package mailgunmail

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/haydenwoodhead/burner.kiwi/burner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	mailgun "gopkg.in/mailgun/mailgun-go.v1"

	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/data/inmemory"
)

// TODO swap out for proper mock db instead of in memory
func TestMailgun_MailgunIncoming_Verified(t *testing.T) {
	mockMailgun := new(MockMailgun)
	mockMailgun.On("VerifyWebhookRequest", mock.Anything).Return(true, nil)

	m := MailgunMail{
		mg: mockMailgun,
		db: inmemory.GetInMemoryDB(),
		isBlacklistedDomain: func(email string) bool {
			return false
		},
	}

	m.db.SaveNewInbox(burner.Inbox{
		Address:              "bobby@example.com",
		ID:                   "17b79467-f409-4e7d-86a9-0dc79b77f7c3",
		CreatedAt:            time.Now().Unix(),
		TTL:                  time.Now().Add(1 * time.Hour).Unix(),
		FailedToCreate:       false,
		EmailProviderRouteID: "1234",
	})

	router := mux.NewRouter()
	router.HandleFunc("/mg/incoming/{inboxID}/", m.mailgunIncoming)

	httpServer := httptest.NewServer(router)

	resp, err := http.PostForm(httpServer.URL+"/mg/incoming/17b79467-f409-4e7d-86a9-0dc79b77f7c3/", url.Values{
		"message-id": {"1234"},
		"sender":     {"hayden@example.com"},
		"from":       {"hayden@example.com"},
		"subject":    {"Subject line"},
		"body-plain": {"Hello there"},
		"body-html":  {`<html><body><a href="https://example.com">Hello there</a></body></html>`},
	})
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	msgs, _ := m.db.GetMessagesByInboxID("17b79467-f409-4e7d-86a9-0dc79b77f7c3")
	assert.Equal(t, 1, len(msgs))

	msg := msgs[0]
	assert.Equal(t, msg.EmailProviderID, "1234")
	assert.Equal(t, msg.Sender, "hayden@example.com")
	assert.Equal(t, msg.From, "hayden@example.com")
	assert.Equal(t, msg.Subject, "Subject line")
	assert.Equal(t, msg.BodyPlain, "Hello there")
	const expectedHTML = `<html><head></head><body><a href="https://example.com" target="_blank">Hello there</a></body></html>`
	assert.Equal(t, expectedHTML, msg.BodyHTML)
}

func TestMailgun_MailgunIncoming_Blacklisted(t *testing.T) {
	mockMailgun := new(MockMailgun)
	mockMailgun.On("VerifyWebhookRequest", mock.Anything).Return(true, nil)

	m := MailgunMail{
		mg: mockMailgun,
		db: inmemory.GetInMemoryDB(),
		isBlacklistedDomain: func(email string) bool {
			return true
		},
	}

	m.db.SaveNewInbox(burner.Inbox{
		Address:              "bobby@example.com",
		ID:                   "17b79467-f409-4e7d-86a9-0dc79b77f7c3",
		CreatedAt:            time.Now().Unix(),
		TTL:                  time.Now().Add(1 * time.Hour).Unix(),
		FailedToCreate:       false,
		EmailProviderRouteID: "1234",
	})

	router := mux.NewRouter()
	router.HandleFunc("/mg/incoming/{inboxID}/", m.mailgunIncoming)

	httpServer := httptest.NewServer(router)

	resp, err := http.PostForm(httpServer.URL+"/mg/incoming/17b79467-f409-4e7d-86a9-0dc79b77f7c3/", url.Values{
		"message-id": {"1234"},
		"sender":     {"hayden@example.com"},
		"from":       {"hayden@example.com"},
		"subject":    {"Hello there"},
		"body-plain": {"Hello there"},
		"body-html":  {`<html><body><a href="https://example.com">Hello there</a></body></html>`},
	})

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotAcceptable, resp.StatusCode)
}

func TestMailgun_MailgunIncoming_UnVerified(t *testing.T) {
	mockMailgun := new(MockMailgun)
	mockMailgun.On("VerifyWebhookRequest", mock.Anything).Return(false, nil)

	m := MailgunMail{
		mg: mockMailgun,
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	m.mailgunIncoming(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	mockMailgun.AssertExpectations(t)
}

func TestMailgun_DeleteExpiredRoutes(t *testing.T) {
	mockMailgun := new(MockMailgun)

	mockMailgun.On("GetRoutes", mock.Anything, mock.Anything).Return(3, []mailgun.Route{
		{ // should be called to be deleted
			Priority:    1,
			Description: "1",
			Expression:  "",
			Actions:     []string{},
			CreatedAt:   "1",
			ID:          "1234",
		},
		{ // should be called to be deleted
			Priority:    1,
			Description: fmt.Sprintf("%v", time.Now().Add(-1*time.Second).Unix()),
			Expression:  "",
			Actions:     []string{},
			CreatedAt:   "1",
			ID:          "91011",
		},
		{
			Priority:    1,
			Description: "2124941352",
			Expression:  "",
			Actions:     []string{},
			CreatedAt:   "1",
			ID:          "5678",
		},
	}, nil)

	mockMailgun.On("DeleteRoute", "1234").Return(nil)
	mockMailgun.On("DeleteRoute", "91011").Return(nil)

	m := MailgunMail{
		mg: mockMailgun,
	}
	m.deleteExpiredRoutes()

	mockMailgun.AssertExpectations(t)
}

type MockMailgun struct {
	mock.Mock
}

func (f *MockMailgun) DeleteRoute(id string) error {
	args := f.Called(id)
	return args.Error(0)
}

func (f *MockMailgun) GetRoutes(limit, skip int) (int, []mailgun.Route, error) {
	args := f.Called(limit, skip)
	return args.Int(0), args.Get(1).([]mailgun.Route), args.Error(2)
}

func (f *MockMailgun) CreateRoute(m mailgun.Route) (mailgun.Route, error) {
	args := f.Called(m)
	return args.Get(0).(mailgun.Route), args.Error(1)
}

func (f *MockMailgun) VerifyWebhookRequest(req *http.Request) (verified bool, err error) {
	args := f.Called(req)
	return args.Bool(0), args.Error(1)
}
