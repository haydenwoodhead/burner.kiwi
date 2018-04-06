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

	ve := false

	// loop over our email generation section until we generate an email that's not already in our db
	// I'm being lazy here. I should really come up with a better email generation technique
	for i := 0; !ve; i++ {
		addr, err := s.eg.NewRandom()

		if err != nil {
			log.Printf("NewEmail: failed to generate new random email: %v", err)
			returnHTML500(w, r, "Failed to generate email")
			return
		}

		exist, err := s.emailExists(addr)

		if err != nil {
			log.Printf("NewEmail: failed to check if email exists: %v", err)
			returnHTML500(w, r, "Failed to generate email")
			return
		}

		// i.e our email is all fine
		if !exist {
			e.Address = addr
			ve = true
			break
		}

		// If we start looping too many times then return 500. Hopefully we shouldn't get here
		if i > 10 {
			log.Print("NewEmail: looped > 10 times in order to generate an email")
			returnHTML500(w, r, "Failed to generate email")
			return
		}
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
