package server

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (s *Server) IndexJSON(w http.ResponseWriter, r *http.Request) {
	var e Email

	addr, err := s.eg.NewRandom()

	if err != nil {
		log.Printf("Index: failed to generate new random email: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	id, err := uuid.NewRandom()

	if err != nil {
		log.Printf("Index: failed to generate new random id: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	e.Address = addr
	e.ID = id.String()
	e.CreatedAt = time.Now().Unix()
	e.TTL = time.Now().Add(time.Hour * 24).Unix()

	// Create route and save to dynamodb
	go func(e *Email) {
		err := s.createRoute(e)

		if err != nil {
			log.Println(err)
		}

		err = s.saveEmail(e)

		if err != nil {
			log.Println(err)
		}
	}(&e)
}

func InboxJSON(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("success json"))
}
