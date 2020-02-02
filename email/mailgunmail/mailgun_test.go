package mailgunmail

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/haydenwoodhead/burner.kiwi/burner"
	"github.com/stretchr/testify/assert"
	mailgun "gopkg.in/mailgun/mailgun-go.v1"

	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/data/inmemory"
)

func TestMailgun_MailgunIncoming_Verified(t *testing.T) {
	m := MailgunMail{
		mg: FakeMG{Verify: true},
		db: inmemory.GetInMemoryDB(),
		isBlacklisted: func(email string) bool {
			return false
		},
	}

	m.db.SaveNewInbox(burner.Inbox{
		Address:        "bobby@example.com",
		ID:             "17b79467-f409-4e7d-86a9-0dc79b77f7c3",
		CreatedAt:      time.Now().Unix(),
		TTL:            time.Now().Add(1 * time.Hour).Unix(),
		FailedToCreate: false,
		MGRouteID:      "1234",
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

	if err != nil {
		t.Fatalf("TestServer_MailgunIncoming_Verified: failed to post data: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("TestServer_MailgunIncoming_Verified: returned status not 200 got: %v", resp.StatusCode)
	}

	msgs, _ := m.db.GetMessagesByInboxID("17b79467-f409-4e7d-86a9-0dc79b77f7c3")

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

func TestMailgun_MailgunIncoming_Blacklisted(t *testing.T) {
	m := MailgunMail{
		mg: FakeMG{Verify: true},
		db: inmemory.GetInMemoryDB(),
		isBlacklisted: func(email string) bool {
			return true
		},
	}

	m.db.SaveNewInbox(burner.Inbox{
		Address:        "bobby@example.com",
		ID:             "17b79467-f409-4e7d-86a9-0dc79b77f7c3",
		CreatedAt:      time.Now().Unix(),
		TTL:            time.Now().Add(1 * time.Hour).Unix(),
		FailedToCreate: false,
		MGRouteID:      "1234",
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
	m := MailgunMail{
		mg: FakeMG{Verify: false},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	m.mailgunIncoming(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("TestServer_MailgunIncoming_UnVerified: expected status code: %v, got %v", http.StatusUnauthorized, rr.Code)
	}
}

func TestMailgun_DeleteExpiredRoutes(t *testing.T) {
	m := MailgunMail{
		mg: FakeMG{},
	}

	// Should be deleted
	routes["1234"] = mailgun.Route{
		Priority:    1,
		Description: "1",
		Expression:  "",
		Actions:     []string{},
		CreatedAt:   "1",
		ID:          "1234",
	}

	// Should be deleted
	routes["91011"] = mailgun.Route{
		Priority:    1,
		Description: fmt.Sprintf("%v", time.Now().Add(-1*time.Second).Unix()),
		Expression:  "",
		Actions:     []string{},
		CreatedAt:   "1",
		ID:          "91011",
	}

	// should not be deleted
	routes["5678"] = mailgun.Route{
		Priority:    1,
		Description: "2124941352",
		Expression:  "",
		Actions:     []string{},
		CreatedAt:   "1",
		ID:          "5678",
	}

	m.DeleteExpiredRoutes()

	if _, ok := routes["1234"]; ok {
		t.Errorf("TestServer_DeleteOldRoutes: Expired route still exists.")
	}

	if _, ok := routes["91011"]; ok {
		t.Errorf("TestServer_DeleteOldRoutes: Expired route still exists.")
	}

	if _, ok := routes["5678"]; !ok {
		t.Errorf("TestServer_DeleteOldRoutes: Valid route deleted.")
	}
}

type FakeMG struct {
	Verify bool
}

var routes = map[string]mailgun.Route{}

func (f FakeMG) DeleteRoute(id string) error {
	if routes == nil {
		routes = make(map[string]mailgun.Route)
	}

	delete(routes, id)

	return nil
}

func (FakeMG) GetRoutes(limit, skip int) (int, []mailgun.Route, error) {
	if routes == nil {
		routes = make(map[string]mailgun.Route)
	}

	var r []mailgun.Route

	for _, v := range routes {
		r = append(r, v)
	}

	return len(r), r, nil
}

func (FakeMG) CreateRoute(m mailgun.Route) (mailgun.Route, error) {
	if routes == nil {
		routes = make(map[string]mailgun.Route)
	}

	id, _ := uuid.NewRandom() // not the same format as mailgun but lets give each route a random id

	m.ID = id.String()
	m.CreatedAt = fmt.Sprintf("%v", time.Now().Unix())

	routes[m.ID] = m

	time.Sleep(time.Millisecond * 500) // add fake network latency

	return m, nil
}

func (f FakeMG) VerifyWebhookRequest(req *http.Request) (verified bool, err error) {
	return f.Verify, nil
}

// Argh... mailgun and their massive interface making me implement all these methods
// nolint
func (FakeMG) ApiBase() string {
	panic("implement me")
}

func (FakeMG) Domain() string {
	panic("implement me")
}

// nolint
func (FakeMG) ApiKey() string {
	panic("implement me")
}

// nolint
func (FakeMG) PublicApiKey() string {
	panic("implement me")
}

func (FakeMG) Client() *http.Client {
	panic("implement me")
}

func (FakeMG) SetClient(client *http.Client) {
	panic("implement me")
}

func (FakeMG) Send(m *mailgun.Message) (string, string, error) {
	panic("implement me")
}

func (FakeMG) ValidateEmail(email string) (mailgun.EmailVerification, error) {
	panic("implement me")
}

func (FakeMG) ParseAddresses(addresses ...string) ([]string, []string, error) {
	panic("implement me")
}

func (FakeMG) GetBounces(limit, skip int) (int, []mailgun.Bounce, error) {
	panic("implement me")
}

func (FakeMG) GetSingleBounce(address string) (mailgun.Bounce, error) {
	panic("implement me")
}

func (FakeMG) AddBounce(address, code, error string) error {
	panic("implement me")
}

func (FakeMG) DeleteBounce(address string) error {
	panic("implement me")
}

func (FakeMG) GetStats(limit int, skip int, startDate *time.Time, event ...string) (int, []mailgun.Stat, error) {
	panic("implement me")
}

func (FakeMG) GetTag(tag string) (mailgun.TagItem, error) {
	panic("implement me")
}

func (FakeMG) DeleteTag(tag string) error {
	panic("implement me")
}

func (FakeMG) ListTags(*mailgun.TagOptions) *mailgun.TagIterator {
	panic("implement me")
}

func (FakeMG) GetDomains(limit, skip int) (int, []mailgun.Domain, error) {
	panic("implement me")
}

func (FakeMG) GetSingleDomain(domain string) (mailgun.Domain, []mailgun.DNSRecord, []mailgun.DNSRecord, error) {
	panic("implement me")
}

func (FakeMG) CreateDomain(name string, smtpPassword string, spamAction string, wildcard bool) error {
	panic("implement me")
}

func (FakeMG) DeleteDomain(name string) error {
	panic("implement me")
}

func (FakeMG) GetCampaigns() (int, []mailgun.Campaign, error) {
	panic("implement me")
}

func (FakeMG) CreateCampaign(name, id string) error {
	panic("implement me")
}

// nolint
func (FakeMG) UpdateCampaign(oldId, name, newId string) error {
	panic("implement me")
}

func (FakeMG) DeleteCampaign(id string) error {
	panic("implement me")
}

func (FakeMG) GetComplaints(limit, skip int) (int, []mailgun.Complaint, error) {
	panic("implement me")
}

func (FakeMG) GetSingleComplaint(address string) (mailgun.Complaint, error) {
	panic("implement me")
}

func (FakeMG) GetStoredMessage(id string) (mailgun.StoredMessage, error) {
	panic("implement me")
}

func (FakeMG) GetStoredMessageRaw(id string) (mailgun.StoredMessageRaw, error) {
	panic("implement me")
}

func (FakeMG) DeleteStoredMessage(id string) error {
	panic("implement me")
}

func (FakeMG) GetCredentials(limit, skip int) (int, []mailgun.Credential, error) {
	panic("implement me")
}

func (FakeMG) CreateCredential(login, password string) error {
	panic("implement me")
}

func (FakeMG) ChangeCredentialPassword(id, password string) error {
	panic("implement me")
}

func (FakeMG) DeleteCredential(id string) error {
	panic("implement me")
}

func (FakeMG) GetUnsubscribes(limit, skip int) (int, []mailgun.Unsubscription, error) {
	panic("implement me")
}

func (FakeMG) GetUnsubscribesByAddress(string) (int, []mailgun.Unsubscription, error) {
	panic("implement me")
}

func (FakeMG) Unsubscribe(address, tag string) error {
	panic("implement me")
}

func (FakeMG) RemoveUnsubscribe(string) error {
	panic("implement me")
}

func (FakeMG) RemoveUnsubscribeWithTag(a, t string) error {
	panic("implement me")
}

func (FakeMG) CreateComplaint(string) error {
	panic("implement me")
}

func (FakeMG) DeleteComplaint(string) error {
	panic("implement me")
}

func (FakeMG) GetRouteByID(string) (mailgun.Route, error) {
	panic("implement me")
}

func (FakeMG) UpdateRoute(string, mailgun.Route) (mailgun.Route, error) {
	panic("implement me")
}

func (FakeMG) GetWebhooks() (map[string]string, error) {
	panic("implement me")
}

func (FakeMG) CreateWebhook(kind, url string) error {
	panic("implement me")
}

func (FakeMG) DeleteWebhook(kind string) error {
	panic("implement me")
}

func (FakeMG) GetWebhookByType(kind string) (string, error) {
	panic("implement me")
}

func (FakeMG) UpdateWebhook(kind, url string) error {
	panic("implement me")
}

func (FakeMG) GetLists(limit, skip int, filter string) (int, []mailgun.List, error) {
	panic("implement me")
}

func (FakeMG) CreateList(mailgun.List) (mailgun.List, error) {
	panic("implement me")
}

func (FakeMG) DeleteList(string) error {
	panic("implement me")
}

func (FakeMG) GetListByAddress(string) (mailgun.List, error) {
	panic("implement me")
}

func (FakeMG) UpdateList(string, mailgun.List) (mailgun.List, error) {
	panic("implement me")
}

func (FakeMG) GetMembers(limit, skip int, subfilter *bool, address string) (int, []mailgun.Member, error) {
	panic("implement me")
}

func (FakeMG) GetMemberByAddress(MemberAddr, listAddr string) (mailgun.Member, error) {
	panic("implement me")
}

func (FakeMG) CreateMember(merge bool, addr string, prototype mailgun.Member) error {
	panic("implement me")
}

func (FakeMG) CreateMemberList(subscribed *bool, addr string, newMembers []interface{}) error {
	panic("implement me")
}

func (FakeMG) UpdateMember(Member, list string, prototype mailgun.Member) (mailgun.Member, error) {
	panic("implement me")
}

func (FakeMG) DeleteMember(Member, list string) error {
	panic("implement me")
}

func (FakeMG) NewMessage(from, subject, text string, to ...string) *mailgun.Message {
	panic("implement me")
}

func (FakeMG) NewMIMEMessage(body io.ReadCloser, to ...string) *mailgun.Message {
	panic("implement me")
}

func (FakeMG) NewEventIterator() *mailgun.EventIterator {
	panic("implement me")
}

func (FakeMG) ListEvents(*mailgun.EventsOptions) *mailgun.EventIterator {
	panic("implement me")
}

func (FakeMG) PollEvents(*mailgun.EventsOptions) *mailgun.EventPoller {
	panic("implement me")
}

func (FakeMG) SetAPIBase(url string) {
	panic("implement me")
}

func (f FakeMG) GetStatsTotal(start *time.Time, end *time.Time, resolution string, duration string, event ...string) (*mailgun.StatsTotalResponse, error) {
	panic("implement me")
}

func (f FakeMG) GetStoredMessageForURL(url string) (mailgun.StoredMessage, error) {
	panic("implement me")
}

func (f FakeMG) GetStoredMessageRawForURL(url string) (mailgun.StoredMessageRaw, error) {
	panic("implement me")
}
