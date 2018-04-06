package server

import (
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/justinas/alice"
)

const FAKEHANDLERRESP = "fake handler"
const NEWHANDLERRESP = "new handler"

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
