package burner

import (
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
)

// Session Related constants
const sessionKey = "burner_kiwi"
const inboxIDKey = "inbox_id"

type session struct {
	InboxID string
	IsNew   bool
	cookie  *sessions.Session
	r       *http.Request
}

func (s *session) SetInboxID(inboxID string, w http.ResponseWriter) error {
	s.InboxID = inboxID
	s.cookie.Values[inboxIDKey] = inboxID
	err := s.cookie.Save(s.r, w)
	if err != nil {
		return fmt.Errorf("cookie - failed to save inbox id: %w", err)
	}
	return nil
}

func (s *session) Delete(w http.ResponseWriter) error {
	s.cookie.Options.MaxAge = -1
	err := s.cookie.Save(s.r, w)
	if err != nil {
		return fmt.Errorf("cookie - failed to delete cookie: %w", err)
	}
	return nil
}

func (s *Server) getSessionFromCookie(r *http.Request) session {
	cookie, _ := s.sessionStore.Get(r, sessionKey)

	session := session{
		cookie: cookie,
		r:      r,
	}

	if !cookie.IsNew {
		id, ok := cookie.Values[inboxIDKey].(string)
		if !ok || id == "" {
			session.InboxID = ""
			session.IsNew = true
		} else {
			session.InboxID = id
			session.IsNew = false
		}
	} else {
		session.InboxID = ""
		session.IsNew = true
	}

	return session
}

const checkCookieExistsErrorResponse = "You do not have permission to view this item. Your session may have expired or you may be viewing this on a different device."

// CheckSessionExists checks for the existence of the session cookie and displays and error if false
func (s *Server) CheckSessionCookieExists(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := s.getSessionFromCookie(r)
		if session.IsNew {
			http.Error(w, checkCookieExistsErrorResponse, http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}
