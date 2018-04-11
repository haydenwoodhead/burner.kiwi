package server

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// MailgunIncoming receives incoming email webhooks from mailgun. It saves the email to
// the database. Any failures return a 500. Mailgun will then retry.
func (s *Server) MailgunIncoming(w http.ResponseWriter, r *http.Request) {
	ver, err := s.mg.VerifyWebhookRequest(r)

	if err != nil {
		log.Printf("MailgunIncoming: failed to verify request: %v", err)
	}

	if !ver {
		log.Printf("MailgunIncoming: invalid request")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["emailID"]

	e, err := s.getEmailByID(id)

	if err != nil {
		log.Printf("MailgunIncoming: failed to get email: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var m Message

	m.EmailID = e.ID
	m.TTL = e.TTL

	mID, err := uuid.NewRandom()

	if err != nil {
		log.Printf("MailgunIncoming: failed to generate uuid for email: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	m.ID = mID.String()

	m.ReceivedAt = time.Now().Unix()
	m.MGID = r.FormValue("message-id")
	m.Sender = r.FormValue("sender")
	m.From = r.FormValue("from")
	m.Subject = r.FormValue("subject")
	m.BodyHTML = r.FormValue("body-html")
	m.BodyPlain = r.FormValue("body-plain")

	err = s.saveNewMessage(m)

	if err != nil {
		log.Printf("MailgunIncomig: failed to save message to db: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(id))
}
