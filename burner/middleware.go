package burner

import (
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

const self = "'self'"

//SecurityHeaders sets a whole bunch of headers to secure the site
func (s *Server) SecurityHeaders(extStyle bool) alice.Constructor {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// check to see if we are developing before forcing strict transport
			if !s.cfg.Developing {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			var styleSrc string
			var imgSrc string
			var fntSrc string

			if strings.Compare(s.cfg.StaticURL, "/static") == 0 {
				styleSrc = self
				imgSrc = self
				fntSrc = self
			} else {
				styleSrc = s.cfg.StaticURL
				imgSrc = s.cfg.StaticURL
				fntSrc = s.cfg.StaticURL
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

//RestoreRealIP uses the real ip of the request from theCF-Connecting-IP header
func RestoreRealIP(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("CF-Connecting-IP")
		if ip != "" {
			r.RemoteAddr = ip
		}
		h.ServeHTTP(w, r)
	})
}
