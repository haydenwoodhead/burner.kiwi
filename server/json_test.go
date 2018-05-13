package server

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/haydenwoodhead/burner.kiwi/data/inmemory"
	"github.com/haydenwoodhead/burner.kiwi/generateemail"
	"github.com/haydenwoodhead/burner.kiwi/token"
)

func TestServer_NewInboxJSON(t *testing.T) {
	db := inmemory.GetInMemoryDB()

	s := Server{
		db:          db,
		tg:          token.NewGenerator("testexample12344", time.Hour),
		mg:          FakeMG{},
		eg:          generateemail.NewEmailGenerator([]string{"example.com"}, 8),
		usingLambda: true, // make sure the create route goroutine finishes before we check the result
	}

	rr := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	s.NewInboxJSON(rr, r)

	if rr.Code != 200 {
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

	resID, ok := resEmail["id"].(string)

	if !ok {
		t.Errorf("TestServer_NewInboxJSON: response.Result.Email.ID not string actually %v", reflect.TypeOf(resMap["email"]))
	}

	//time.Sleep(time.Millisecond * 50)

	inbox, err := db.GetInboxByID(resID)

	if err != nil {
		t.Errorf("TestServer_NewInboxJSON: failed to retireve inbox from db. Error: %v", err)
	}

	if inbox.FailedToCreate {
		t.Error("TestServer_NewInboxJSON: inbox not set as created")
	}
}
