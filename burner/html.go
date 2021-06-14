package burner

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// Index either creates a new inbox or returns the existing one
func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	session := s.getSessionFromCookie(r)

	if session.IsNew {
		s.newRandomInbox(session, w, r)
		return
	}

	s.getInbox(session, w, r)
}

func (s *Server) getInbox(session *session, w http.ResponseWriter, r *http.Request) {
	id := session.InboxID
	i, err := s.db.GetInboxByID(id)
	if err != nil {
		log.WithField("inboxID", id).WithError(err).Error("Index: failed to get inbox")
		http.Error(w, "Failed to get inbox", http.StatusInternalServerError)
		return
	}

	// If we failed to register the mailgun route then delete the session cookie
	if i.FailedToCreate {
		err := session.Delete(w)
		if err != nil {
			log.WithField("inboxID", id).WithError(err).Error("Index: failed to clear session cookie")
			http.Error(w, "Failed to create inbox. Please clear your cookies and try again", http.StatusInternalServerError)
			return
		}

		http.Error(w, "Failed to create inbox. Please refresh.", http.StatusInternalServerError)
		return
	}

	msgs, err := s.db.GetMessagesByInboxID(id)
	if err != nil {
		log.WithField("inboxID", id).WithError(err).Error("Index: failed to get all messages for inbox")
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].ReceivedAt > msgs[j].ReceivedAt
	})

	vars := inboxOut{
		Static:   s.getStaticDetails(),
		Messages: transformMessagesForTemplate(msgs),
		Inbox:    transformInboxForTemplate(i),
	}

	err = s.getIndexTemplate().ExecuteTemplate(w, "base", vars)
	if err != nil {
		log.WithField("inboxID", id).WithError(err).Error("Index: failed to write template response")
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

// newRandomInbox generates a new Inbox with a random route and host from availabile options.
func (s *Server) newRandomInbox(session *session, w http.ResponseWriter, r *http.Request) {
	i := NewInbox()
	i.Address = s.eg.NewRandom()

	exists, err := s.db.EmailAddressExists(i.Address) // while it's VERY unlikely that the email address already exists but lets check anyway
	if err != nil {
		log.WithError(err).Error("NewRandomInbox: failed to check if email exists")
		http.Error(w, "Failed to generate email. Please refresh", http.StatusInternalServerError)
		return
	}

	if exists {
		log.Error("NewRandomInbox: duplicate random email created")
		http.Error(w, "Failed to generate email. Please refresh", http.StatusInternalServerError)
		return
	}

	err = s.createRouteFromInbox(session, i, r.RemoteAddr, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.getInbox(session, w, r)
}

// newNamedInbox generates a new Inbox with a specific route and host.
func (s *Server) NewNamedInbox(w http.ResponseWriter, r *http.Request) {
	session := s.getSessionFromCookie(r)

	address, err := s.eg.NewFromUserAndHost(r.PostFormValue("user"), r.PostFormValue("host"))
	if err != nil {
		log.WithError(err).Info("NewNamedInbox: failed to create new inbox address")
		s.editInbox(w, r, "Failed to create new inbox: bad address")
		return
	}

	i := NewInbox()
	i.Address = address

	exists, err := s.db.EmailAddressExists(i.Address)
	if err != nil {
		log.Printf("NewNamedInbox: failed to check if email exists: %v", err)
		s.editInbox(w, r, "Failed to create new inbox: try again")
		return
	}

	if exists {
		log.WithField("address", address).Debug("NewNamedInbox: email already exists")
		s.editInbox(w, r, "Failed to create new inbox: address in uses")
		return
	}

	err = s.createRouteFromInbox(session, i, r.RemoteAddr, w)
	if err != nil {
		log.WithError(err).Info("NewNamedInbox: failed to create new inbox address")
		http.Error(w, "Failed to create inbox. Please clear cookies and try again.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// CreateRouteFromInbox creates a new route based on Inbox settings and writes out the new inbox
func (s *Server) createRouteFromInbox(session *session, i Inbox, remoteAddr string, w http.ResponseWriter) error {
	i.ID = uuid.Must(uuid.NewRandom()).String()
	i.CreatedAt = time.Now().Unix()
	i.TTL = time.Now().Add(time.Hour * 24).Unix()
	i.CreatedBy = remoteAddr

	// Mailgun can take a really long time to register a route (sometimes up to 2 seconds) so
	// we should do this out of the request thread and then update our db with the results. However if we're using
	// lambda we need to make the request wait for this operation to finish. Otherwise the route will never
	// get created.
	var wg sync.WaitGroup

	if s.cfg.UsingLambda {
		wg.Add(1)
		go s.lambdaCreateRouteAndUpdate(&wg, i)
	} else {
		go s.createRouteAndUpdate(i)
	}

	err := s.db.SaveNewInbox(i)
	if err != nil {
		log.WithError(err).Error("CreateRouteFromInbox: failed to save new email")
		return fmt.Errorf("Failed to save new inbox: %w", err)
	}

	err = session.SetInboxID(i.ID, w)
	if err != nil {
		log.WithError(err).Error("CreateRouteFromInbox: failed to set session cookie")
		return fmt.Errorf("Failed to set session cookie: %w", err)
	}

	// if we're using lambda then wait for our create route and update goroutine to finish before exiting the
	// func and therefore returning a response
	if s.cfg.UsingLambda {
		wg.Wait()
	}

	return nil
}

// IndividualMessage returns a singular message to the user
func (s *Server) IndividualMessage(w http.ResponseWriter, r *http.Request) {
	inboxID := s.getSessionFromCookie(r).InboxID
	messageID := mux.Vars(r)["messageID"]

	inbox, err := s.db.GetInboxByID(inboxID)
	if err != nil {
		log.WithField("inboxID", inboxID).WithError(err).Error("IndividualMessage: failed to get inbox")
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	msgs, err := s.db.GetMessagesByInboxID(inboxID)
	if err != nil {
		log.WithField("inboxID", inboxID).WithError(err).Error("IndividualMessage: failed to get all messages for inbox")
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].ReceivedAt > msgs[j].ReceivedAt
	})

	templateMsgs := transformMessagesForTemplate(msgs)

	msg, ok := getIndividualMsgById(messageID, templateMsgs)
	if !ok {
		http.Error(w, "Message not found on burner.kiwi", http.StatusNotFound)
		return
	}

	vars := inboxOut{
		Static:             s.getStaticDetails(),
		Messages:           transformMessagesForTemplate(msgs),
		Inbox:              transformInboxForTemplate(inbox),
		SelectedMessage:    msg,
		HasSelectedMessage: true,
	}

	err = s.getIndexTemplate().ExecuteTemplate(w, "base", vars)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"inboxID": inboxID, "messageID": messageID}).Printf("IndividualMessage: failed to execute template: %v", err)
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

func getIndividualMsgById(id string, haystack []templateMessage) (templateMessage, bool) {
	for _, msg := range haystack {
		if msg.ID == id {
			return msg, true
		}
	}
	return templateMessage{}, false
}

func (s *Server) editInbox(w http.ResponseWriter, r *http.Request, errMessage string) {
	session := s.getSessionFromCookie(r)
	i, err := s.db.GetInboxByID(session.InboxID)
	if err != nil {
		log.WithField("inboxID", session.InboxID).WithError(err).Error("DeleteInbox: failed to get inbox")
		http.Error(w, "Failed to get inbox", http.StatusInternalServerError)
		return
	}

	msgs, err := s.db.GetMessagesByInboxID(i.ID)
	if err != nil {
		log.WithField("inboxID", i.ID).WithError(err).Error("DeleteInbox: failed to get all messages for inbox")
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].ReceivedAt > msgs[j].ReceivedAt
	})

	vars := inboxOut{
		Static:   s.getStaticDetails(),
		Messages: transformMessagesForTemplate(msgs),
		Inbox:    transformInboxForTemplate(i),
		ModalData: editModalData{
			Hosts: s.cfg.Domains,
			Err:   errMessage,
		},
	}

	err = s.getEditTemplate().ExecuteTemplate(w, "base", vars)
	if err != nil {
		log.Printf("DeleteInbox: failed to execute template: %v", err)
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

//EditInbox prompts the user for a new name for the inbox route
func (s *Server) EditInbox(w http.ResponseWriter, r *http.Request) {
	s.editInbox(w, r, "")
}

//DeleteInbox prompts for a confirmation to delete from the user
func (s *Server) DeleteInbox(w http.ResponseWriter, r *http.Request) {
	session := s.getSessionFromCookie(r)
	i, err := s.db.GetInboxByID(session.InboxID)
	if err != nil {
		log.WithField("inboxID", session.InboxID).WithError(err).Error("DeleteInbox: failed to get inbox")
		http.Error(w, "Failed to get inbox", http.StatusInternalServerError)
		return
	}

	msgs, err := s.db.GetMessagesByInboxID(i.ID)
	if err != nil {
		log.WithField("inboxID", i.ID).WithError(err).Error("DeleteInbox: failed to get all messages for inbox")
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].ReceivedAt > msgs[j].ReceivedAt
	})

	vars := inboxOut{
		Static:   s.getStaticDetails(),
		Messages: transformMessagesForTemplate(msgs),
		Inbox:    transformInboxForTemplate(i),
	}

	err = s.getDeleteTemplate().ExecuteTemplate(w, "base", vars)
	if err != nil {
		log.Printf("DeleteInbox: failed to execute template: %v", err)
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

//ConfirmDeleteInbox removes the user session cookie
func (s *Server) ConfirmDeleteInbox(w http.ResponseWriter, r *http.Request) {
	session := s.getSessionFromCookie(r)

	dlt, err := strconv.ParseBool(r.PostFormValue("really-delete"))
	if err != nil {
		log.Printf("ConfirmDeleteInbox: failed to parse really-delete %v", err)
		http.Error(w, "Failed to parse really-delete", http.StatusInternalServerError)
		return
	}

	if !dlt {
		http.Redirect(w, r, "/", http.StatusFound)
	}

	err = session.Delete(w)
	if err != nil {
		log.WithError(err).Error("ConfirmDeleteInbox: failed to delete user session")
		http.Error(w, "Failed to delete your session. Please clear your cookies", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
