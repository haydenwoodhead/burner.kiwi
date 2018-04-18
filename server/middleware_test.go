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

			errors, ok := resp.Errors.(map[string]interface{})

			if !ok {
				t.Errorf("TestServer_CheckPermissionJSON: %v - failed to assert that resp.Errors is a map. It's actually: %v", i, reflect.TypeOf(resp.Errors))
			}

			msg, mOK := errors["msg"].(string)

			if !mOK {
				t.Errorf("TestServer_CheckPermissionJSON: %v - failed to assert that Msg is a string. It's actually: %v", i, reflect.TypeOf(msg))
			}

			if strings.Compare(msg, test.ExpectedMsg) != 0 {
				t.Errorf("TestServer_CheckPermissionJSON: %v - Message different. Expected %v, got %v", i, test.ExpectedMsg, msg)
			}
		}
	}
}

func TestServer_CheckCookieExists(t *testing.T) {
	s := Server{
		store: sessions.NewCookieStore([]byte("testtest1234")),
	}

	cj, _ := cookiejar.New(nil)

	client := &http.Client{
		Jar: cj,
	}

	r := mux.NewRouter()
	r.Handle("/fake", alice.New(s.CheckCookieExists(fakeErrorPrinter)).ThenFunc(fakeHandler))
	r.HandleFunc("/setcookie", setCookieHandler)
	r.Handle("/getcookie", alice.New(s.CheckCookieExists(fakeErrorPrinter)).Then(checkCookieHandler(t)))

	server := httptest.NewServer(r)

	// First request - we want this to send us to fakeErrorPrinter
	resp1, err := client.Get(server.URL + "/fake")

	if err != nil {
		t.Fatalf("TestServer_CheckCookieExists: failed to perform first request: %v", err)
	}

	defer resp1.Body.Close()
	body1, err := ioutil.ReadAll(resp1.Body)

	if err != nil {
		t.Fatalf("TestServer_CheckCookieExists: Failed to read body1: %v", err)
	}

	if strings.Compare(string(body1), checkCookieExistsErrorResponse) != 0 {
		t.Fatalf("TestServer_CheckCookieExists: Body1 not expected. Expected %v, got %v", checkCookieExistsErrorResponse, string(body1))
	}

	// Second request to setup cookie/session
	resp2, err := client.Get(server.URL + "/setcookie")

	if err != nil {
		t.Fatalf("TestServer_CheckCookieExists: failed to perform second request: %v", err)
	}

	defer resp2.Body.Close()
	body2, err := ioutil.ReadAll(resp2.Body)

	if err != nil {
		t.Fatalf("TestServer_CheckCookieExists: Failed to read body2: %v", err)
	}

	if strings.Compare(string(body2), "success") != 0 {
		t.Fatalf("TestServer_CheckCookieExists: Body2 not expected. Expected %v, got %v", "success", string(body2))
	}

	// Third request to check that session ctx is properly set
	resp3, err := client.Get(server.URL + "/getcookie")

	if err != nil {
		t.Fatalf("TestServer_CheckCookieExists: failed to perform third request: %v", err)
	}

	defer resp3.Body.Close()
	body3, err := ioutil.ReadAll(resp3.Body)

	if err != nil {
		t.Fatalf("TestServer_CheckCookieExists: Failed to read body3: %v", err)
	}

	if strings.Compare(string(body3), "success") != 0 {
		t.Fatalf("TestServer_CheckCookieExists: Body3 not expected. Expected %v, got %v", "success", string(body3))
	}
}

// Mock handler implementations

var store = sessions.NewCookieStore([]byte("testtest1234"))

func fakeErrorPrinter(w http.ResponseWriter, r *http.Request, code int, msg string) {
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func fakeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(FAKEHANDLERRESP))
}

func setCookieHandler(w http.ResponseWriter, r *http.Request) {
	sess, _ := store.Get(r, sessionStoreKey)

	sess.Values["id"] = "1234"

	err := sess.Save(r, w)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed"))
	}

	w.Write([]byte("success"))
}

func checkCookieHandler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := r.Context().Value(sessionCTXKey).(*sessions.Session)

		if !ok {
			t.Fatalf("%v - Sess not of type sessions.Session actual type: %v", t.Name(), reflect.TypeOf(sess))
		}

		if sess.IsNew {
			t.Errorf("%v failed. Session not set properly", t.Name())
		}

		w.Write([]byte("success"))
	})
}

func newHandler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := r.Context().Value(sessionCTXKey).(*sessions.Session)

		if !ok {
			t.Fatalf("Sess not of type sessions.Session actual type: %v", reflect.TypeOf(sess))
		}

		sess.Values["id"] = "1234"

		sess.Save(r, w)
		w.Write([]byte(NEWHANDLERRESP))
	})
}
