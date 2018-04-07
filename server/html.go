package server

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
)

// Index checks to see if a session already exists for the user. If so it redirects them to their page otherwise
// it generates a new email address for them and then redirects.
func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("index"))
}

func (s *Server) NewEmail(w http.ResponseWriter, r *http.Request) {
	e := NewEmail()
	sess, ok := r.Context().Value("sess").(*sessions.Session)

	if !ok {
		log.Printf("New Email: failed to get sess var. Sess not of type sessions.Session actual type: %v", reflect.TypeOf(sess))
		returnHTML500(w, r, "Failed to generate email")
		return
	}

	e.Address = s.eg.NewRandom()

	exist, err := s.emailExists(e.Address) // while it's VERY unlikely that the email already exists but lets check anyway

	if err != nil {
		log.Printf("New Email: failed to check if email exists: %v", err)
		returnHTML500(w, r, "Failed to generate email")
		return
	}

	if exist {
		log.Printf("New Email: email already exisists: %v", err)
		returnHTML500(w, r, "Failed to generate email")
		return
	}

	id, err := uuid.NewRandom()

	if err != nil {
		log.Printf("Index: failed to generate new random id: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	e.ID = id.String()
	e.CreatedAt = time.Now().Unix()
	e.TTL = time.Now().Add(time.Hour * 24).Unix()

	// Mailgun can take a really long time to register a route (sometimes up to 2 seconds) so
	// we should do this out of the request thread and then update our db with the results
	go s.createRouteAndUpdate(e)

	err = s.saveNewEmail(e)

	if err != nil {
		log.Printf("NewEmail: failed to save email: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sess.Values["id"] = e.ID
	sess.Save(r, w)
	w.Write([]byte("new email"))
}

func InboxHTML(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("success"))
}

func returnHTML500(w http.ResponseWriter, r *http.Request, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(fmt.Sprintf("Internal Server Error: %v", msg)))
}
