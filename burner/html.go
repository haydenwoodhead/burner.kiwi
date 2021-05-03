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
	"github.com/haydenwoodhead/burner.kiwi/stringduration"
	log "github.com/sirupsen/logrus"
)

//staticDetails contains the names of the static files used in the project
type staticDetails struct {
	URL       string
	Milligram string
	Logo      string
	Normalize string
	Custom    string
	Icons     string
}

//getStaticDetails returns current static details
func (s *Server) getStaticDetails() staticDetails {
	return staticDetails{
		URL:       s.cfg.StaticURL,
		Milligram: milligram,
		Logo:      logo,
		Normalize: normalize,
		Custom:    custom,
		Icons:     icons,
	}
}

// indexOut contains data to be rendered by the index template
type indexOut struct {
	Static     staticDetails
	Messages   []Message
	ReceivedAt []string
	Inbox      Inbox
	Expires    expires
}

// expires contains a number of hours and minutes for use in displaying time
type expires struct {
	Hours   string
	Minutes string
}

// messageOut contains data to be rendered by message template
type messageOut struct {
	Static           staticDetails
	ReceivedTimeDiff string
	ReceivedAt       string
	Message          Message
}

// editOut contains data to be rendered by edit template
type editOut struct {
	Static staticDetails
	Hosts  []string
	Error  string
}

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

	received := GetReceivedDetails(msgs)

	expiration := time.Until(time.Unix(i.TTL, 0))
	h, m := stringduration.GetHoursAndMinutes(expiration)

	io := indexOut{
		Static:     s.getStaticDetails(),
		Messages:   msgs,
		Inbox:      i,
		ReceivedAt: received,
		Expires: expires{
			Hours:   h,
			Minutes: m,
		},
	}

	err = indexTemplate.ExecuteTemplate(w, "base", io)
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

	errs := ""

	route := r.PostFormValue("route")
	err := s.eg.VerifyUser(route)
	if err != nil {
		log.Printf("NewNamedInbox: failed to verify route: %v", err)
		errs = err.Error()
	}

	host := r.PostFormValue("host")
	err = s.eg.VerifyHost(host)
	if err != nil {
		log.Printf("NewNamedInbox: failed to verify host: %v", err)
		errs = err.Error()
	}

	address, err := s.eg.NewFromUserAndHost(route, host)
	if err != nil {
		log.Printf("NewNamedInbox: failed to create new inbox address: %v", err)
		http.Error(w, "Failed to generate email", http.StatusInternalServerError)
		return
	}

	i := NewInbox()
	i.Address = address

	exists, err := s.db.EmailAddressExists(i.Address)
	if err != nil {
		log.Printf("NewNamedInbox: failed to check if email exists: %v", err)
		http.Error(w, "Failed to generate email", http.StatusInternalServerError)
		return
	}

	if exists {
		log.Printf("NewNamedInbox: email already exists: %v", i.Address)
		errs = fmt.Sprintf("address already in use: %v", i.Address)
	}

	if errs != "" {
		eo := editOut{
			Static: s.getStaticDetails(),
			Hosts:  s.eg.GetHosts(),
			Error:  errs,
		}

		err := editTemplate.ExecuteTemplate(w, "base", eo)

		if err != nil {
			log.Printf("NewNamedInbox: failed to execute template: %v", err)
			http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		}
	} else {
		err := s.createRouteFromInbox(session, i, r.RemoteAddr, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusFound)
	}
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
	session := s.getSessionFromCookie(r)
	iID := session.InboxID

	vars := mux.Vars(r)
	mID := vars["messageID"]

	m, err := s.db.GetMessageByID(iID, mID)
	if err == ErrMessageDoesntExist {
		http.Error(w, "Message not found on burner.kiwi", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("IndividualMessage: failed to get message. Error: %v", err)
		http.Error(w, "Failed to get message", http.StatusInternalServerError)
		return
	}

	rtd := GetReceivedDetails([]Message{m})
	ra := time.Unix(m.ReceivedAt, 0)
	ras := ra.Format("Mon Jan 2 15:04:05")
	mo := messageOut{
		Static:           s.getStaticDetails(),
		ReceivedTimeDiff: rtd[0],
		ReceivedAt:       ras,
		Message:          m,
	}

	// If our html doesn't contain anything then render the plaintext version
	if m.BodyHTML == "" {
		err = messagePlainTemplate.ExecuteTemplate(w, "base", mo)

		if err != nil {
			log.Printf("IndividualMessage: failed to execute template: %v", err)
			http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		}

		return
	}

	err = messageHTMLTemplate.ExecuteTemplate(w, "base", mo)
	if err != nil {
		log.Printf("IndividualMessage: failed to execute template: %v", err)
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

//EditInbox prompts the user for a new name for the inbox route
func (s *Server) EditInbox(w http.ResponseWriter, r *http.Request) {
	eo := editOut{
		Static: s.getStaticDetails(),
		Hosts:  s.eg.GetHosts(),
		Error:  "",
	}

	err := editTemplate.ExecuteTemplate(w, "base", eo)

	if err != nil {
		log.Printf("DeleteInbox: failed to execute template: %v", err)
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

//DeleteInbox prompts for a confirmation to delete from the user
func (s *Server) DeleteInbox(w http.ResponseWriter, r *http.Request) {
	err := deleteTemplate.ExecuteTemplate(w, "base", struct {
		Static staticDetails
	}{
		Static: s.getStaticDetails(),
	})

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
