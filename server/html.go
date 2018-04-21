package server

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
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

// Index returns messages in inbox to user
func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	sess, ok := r.Context().Value(sessionCTXKey).(*sessions.Session)

	if !ok {
		log.Printf("Index: failed to get sess var. Sess not of type sessions.Session actual type: %v", reflect.TypeOf(sess))
		returnHTML500(w, r, "Failed to get session")
		return
	}

	id, ok := sess.Values["id"].(string)

	if !ok {
		log.Printf("Index: failed to get id from session. ID not of type string. ID actual type: %v", reflect.TypeOf(sess.Values["id"]))
		returnHTML500(w, r, "Failed to get session id")
		return
	}

	i, err := s.getInboxByID(id)

	if err != nil {
		log.Printf("Index: failed to get inbox: %v", err)
		returnHTML500(w, r, "Failed to get inbox")
		return
	}

	// If we failed to register the mailgun route then delete the session cookie
	if i.FailedToCreate {
		sess.Options.MaxAge = -1
		err = sess.Save(r, w)

		if err != nil {
			log.Printf("Index: failed to delete session cookie: %v", err)
			returnHTML500(w, r, "Failed to create inbox. Please clear your cookies.")
			return
		}

		returnHTML500(w, r, "Failed to create inbox. Please refresh.")
		return
	}

	msgs, err := s.getAllMessagesByInboxID(id)

	if err != nil {
		log.Printf("Index: failed to get all messages from inbox. %v", err)
		returnHTML500(w, r, "Failed to get messages")
		return
	}

	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].ReceivedAt > msgs[j].ReceivedAt
	})

	received := getReceivedDetails(msgs)

	expiration := time.Until(time.Unix(i.TTL, 0))
	h, m := GetHoursAndMinutes(expiration)

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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// IndividualMessage returns a singular message to the user
func (s *Server) IndividualMessage(w http.ResponseWriter, r *http.Request) {
	sess, ok := r.Context().Value(sessionCTXKey).(*sessions.Session)

	if !ok {
		log.Printf("IndividualMessage: failed to get sess var. Sess not of type *sessions.Session actual type: %v", reflect.TypeOf(sess))
		returnHTML500(w, r, "Failed to get email")
		return
	}

	vars := mux.Vars(r)
	mID := vars["messageID"]

	iID, ok := sess.Values["id"].(string)

	if !ok {
		log.Printf("IndividualMessage: failed to get inbox id. Id not of type string. Actual type: %v", reflect.TypeOf(iID))
		returnHTML500(w, r, "Failed to get message")
		return
	}

	m, err := s.getMessageByID(iID, mID)

	if err == errMessageDoesntExist {
		returnHTMLError(w, r, http.StatusNotFound, "Message not found on server")
		return
	} else if err != nil {
		log.Printf("IndividualMessage: failed to get message. Error: %v", err)
		returnHTML500(w, r, "Failed to get message")
		return
	}

	rtd := getReceivedDetails([]Message{m})

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
			w.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	err = messageHTMLTemplate.ExecuteTemplate(w, "base", mo)

	if err != nil {
		log.Printf("IndividualMessage: failed to execute template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
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
		w.WriteHeader(http.StatusInternalServerError)
	}
}

//ConfirmDeleteInbox removes the user session cookie
func (s *Server) ConfirmDeleteInbox(w http.ResponseWriter, r *http.Request) {
	sess, ok := r.Context().Value(sessionCTXKey).(*sessions.Session)

	if !ok {
		log.Printf("ConfirmDeleteInbox: failed to get sess var. Sess not of type *sessions.Session actual type: %v", reflect.TypeOf(sess))
		returnHTML500(w, r, "Failed to get user session")
		return
	}

	err := r.ParseForm()

	if err != nil {
		log.Printf("ConfirmDeleteInbox: failed to parse form %v", err)
		returnHTML500(w, r, "Failed to parse form")
		return
	}

	delete, err := strconv.ParseBool(r.FormValue("really-delete"))

	if err != nil {
		log.Printf("ConfirmDeleteInbox: failed to parse really-delete %v", err)
		returnHTML500(w, r, "Failed to parse form")
		return
	}

	if !delete {
		http.Redirect(w, r, "/", http.StatusFound)
	}

	sess.Options.MaxAge = -1
	err = sess.Save(r, w)

	if err != nil {
		log.Printf("ConfirmDeleteInbox: failed to delete user session %v", err)
		returnHTML500(w, r, "Failed to delete user session")
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// TODO: refactor to remove duplicate functions returnHTML500 and returnHTML error and do the same for json
func returnHTML500(w http.ResponseWriter, r *http.Request, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	_, err := w.Write([]byte(fmt.Sprintf("Internal Server Error: %v", msg)))

	if err != nil {
		log.Printf("returnHTML500: failed to write response: %v", err)
		return
	}
}

// ErrorPrinter are funcs that are used to send specific error messages with codes to users
type ErrorPrinter func(w http.ResponseWriter, r *http.Request, code int, msg string)

func returnHTMLError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	w.WriteHeader(code)
	_, err := w.Write([]byte(msg))

	if err != nil {
		log.Printf("returnHTML500: failed to write response: %v", err)
		return
	}
}
