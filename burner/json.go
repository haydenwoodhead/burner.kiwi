package burner

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Response is the root response for every api call
type Response struct {
	Success bool        `json:"success"`
	Errors  interface{} `json:"errors"`
	Result  interface{} `json:"result"`
	Meta    Meta        `json:"meta"`
}

// Errors is our error struct for if something goes wrong
type Errors struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// Meta contains our version number and by
type Meta struct {
	Version string `json:"version"`
	By      string `json:"by"`
}

// GetMeta returns meta info for json api responses
func GetMeta() Meta {
	return Meta{
		Version: version,
		By:      "Hayden Woodhead",
	}
}

// NewInboxJSON generates a new email address and returns it to the caller
func (s *Server) NewInboxJSON(w http.ResponseWriter, r *http.Request) {
	i := NewInbox()
	i.Address = s.eg.NewRandom()

	exists, err := s.db.EmailAddressExists(i.Address) // while it's VERY unlikely that the email already exists but lets check anyway
	if err != nil {
		log.Printf("JSON Index: failed to check if email exists: %v", err)
		returnJSONError(w, r, http.StatusInternalServerError, "Failed to generate email")
		return
	}

	if exists {
		log.Printf("JSON Index: email already exisists: %v", err)
		returnJSONError(w, r, http.StatusInternalServerError, "Failed to generate email")
		return
	}

	id, err := uuid.NewRandom()
	if err != nil {
		log.Printf("JSON Index: failed to generate new random id: %v", err)
		returnJSONError(w, r, http.StatusInternalServerError, "Failed to generate random id")
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

	if s.cfg.UsingLambda {
		wg.Add(1)
		go s.lambdaCreateRouteAndUpdate(&wg, i)
	} else {
		go s.createRouteAndUpdate(i)
	}

	err = s.db.SaveNewInbox(i)
	if err != nil {
		log.Printf("JSON Index: failed to save email: %v", err)
		returnJSONError(w, r, http.StatusInternalServerError, "Failed to save email")
		return
	}

	token := s.tg.NewToken(i.ID)

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
		Meta:    GetMeta(),
		Success: true,
	})
}

// GetInboxDetailsJSON returns details on a singular inbox by the given inbox id
func (s *Server) GetInboxDetailsJSON(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["inboxID"]

	e, err := s.db.GetInboxByID(id)
	if err != nil {
		log.Printf("GetInboxDetailsJSON: failed to retrieve email from db: %v", err)
		returnJSONError(w, r, http.StatusInternalServerError, "Failed to get email details")
		return
	}

	returnJSON(w, r, http.StatusOK, Response{
		Success: true,
		Result:  e,
		Meta:    GetMeta(),
	})
}

// GetAllMessagesJSON returns all messages in json
func (s *Server) GetAllMessagesJSON(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["inboxID"]

	m, err := s.db.GetMessagesByInboxID(id)

	if err != nil {
		log.Printf("GetAllMessagesJSON: failed to get messages with id %v: %v", id, err)
		returnJSONError(w, r, http.StatusInternalServerError, "Failed to get messages")
		return
	}

	returnJSON(w, r, http.StatusOK, Response{
		Success: true,
		Result:  m,
		Meta:    GetMeta(),
	})
}

// returnJSONError returns json with custom error message
func returnJSONError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	returnJSON(w, r, status, Response{
		Success: false,
		Result:  nil,
		Meta:    GetMeta(),
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
		log.Printf("returnJSON: failed to write response. err=%v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func recombineFromField(from []*mail.Address) string {
	var fromString strings.Builder
	for i, f := range from {
		if f == nil {
			continue
		}

		if f.Name != "" {
			if i == len(from)-1 {
				fromString.WriteString(fmt.Sprintf("%s <%s>", f.Name, f.Address))
			} else {
				fromString.WriteString(fmt.Sprintf("%s <%s>, ", f.Name, f.Address))
			}
		} else {
			if i == len(from)-1 {
				fromString.WriteString(f.Address)
			} else {
				fromString.WriteString(f.Address)
			}
		}
	}
	return fromString.String()
}
