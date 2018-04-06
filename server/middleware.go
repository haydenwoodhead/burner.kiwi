package server

import (
	"context"
	"net/http"

	"github.com/justinas/alice"
)

// JSONContentType sets content type of request to json
func JSONContentType(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)
	})
}

// IsNew returns an alice middleware that checks whether or not a session cookie is set. If the session is set it passes
// the request on otherwise it redirects to the given http handler n.
func (s *Server) IsNew(n http.Handler) alice.Constructor {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, _ := s.store.Get(r, "session")
			ctx := context.WithValue(r.Context(), "sess", sess)

			if sess.IsNew {
				n.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
