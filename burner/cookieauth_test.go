package burner

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/justinas/alice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSessionFromCookie(t *testing.T) {
	s := &Server{
		sessionStore: sessions.NewCookieStore([]byte("testtest1234")),
	}

	cj, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cj,
	}

	r := mux.NewRouter()
	r.Handle("/setcookie", setCookieHandler(s))
	r.Handle("/getcookie", checkCookieHandler(s, t))

	server := httptest.NewServer(r)

	setCookieResp, err := client.Get(server.URL + "/setcookie")
	require.NoError(t, err)

	setCookieBody, err := ioutil.ReadAll(setCookieResp.Body)
	require.NoError(t, err)
	defer setCookieResp.Body.Close()

	assert.Equal(t, "OK", string(setCookieBody))

	getCookieResp, err := client.Get(server.URL + "/getcookie")
	require.NoError(t, err)

	getCookieBody, err := ioutil.ReadAll(getCookieResp.Body)
	require.NoError(t, err)
	defer getCookieResp.Body.Close()

	assert.Equal(t, "OK", string(getCookieBody))
}

func TestServer_CheckSessionCookieExists(t *testing.T) {
	s := &Server{
		sessionStore: sessions.NewCookieStore([]byte("testtest1234")),
	}

	cj, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cj,
	}

	r := mux.NewRouter()
	r.Handle("/protected", alice.New(s.CheckSessionCookieExists).ThenFunc(fakeHandler))

	server := httptest.NewServer(r)

	resp, err := client.Get(server.URL + "/protected")
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func setCookieHandler(s *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := s.getSessionFromCookie(r)
		err := session.SetInboxID("1234", w)
		if err != nil {
			http.Error(w, "fail", http.StatusInternalServerError)
		}
		fmt.Fprintf(w, "OK")
	})
}

func checkCookieHandler(s *Server, t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := s.getSessionFromCookie(r)
		assert.Equal(t, false, session.IsNew)
		assert.Equal(t, "1234", session.InboxID)
		fmt.Fprintf(w, "OK")
	})
}
