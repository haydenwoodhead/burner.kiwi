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

// IndexOut contains data to be rendered by the index template
type IndexOut struct {
	StaticURL string
	Messages  []Message
	Inbox     Inbox
	Expires   Expires
}

// Expires contains the number of hours and minutes the inbox is available for
type Expires struct {
	Hours   string
	Minutes string
}

// Index checks to see if a session already exists for the user. If so it redirects them to their page otherwise
// it generates a new email address for them and then redirects.
func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("index"))

	if err != nil {
		log.Printf("IndexHTML: failed to write response: %v", err)
		return
	}
}

// NewInbox creates a new inbox and returns details to the user
func (s *Server) NewInbox(w http.ResponseWriter, r *http.Request) {
	i := NewInbox()
	sess, ok := r.Context().Value(sessionCTXKey).(*sessions.Session)

	if !ok {
		log.Printf("New Inbox: failed to get sess var. Sess not of type sessions.Session actual type: %v", reflect.TypeOf(sess))
		returnHTML500(w, r, "Failed to generate email")
		return
	}

	i.Address = s.eg.NewRandom()

	exist, err := s.emailExists(i.Address) // while it's VERY unlikely that the email address already exists but lets check anyway

	if err != nil {
		log.Printf("New Inbox: failed to check if email exists: %v", err)
		returnHTML500(w, r, "Failed to generate email")
		return
	}

	if exist {
		log.Printf("NewInbox: email already exisists: %v", err)
		returnHTML500(w, r, "Failed to generate email")
		return
	}

	id, err := uuid.NewRandom()

	if err != nil {
		log.Printf("Index: failed to generate new random id: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	i.ID = id.String()
	i.CreatedAt = time.Now().Unix()
	i.TTL = time.Now().Add(time.Hour * 24).Unix()

	// Mailgun can take a really long time to register a route (sometimes up to 2 seconds) so
	// we should do this out of the request thread and then update our db with the results
	go s.createRouteAndUpdate(i)

	err = s.saveNewInbox(i)

	if err != nil {
		log.Printf("NewInbox: failed to save email: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sess.Values["id"] = i.ID
	err = sess.Save(r, w)

	if err != nil {
		log.Printf("NewInbox: failed to set session cookie: %v", err)
		returnHTML500(w, r, "Failed to set session cookie")
		return
	}

	io := IndexOut{
		StaticURL: s.staticURL,
		Messages:  nil,
		Inbox:     i,
		Expires: Expires{
			Hours:   "23",
			Minutes: "59",
		},
	}

	err = indexTemplate.ExecuteTemplate(w, "base", io)

	if err != nil {
		log.Printf("NewInbox: failed to write response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

}

func returnHTML500(w http.ResponseWriter, r *http.Request, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	_, err := w.Write([]byte(fmt.Sprintf("Internal Server Error: %v", msg)))

	if err != nil {
		log.Printf("returnHTML500: failed to write response: %v", err)
		return
	}
}
