package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/token"
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
func (s *Server) CheckCookieExists(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := s.store.Get(r, sessionStoreKey)

		if err != nil {
			log.Printf("CheckCookieExists: failed to decode session cookie.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

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

			http.Error(w, checkCookieExistsErrorResponse, http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// IsNew returns an alice middleware that checks whether or not a session cookie is set. If the session is set it passes
// the request on otherwise it redirects to the given http handler n.
func (s *Server) IsNew(n http.Handler) alice.Constructor {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, err := s.store.Get(r, sessionStoreKey)

			if err != nil {
				log.Printf("IsNew: failed to decode session cookie.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), sessionCTXKey, sess)

			if sess.IsNew {
				n.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

//CacheControl sets the Cache-Control header
func CacheControl(sec int) alice.Constructor {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%v", sec))

			h.ServeHTTP(w, r)
		})
	}
}

//Refresh sets how often the page should refresh
func Refresh(sec int) alice.Constructor {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Refresh", fmt.Sprintf("%v", sec))

			h.ServeHTTP(w, r)
		})
	}
}

const self = "'self'"

//SecurityHeaders sets a whole bunch of headers to secure the site
func (s *Server) SecurityHeaders(extStyle bool) alice.Constructor {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// check to see if we are developing before forcing strict transport
			if !s.developing {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			var styleSrc string
			var imgSrc string
			var fntSrc string

			if strings.Compare(s.staticURL, "/static") == 0 {
				styleSrc = self
				imgSrc = self
				fntSrc = self
			} else {
				styleSrc = s.staticURL
				imgSrc = s.staticURL
				fntSrc = s.staticURL
			}

			// if we're allowing external styles then override then csp
			if extStyle {
				styleSrc = "* 'unsafe-inline'"
				imgSrc = "*"
				fntSrc = "*"
			}

			csp := fmt.Sprintf("script-src 'none'; font-src %v https://fonts.gstatic.com/; style-src %v http://fonts.googleapis.com/; img-src %v; default-src 'self'", fntSrc, styleSrc, imgSrc)

			w.Header().Set("Content-Security-Policy", csp)

			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "no-referrer")

			h.ServeHTTP(w, r)
		})
	}
}

//SetVersionHeader adds a header with the current version
func SetVersionHeader(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Burner-Kiwi-version", version)

		h.ServeHTTP(w, r)
	})
}
