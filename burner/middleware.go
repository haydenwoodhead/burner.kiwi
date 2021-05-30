package burner

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/token"
	"github.com/justinas/alice"
	log "github.com/sirupsen/logrus"
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
			log.WithError(err).Warn("CheckPermissionJSON: failed to verify token")
			returnJSONError(w, r, http.StatusInternalServerError, "Something went wrong")
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

//SecurityHeaders sets a whole bunch of headers to secure the site
func (s *Server) SecurityHeaders() alice.Constructor {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// check to see if we are developing before forcing strict transport
			if !s.cfg.Developing {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			w.Header().Set("Content-Security-Policy", "default-src *; img-src *; font-src *; style-src * 'unsafe-inline'; script-src 'none';")

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
		w.Header().Set("X-Burner-Kiwi-Version", version)

		h.ServeHTTP(w, r)
	})
}

//RestoreRealIP uses the real ip of the request from the CF-Connecting-IP header
func RestoreRealIP(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("CF-Connecting-IP")
		if ip != "" {
			r.RemoteAddr = ip
		}
		h.ServeHTTP(w, r)
	})
}
