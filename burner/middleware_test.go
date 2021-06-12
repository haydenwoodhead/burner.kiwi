package burner

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/notary"
	"github.com/justinas/alice"
	"github.com/stretchr/testify/assert"
)

const FAKEHANDLERRESP = "fake handler"
const NEWHANDLERRESP = "new handler"

func TestServer_CheckPermissionJSON(t *testing.T) {
	tests := []struct {
		Name               string
		TokenToPass        string
		ExpectedStatusCode int
	}{
		{
			Name:               "valid token",
			TokenToPass:        "eyJhbGciOiJIUzI1NiJ9.eyJJbmJveElEIjoiZGFmZDU2MDYtOGFhOC00NzI0LWEyYzUtZjY2MTEwYWJhNTM2IiwiX19wdXJwb3NlIjoiYXV0aCIsImV4cCI6MTYxODkyOTcyMCwiaWF0IjoxNjE4OTI5NjYwLCJpc3MiOiJidXJuZXIua2l3aSJ9.jQdwYLV-7JdYNvvU7NX2jmnl5dORwad3LTS2ecLEWnI",
			ExpectedStatusCode: http.StatusOK,
		},
		{
			Name:               "missing token",
			TokenToPass:        "",
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		{
			Name:               "token modified",
			TokenToPass:        "not-a-real-token",
			ExpectedStatusCode: http.StatusUnauthorized,
		},
		{
			Name:               "expired token",
			TokenToPass:        "eyJhbGciOiJIUzI1NiJ9.eyJJbmJveElEIjoiZGFmZDU2MDYtOGFhOC00NzI0LWEyYzUtZjY2MTEwYWJhNTM2IiwiX19wdXJwb3NlIjoiYXV0aCIsImV4cCI6MTYxODkyOTU0MCwiaWF0IjoxNjE4OTI5NjYwLCJpc3MiOiJidXJuZXIua2l3aSJ9.rGzq4xvdbOA_NzeWAxJcYSr6YNDlT1EDBMfA95zxHv8",
			ExpectedStatusCode: http.StatusForbidden,
		},
		{
			Name:               "different inbox id",
			TokenToPass:        "eyJhbGciOiJIUzI1NiJ9.eyJJbmJveElEIjoiZGFmZDU2MDYtOGFhOC00NzI0LWEyYzUtZjY2MTEwYWJhNTM1IiwiX19wdXJwb3NlIjoiYXV0aCIsImV4cCI6MTYxODkyOTcyMCwiaWF0IjoxNjE4OTI5NjYwLCJpc3MiOiJidXJuZXIua2l3aSJ9.VuPj2SZpqOVEAcrHyuIHiFmqx7lYmQUy8JhQ-eN0dhg",
			ExpectedStatusCode: http.StatusForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			s := Server{
				notariser: &notary.Notary{
					SigningKey: "test",
					Clock: func() time.Time {
						return time.Date(2021, 04, 20, 14, 41, 0, 0, time.UTC)
					},
				},
			}

			rr := httptest.NewRecorder()

			h := mux.NewRouter()
			h.Handle("/{inboxID}", alice.New(JSONContentType, s.CheckPermissionJSON).ThenFunc(fakeHandler))

			r := httptest.NewRequest(http.MethodGet, "/dafd5606-8aa8-4724-a2c5-f66110aba536", nil)
			r.Header.Set("X-Burner-Key", test.TokenToPass)

			h.ServeHTTP(rr, r)

			assert.Equal(t, test.ExpectedStatusCode, rr.Result().StatusCode)
		})
	}
}

func TestSetVersionHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	h := SetVersionHeader(http.HandlerFunc(fakeHandler))

	h.ServeHTTP(rr, req)

	if rr.Header().Get("X-Burner-Kiwi-Version") != version {
		t.Fatalf("TestSetVersionHeader: returned version header doesn't equal default header. Got %v, expected %v", rr.Header().Get("X-Burner-Kiwi-version"), version)
	}
}

func TestRefresh(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	r := Refresh(10)
	h := r(http.HandlerFunc(fakeHandler))

	h.ServeHTTP(rr, req)

	if rr.Header().Get("Refresh") != "10" {
		t.Fatalf("TestRefresh: returned refresh header doesn't equal expected header. Got %v, expected %v", rr.Header().Get("Refresh"), 10)
	}
}

func TestCacheControl(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	r := CacheControl(10)
	h := r(http.HandlerFunc(fakeHandler))

	h.ServeHTTP(rr, req)

	if rr.Header().Get("Cache-Control") != "max-age=10" {
		t.Fatalf("TestCahceControl: returned cache header doesn't equal expected header. Got %v, expected %v", rr.Header().Get("Cache-Control"), "max-age=10")
	}
}

func TestServer_SecurityHeaders_SelfServe(t *testing.T) {
	s := Server{
		cfg: Config{
			Developing: false,
			StaticURL:  "/static",
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	m := s.SecurityHeaders()
	h := m(http.HandlerFunc(fakeHandler))

	h.ServeHTTP(rr, req)

	assert.Equal(t, "default-src *; img-src *; font-src *; style-src * 'unsafe-inline'; script-src 'none';", rr.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1", rr.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "no-referrer", rr.Header().Get("Referrer-Policy"))
}

func TestRestoreRealIP(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.RemoteAddr))
	}
	handler := RestoreRealIP(http.HandlerFunc(h))

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "8.8.8.8"
	r.Header.Set("CF-Connecting-IP", "1.1.1.1")

	handler.ServeHTTP(rr, r)

	assert.Equal(t, "1.1.1.1", rr.Body.String())
}

func fakeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(FAKEHANDLERRESP))
}
