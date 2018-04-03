package server

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Index checks to see if a session already exists for the user. If so it redirects them to their page otherwise
// it generates a new email address for them and then redirects.
func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.store.Get(r, "session")

	if !sess.IsNew {
		id, ok := sess.Values["email_id"].(string)

		if !ok {
			log.Printf("Index: session id value not a string")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, s.websiteURL+"/inbox/"+id, 302)
	}

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

	sess.Values["email_id"] = e.ID
	sess.Values["email"] = e.Address
	sess.Values["ttl"] = e.TTL
	sess.Save(r, w)
	http.Redirect(w, r, s.websiteURL+"/inbox/"+id.String(), 302)
}

func InboxHTML(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("success"))
}
