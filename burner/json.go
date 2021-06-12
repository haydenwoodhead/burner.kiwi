package burner

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// Response is the root response for every api call
type Response struct {
	Success bool        `json:"success"`
	Errors  interface{} `json:"errors"`
	Result  interface{} `json:"result"`
}

// Errors is our error struct for if something goes wrong
type Errors struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type jwtToken struct {
	InboxID string
}

// NewInboxJSON generates a new email address and returns it to the caller
func (s *Server) NewInboxJSON(w http.ResponseWriter, r *http.Request) {
	i := NewInbox()
	i.Address = s.eg.NewRandom()

	exists, err := s.db.EmailAddressExists(i.Address) // while it's VERY unlikely that the email already exists but lets check anyway
	if err != nil {
		log.WithError(err).Error("JSON Index: failed to check if email exists")
		returnJSONError(w, r, http.StatusInternalServerError, "Failed to generate email")
		return
	}

	if exists {
		log.Error("JSON Index: email already exists")
		returnJSONError(w, r, http.StatusInternalServerError, "Failed to generate email")
		return
	}

	i.ID = uuid.Must(uuid.NewRandom()).String()
	i.CreatedAt = time.Now().Unix()
	i.TTL = time.Now().Add(time.Hour * 24).Unix()
	i.CreatedBy = r.RemoteAddr

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

	err = s.db.SaveNewInbox(i)
	if err != nil {
		log.WithError(err).Error("JSON Index: failed to save email")
		returnJSONError(w, r, http.StatusInternalServerError, "Failed to save email")
		return
	}

	token, err := s.notariser.Sign("auth", jwtToken{InboxID: i.ID}, i.TTL)
	if err != nil {
		log.WithError(err).Error("JSON Index: failed to generate auth toke")
		returnJSONError(w, r, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	res := struct {
		Inbox Inbox  `json:"email"`
		Token string `json:"token"`
	}{
		Inbox: i,
		Token: token,
	}

	// if we're using lambda then wait for our create route and update goroutine to finish before exiting the
	// func and therefore returning a response
	if s.cfg.UsingLambda {
		wg.Wait()
	}

	returnJSON(w, r, http.StatusOK, Response{
		Result:  res,
		Success: true,
	})
}

// GetInboxDetailsJSON returns details on a singular inbox by the given inbox id
func (s *Server) GetInboxDetailsJSON(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["inboxID"]

	e, err := s.db.GetInboxByID(id)
	if err != nil {
		log.WithError(err).WithField("inboxID", id).Printf("GetInboxDetailsJSON: failed to retrieve email from db")
		returnJSONError(w, r, http.StatusInternalServerError, "Failed to get email details")
		return
	}

	returnJSON(w, r, http.StatusOK, Response{
		Success: true,
		Result:  e,
	})
}

// GetAllMessagesJSON returns all messages in json
func (s *Server) GetAllMessagesJSON(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["inboxID"]

	m, err := s.db.GetMessagesByInboxID(id)
	if err != nil {
		log.WithError(err).WithField("inboxID", id).Error("GetAllMessagesJSON: failed to get messages with id")
		returnJSONError(w, r, http.StatusInternalServerError, "Failed to get messages")
		return
	}

	returnJSON(w, r, http.StatusOK, Response{
		Success: true,
		Result:  m,
	})
}

// returnJSONError returns json with custom error message
func returnJSONError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	returnJSON(w, r, status, Response{
		Success: false,
		Result:  nil,
		Errors: Errors{
			Code: 500,
			Msg:  msg,
		},
	})
}

func returnJSON(w http.ResponseWriter, r *http.Request, status int, resp interface{}) {
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	err := encoder.Encode(resp)
	if err != nil {
		log.WithError(err).Error("returnJSON: failed to write response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
