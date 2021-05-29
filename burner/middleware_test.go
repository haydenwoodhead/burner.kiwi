package burner

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/token"
	"github.com/justinas/alice"
)

const FAKEHANDLERRESP = "fake handler"
const NEWHANDLERRESP = "new handler"
const PASSKEY = "pass key"
const DONTCHECK = "string"

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

		if test.ExpectedMsg != DONTCHECK {
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

			if msg != test.ExpectedMsg {
				t.Errorf("TestServer_CheckPermissionJSON: %v - Message different. Expected %v, got %v", i, test.ExpectedMsg, msg)
			}
		}
	}
}

func TestSetVersionHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	h := SetVersionHeader(http.HandlerFunc(fakeHandler))

	h.ServeHTTP(rr, req)

	if rr.Header().Get("X-Burner-Kiwi-version") != version {
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
