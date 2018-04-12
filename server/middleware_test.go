package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/haydenwoodhead/burnerkiwi/token"
	"github.com/justinas/alice"
)

const FAKEHANDLERRESP = "fake handler"
const NEWHANDLERRESP = "new handler"
const PASSKEY = "pass key"
const DONTCHECK = "string"

func TestServer_IsNew(t *testing.T) {
	s := Server{
		store: sessions.NewCookieStore([]byte("testtest1234")),
	}

	cj, _ := cookiejar.New(nil)

	client := &http.Client{
		Jar: cj,
	}

	server := httptest.NewServer(alice.New(s.IsNew(newHandler(t))).ThenFunc(fakeHandler))

	resp, err := client.Get(server.URL)

	if err != nil {
		t.Fatalf("TestServer_IsNew: failed to perform first request: %v", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		t.Errorf("TestServer_IsNew: Failed to read body: %v", err)
	}

	if strings.Compare(string(body), NEWHANDLERRESP) != 0 {
		t.Fatalf("TestServer_IsNew: First request not matching. Expected %v, got %v.", NEWHANDLERRESP, string(body))
	}

	resp2, err := client.Get(server.URL)

	if err != nil {
		t.Fatalf("TestServer_IsNew: failed to perform request 2: %v", err)
	}

	defer resp2.Body.Close()
	body2, err := ioutil.ReadAll(resp2.Body)

	if err != nil {
		t.Errorf("TestServer_IsNew: Failed to read 2nd body: %v", err)
	}

	if strings.Compare(string(body2), FAKEHANDLERRESP) != 0 {
		t.Fatalf("TestServer_IsNew: Second request not matching. Expected '%v', got '%v'.", FAKEHANDLERRESP, string(body2))
	}
}

func TestServer_CheckPermissionJSON(t *testing.T) {
	tests := []struct {
		Duration       time.Duration
		AuthHeader     string
		ExpectedStatus int
		ExpectedMsg    string
	}{
		{
			Duration:       time.Hour,
			AuthHeader:     PASSKEY,
			ExpectedStatus: http.StatusOK,
			ExpectedMsg:    DONTCHECK,
		},
		{
			Duration:       time.Second,
			AuthHeader:     PASSKEY,
			ExpectedStatus: http.StatusForbidden,
			ExpectedMsg:    "Forbidden: your token has expired",
		},
		{
			Duration:       time.Hour,
			AuthHeader:     "not valid",
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedMsg:    DONTCHECK,
		},
	}

	for i, test := range tests {
		s := Server{
			tg: token.NewGenerator("test1234", test.Duration),
		}

		tk := s.tg.NewToken("dafd5606-8aa8-4724-a2c5-f66110aba536")

		time.Sleep(2 * time.Second) // so we can test whether token expiration works correctly

		rr := httptest.NewRecorder()

		h := mux.NewRouter()
		h.Handle("/{inboxID}", alice.New(JSONContentType, s.CheckPermissionJSON).ThenFunc(fakeHandler))

		r := httptest.NewRequest(http.MethodGet, "/dafd5606-8aa8-4724-a2c5-f66110aba536", nil)

		if strings.Compare(test.AuthHeader, PASSKEY) == 0 {
			r.Header.Set("X-Burner-Key", tk)
		} else {
			r.Header.Set("X-Burner-Key", test.AuthHeader)
		}

		h.ServeHTTP(rr, r)

		if rr.Code != test.ExpectedStatus {
			t.Errorf("TestServer_CheckPermissionJSON: %v - Status code different. Expected %v, got %v", i, test.ExpectedStatus, rr.Code)
		}

		if strings.Compare(test.ExpectedMsg, DONTCHECK) != 0 {
			resp := Response{}

			err := json.Unmarshal(rr.Body.Bytes(), &resp)

			if err != nil {
				t.Errorf("TestServer_CheckPermissionJSON: %v - Failed to unmarshal json resp: %v", i, err)
			}

			if strings.Compare(resp.Errors.Msg, test.ExpectedMsg) != 0 {
				t.Errorf("TestServer_CheckPermissionJSON: %v - Message different. Expected %v, got %v", i, test.ExpectedMsg, resp.Errors.Msg)
			}
		}
	}
}

func fakeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(FAKEHANDLERRESP))
}

func newHandler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := r.Context().Value("sess").(*sessions.Session)

		if !ok {
			t.Fatalf("Sess not of type sessions.Session actual type: %v", reflect.TypeOf(sess))
		}

		sess.Values["id"] = "1234"

		sess.Save(r, w)
		w.Write([]byte(NEWHANDLERRESP))
	})
}
