package server

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burnerkiwi/token"
	"github.com/justinas/alice"
)

// JSONContentType sets content type of request to json
func JSONContentType(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)
	})
}

// CheckPermissionJSON checks whether or not the user has permission to call a url
func (s *Server) CheckPermissionJSON(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k := r.Header.Get("X-Burner-Key")

		id, err := s.tg.VerifyToken(k) // id from auth key

		if err == token.ErrInvalidToken {
			returnJSONError(w, r, http.StatusUnauthorized, "Unauthorized: given auth key invalid")
			return
		} else if err == token.ErrTokenExpired {
			returnJSONError(w, r, http.StatusForbidden, "Forbidden: your token has expired")
			return
		} else if err != nil {
			log.Printf("CheckPermissionJSON: failed to verify token: %v", err)
			returnJSON500(w, r, "Something went wrong")
			return
		}

		vars := mux.Vars(r)
		urlID := vars["inboxID"] // email id in url

		if id != urlID {
			returnJSONError(w, r, http.StatusForbidden, "Forbidden: you do not have permission to access this resource")
			return
		}

		h.ServeHTTP(w, r)
	})
}

const checkCookieExistsErrorResponse = "You do not have permission to view this message. Your session cookie may have expired or you may be viewing this on a different device."

// CheckCookieExists checks for the existence of the session cookie and displays and error if false
// using the provided ErrorPrinter
func (s *Server) CheckCookieExists(f ErrorPrinter) alice.Constructor {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, _ := s.store.Get(r, sessionStoreKey)
			ctx := context.WithValue(r.Context(), sessionCTXKey, sess)

			// if the session is new we know that the user doesn't have permission to view the page they're requesting
			if sess.IsNew {
				// we need to delete this session we just accidentally created otherwise when the user goes to load
				// the main page they wont be directed to the inbox creation handler
				sess.Options.MaxAge = -1
				err := sess.Save(r, w)

				if err != nil {
					log.Printf("CheckCookieExists: failed to delete accidenatally created session cookie.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				f(w, r, http.StatusUnauthorized, checkCookieExistsErrorResponse)
				return
			}

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// IsNew returns an alice middleware that checks whether or not a session cookie is set. If the session is set it passes
// the request on otherwise it redirects to the given http handler n.
func (s *Server) IsNew(n http.Handler) alice.Constructor {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, _ := s.store.Get(r, sessionStoreKey)
			ctx := context.WithValue(r.Context(), sessionCTXKey, sess)

			if sess.IsNew {
				n.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
