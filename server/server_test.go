package server

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	mailgun "gopkg.in/mailgun/mailgun-go.v1"
)

func TestMustParseTemplates(t *testing.T) {
	indexFile := template.Must(template.New("index").ParseFiles("../templates/base.html", "../templates/index.html"))
	indexPackr := MustParseTemplates(templates, "base.html", "index.html")

	out := indexOut{}

	fRecorder := httptest.NewRecorder()
	pRecorder := httptest.NewRecorder()

	if err := indexFile.ExecuteTemplate(fRecorder, "base", out); err != nil {
		t.Fatal(err)
	}

	if err := indexPackr.ExecuteTemplate(pRecorder, "base", out); err != nil {
		t.Fatal(err)
	}

	if strings.Compare(fRecorder.Body.String(), pRecorder.Body.String()) != 0 {
		t.Fatal("rendered html doesn't match")
	}
}

func TestServer_DeleteOldRoutes(t *testing.T) {
	s := Server{
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

	s.DeleteOldRoutes()

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
