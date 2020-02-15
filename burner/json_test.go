package burner

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/emailgenerator"
	"github.com/haydenwoodhead/burner.kiwi/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestServer_NewInboxJSON(t *testing.T) {
	mDB := new(MockDatabase)
	inbox := Inbox{
		Address:              "fpllngzi@example.com",
		CreatedBy:            "192.168.1.1",
		EmailProviderRouteID: "1234",
	}
	mDB.On("EmailAddressExists", "fpllngzi@example.com").Return(false, nil)
	mDB.On("SaveNewInbox", mock.MatchedBy(InboxMatcher(inbox))).Return(nil)
	mDB.On("SetInboxCreated", mock.MatchedBy(InboxMatcher(inbox))).Return(nil)

	mEP := new(MockEmailProvider)
	mEP.On("RegisterRoute", mock.Anything).Return("1234", nil)

	s := Server{
		db:          mDB,
		tg:          token.NewGenerator("testexample12344", time.Hour),
		email:       mEP,
		eg:          &emailgenerator.EmailGenerator{Hosts: []string{"example.com"}, L: 8},
		usingLambda: true, // make sure the create route goroutine finishes before we check the result
	}

	rr := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.1"

	s.NewInboxJSON(rr, r)

	if rr.Code != http.StatusOK {
		t.Errorf("TestServer_NewInboxJSON: failed response code not 200. Got %v", rr.Code)
	}

	var res Response
	err := json.Unmarshal(rr.Body.Bytes(), &res)
	if err != nil {
		t.Errorf("TestServer_NewInboxJSON: failed to unmarshal response: %v", err)
	}

	resMap, ok := res.Result.(map[string]interface{})
	if !ok {
		t.Errorf("TestServer_NewInboxJSON: response.Result not map[string]interface{} actually %v", reflect.TypeOf(res.Result))
	}

	resEmail, ok := resMap["email"].(map[string]interface{})
	if !ok {
		t.Errorf("TestServer_NewInboxJSON: response.Result.Email not map[string]interface{} actually %v", reflect.TypeOf(resMap["email"]))
	}

	_, ok = resEmail["id"].(string)
	if !ok {
		t.Errorf("TestServer_NewInboxJSON: response.Result.Email.ID not string actually %v", reflect.TypeOf(resMap["email"]))
	}

	mEP.AssertExpectations(t)
	mDB.AssertExpectations(t)
}

func TestServer_GetInboxDetailsJSON(t *testing.T) {
	mDB := new(MockDatabase)
	mDB.On("GetInboxByID", "1234").Return(Inbox{
		Address:              "1234@example.com",
		ID:                   "1234",
		CreatedAt:            1526186018,
		CreatedBy:            "192.168.1.1",
		TTL:                  1526189618,
		EmailProviderRouteID: "1234",
		FailedToCreate:       false,
	}, nil)
	mDB.On("GetInboxByID", "Doesntexist").Return(Inbox{}, errors.New("inbox doesn't exist"))

	s := Server{
		db:          mDB,
		tg:          token.NewGenerator("testexample12344", time.Hour),
		eg:          emailgenerator.New([]string{"example.com"}, 8),
		usingLambda: true, // make sure the create route goroutine finishes before we check the result
	}

	router := mux.NewRouter()
	router.HandleFunc("/{inboxID}", s.GetInboxDetailsJSON)

	test := []struct {
		Name             string
		ID               string
		ExpectedResponse string
		ExpectedCode     int
	}{
		{
			Name:             "inbox exists",
			ID:               "1234",
			ExpectedCode:     200,
			ExpectedResponse: `{"success":true,"errors":null,"result":{"address":"1234@example.com","id":"1234","created_at":1526186018,"ttl":1526189618},"meta":{"version":"dev","by":"Hayden Woodhead"}}`,
		},
		{
			Name:             "inbox doesn't exist",
			ID:               "Doesntexist",
			ExpectedResponse: `{"success":false,"errors":{"code":500,"msg":"Failed to get email details"},"result":null,"meta":{"version":"dev","by":"Hayden Woodhead"}}`,
			ExpectedCode:     500,
		},
	}

	for _, test := range test {
		t.Run(test.Name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/"+test.ID, nil)

			router.ServeHTTP(rr, r)

			assert.Equal(t, test.ExpectedCode, rr.Code)
			assert.JSONEq(t, test.ExpectedResponse, rr.Body.String())
		})
	}

	mDB.AssertExpectations(t)
}

func TestServer_GetAllMessagesJSON(t *testing.T) {
	mDB := new(MockDatabase)
	mDB.On("GetMessagesByInboxID", "1234").Return([]Message{{
		InboxID:         "1234",
		ID:              "91991919",
		ReceivedAt:      1526186100,
		EmailProviderID: "56789",
		Sender:          "bob@example.com",
		From:            "Bobby Tables <bob@example.com>",
		Subject:         "DELETE FROM MESSAGES;",
		BodyPlain:       "Hello there how are you!",
		BodyHTML:        "<html><body><p>Hello there how are you!</p></body></html>",
		TTL:             1526189618,
	}}, nil)

	s := Server{
		db:          mDB,
		tg:          token.NewGenerator("testexample12344", time.Hour),
		eg:          emailgenerator.New([]string{"example.com"}, 8),
		usingLambda: true, // make sure the create route goroutine finishes before we check the result
	}

	router := mux.NewRouter()
	router.HandleFunc("/{inboxID}", s.GetAllMessagesJSON)

	rr := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/1234", nil)

	router.ServeHTTP(rr, r)

	var expected = `{"success":true,"errors":null,"result":[{"id":"91991919","received_at":1526186100,"sender":"bob@example.com","from":"Bobby Tables \u003cbob@example.com\u003e","subject":"DELETE FROM MESSAGES;","body_html":"\u003chtml\u003e\u003cbody\u003e\u003cp\u003eHello there how are you!\u003c/p\u003e\u003c/body\u003e\u003c/html\u003e","body_plain":"Hello there how are you!","ttl":1526189618}],"meta":{"version":"dev","by":"Hayden Woodhead"}}`
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, expected, rr.Body.String())

	mDB.AssertExpectations(t)
}
