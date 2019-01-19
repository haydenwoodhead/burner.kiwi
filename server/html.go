package server

import (
	"log"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/haydenwoodhead/burner.kiwi/data"
	"github.com/haydenwoodhead/burner.kiwi/stringduration"
)

//staticDetails contains the names of the static files used in the project
type staticDetails struct {
	URL       string
	Milligram string
	Logo      string
	Normalize string
	Custom    string
}

//getStaticDetails returns current static details
func (s *Server) getStaticDetails() staticDetails {
	return staticDetails{
		URL:       s.staticURL,
		Milligram: milligram,
		Logo:      logo,
		Normalize: normalize,
		Custom:    custom,
	}
}

// indexOut contains data to be rendered by the index template
type indexOut struct {
	Static     staticDetails
	Messages   []data.Message
	ReceivedAt []string
	Inbox      data.Inbox
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
	Message          data.Message
}

// Index returns messages in inbox to user
func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	sess, ok := r.Context().Value(sessionCTXKey).(*sessions.Session)

	if !ok {
		log.Printf("Index: failed to get sess var. Sess not of type sessions.Session actual type: %v", reflect.TypeOf(sess))
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	id, ok := sess.Values["id"].(string)

	if !ok {
		log.Printf("Index: failed to get id from session. ID not of type string. ID actual type: %v", reflect.TypeOf(sess.Values["id"]))
		http.Error(w, "Failed to get session id", http.StatusInternalServerError)
		return
	}

	i, err := s.db.GetInboxByID(id)

	if err != nil {
		log.Printf("Index: failed to get inbox: %v", err)
		http.Error(w, "Failed to get inbox", http.StatusInternalServerError)
		return
	}

	// If we failed to register the mailgun route then delete the session cookie
	if i.FailedToCreate {
		sess.Options.MaxAge = -1
		err = sess.Save(r, w)

		if err != nil {
			log.Printf("Index: failed to delete session cookie: %v", err)
			http.Error(w, "Failed to create inbox. Please clear your cookies.", http.StatusInternalServerError)
			return
		}

		http.Error(w, "Failed to create inbox. Please refresh.", http.StatusInternalServerError)
		return
	}

	msgs, err := s.db.GetMessagesByInboxID(id)

	if err != nil {
		log.Printf("Index: failed to get all messages from inbox. %v", err)
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].ReceivedAt > msgs[j].ReceivedAt
	})

	received := data.GetReceivedDetails(msgs)

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
		log.Printf("Index: failed to write response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// NewInbox creates a new inbox and returns details to the user
func (s *Server) NewInbox(w http.ResponseWriter, r *http.Request) {
	i := data.NewInbox()
	sess, ok := r.Context().Value(sessionCTXKey).(*sessions.Session)

	if !ok {
		log.Printf("New Inbox: failed to get sess var. Sess not of type sessions.Session actual type: %v", reflect.TypeOf(sess))
		http.Error(w, "Failed to generate email", http.StatusInternalServerError)
		return
	}

	i.Address = s.eg.NewRandom()

	exist, err := s.db.EmailAddressExists(i.Address) // while it's VERY unlikely that the email address already exists but lets check anyway

	if err != nil {
		log.Printf("New Inbox: failed to check if email exists: %v", err)
		http.Error(w, "Failed to generate email", http.StatusInternalServerError)
		return
	}

	if exist {
		log.Printf("NewInbox: email already exisists: %v", err)
		http.Error(w, "Failed to generate email", http.StatusInternalServerError)
		return
	}

	id, err := uuid.NewRandom()

	if err != nil {
		log.Printf("Index: failed to generate new random id: %v", err)
		http.Error(w, "Failed to generate new random id", http.StatusInternalServerError)
		return
	}

	i.ID = id.String()
	i.CreatedAt = time.Now().Unix()
	i.TTL = time.Now().Add(time.Hour * 24).Unix()
	i.CreatedBy = r.RemoteAddr

	// Mailgun can take a really long time to register a route (sometimes up to 2 seconds) so
	// we should do this out of the request thread and then update our db with the results. However if we're using
	// lambda we need to make the request wait for this operation to finish. Otherwise the route will never
	// get created.
	var wg sync.WaitGroup

	if s.usingLambda {
		wg.Add(1)
		go s.lambdaCreateRouteAndUpdate(&wg, i)
	} else {
		go s.createRouteAndUpdate(i)
	}

	err = s.db.SaveNewInbox(i)

	if err != nil {
		log.Printf("NewInbox: failed to save email: %v", err)
		http.Error(w, "Failed to save new email", http.StatusInternalServerError)
		return
	}

	sess.Values["id"] = i.ID
	err = sess.Save(r, w)

	if err != nil {
		log.Printf("NewInbox: failed to set session cookie: %v", err)
		http.Error(w, "Failed to set session cookie", http.StatusInternalServerError)
		return
	}

	io := indexOut{
		Static:   s.getStaticDetails(),
		Messages: nil,
		Inbox:    i,
		Expires: expires{
			Hours:   "23",
			Minutes: "59",
		},
	}

	err = indexTemplate.ExecuteTemplate(w, "base", io)

	if err != nil {
		log.Printf("NewInbox: failed to write response: %v", err)
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}

	// if we're using lambda then wait for our create route and update goroutine to finish before exiting the
	// func and therefore returning a response
	if s.usingLambda {
		wg.Wait()
	}
}

// IndividualMessage returns a singular message to the user
func (s *Server) IndividualMessage(w http.ResponseWriter, r *http.Request) {
	sess, ok := r.Context().Value(sessionCTXKey).(*sessions.Session)

	if !ok {
		log.Printf("IndividualMessage: failed to get sess var. Sess not of type *sessions.Session actual type: %v", reflect.TypeOf(sess))
		http.Error(w, "Failed to get email", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	mID := vars["messageID"]

	iID, ok := sess.Values["id"].(string)

	if !ok {
		log.Printf("IndividualMessage: failed to get inbox id. Id not of type string. Actual type: %v", reflect.TypeOf(iID))
		http.Error(w, "Failed to get message", http.StatusInternalServerError)
		return
	}

	m, err := s.db.GetMessageByID(iID, mID)

	if err == data.ErrMessageDoesntExist {
		http.Error(w, "Message not found on server", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("IndividualMessage: failed to get message. Error: %v", err)
		http.Error(w, "Failed to get message", http.StatusInternalServerError)
		return
	}

	rtd := data.GetReceivedDetails([]data.Message{m})

	ra := time.Unix(m.ReceivedAt, 0)

	ras := ra.Format("Mon Jan 2 15:04:05")

	mo := messageOut{
		Static:           s.getStaticDetails(),
		ReceivedTimeDiff: rtd[0],
		ReceivedAt:       ras,
		Message:          m,
	}

	// If our html doesn't contain anything then render the plaintext version
	if strings.Compare(m.BodyHTML, "") == 0 {
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
	sess, ok := r.Context().Value(sessionCTXKey).(*sessions.Session)

	if !ok {
		log.Printf("ConfirmDeleteInbox: failed to get sess var. Sess not of type *sessions.Session actual type: %v", reflect.TypeOf(sess))
		http.Error(w, "Failed to get user session", http.StatusInternalServerError)
		return
	}

	err := r.ParseForm()

	if err != nil {
		log.Printf("ConfirmDeleteInbox: failed to parse form %v", err)
		http.Error(w, "Failed to parse form", http.StatusInternalServerError)
		return
	}

	dlt, err := strconv.ParseBool(r.FormValue("really-delete"))

	if err != nil {
		log.Printf("ConfirmDeleteInbox: failed to parse really-delete %v", err)
		http.Error(w, "Failed to parse really-delete", http.StatusInternalServerError)
		return
	}

	if !dlt {
		http.Redirect(w, r, "/", http.StatusFound)
	}

	sess.Options.MaxAge = -1
	err = sess.Save(r, w)

	if err != nil {
		log.Printf("ConfirmDeleteInbox: failed to delete user session %v", err)
		http.Error(w, "Failed to delete user session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
