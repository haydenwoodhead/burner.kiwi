package server

import (
	"log"
	"net/http"
	"strings"
)

// DeleteOldRoutesEndpoint authenticates the given request then calls DeleteOldRoutes
func (s *Server) DeleteOldRoutesEndpoint(w http.ResponseWriter, r *http.Request) {
	k := r.Header.Get("X-Burner-Delete-Key")

	if strings.Compare(k, s.deleteKey) != 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	f, err := s.DeleteOldRoutes()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("DeleteOldRoutesEndpoint: Failed to call DeleteOldRoutes: %v", err)
		return
	}

	if len(f) > 0 {
		for _, route := range f {
			log.Printf("Failed to process route id: %v; email: %v; desc: %v", route.ID, route.Expression, route.Description)
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
