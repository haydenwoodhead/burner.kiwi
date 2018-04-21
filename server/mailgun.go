package server

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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
	id := vars["inboxID"]

	i, err := s.db.GetInboxByID(id)

	if err != nil {
		log.Printf("MailgunIncoming: failed to get inbox: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var m Message

	m.InboxID = i.ID
	m.TTL = i.TTL

	mID, err := uuid.NewRandom()

	if err != nil {
		log.Printf("MailgunIncoming: failed to generate uuid for inbox: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	m.ID = mID.String()

	m.ReceivedAt = time.Now().Unix()
	m.MGID = r.FormValue("message-id")
	m.Sender = r.FormValue("sender")
	m.From = r.FormValue("from")
	m.Subject = r.FormValue("subject")
	m.BodyPlain = r.FormValue("body-plain")

	html := r.FormValue("body-html")

	// Check to see if there is anything in html before we modify it. Otherwise we end up setting a blank html doc
	// on all plaintext emails preventing them from being displayed.
	if strings.Compare(html, "") != 0 {
		sr := strings.NewReader(html)

		doc, err := goquery.NewDocumentFromReader(sr)

		if err != nil {
			log.Printf("MailgunIncoming: failed to create goquery doc: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Find all a tags and add a target="_blank" attr to them so they open links in a new tab rather than in the iframe
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			s.SetAttr("target", "_blank")
		})

		modifiedHTML, err := doc.Html()

		if err != nil {
			log.Printf("MailgunIncoming: failed to get html doc: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		m.BodyHTML = modifiedHTML
	}

	err = s.db.SaveNewMessage(m)

	if err != nil {
		log.Printf("MailgunIncoming: failed to save message to db: %v", err)
	}

	_, err = w.Write([]byte(id))

	if err != nil {
		log.Printf("MailgunIncoming: failed to write response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
