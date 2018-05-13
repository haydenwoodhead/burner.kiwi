package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/data"
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
	i := data.NewInbox()
	resp := Response{Meta: GetMeta()}

	i.Address = s.eg.NewRandom()

	exist, err := s.db.EmailAddressExists(i.Address) // while it's VERY unlikely that the email already exists but lets check anyway

	if err != nil {
		log.Printf("JSON Index: failed to check if email exists: %v", err)
		returnJSON500(w, r, "Failed to generate email")
		return
	}

	if exist {
		log.Printf("JSON Index: email already exisists: %v", err)
		returnJSON500(w, r, "Failed to generate email")
		return
	}

	id, err := uuid.NewRandom()

	if err != nil {
		log.Printf("JSON Index: failed to generate new random id: %v", err)
		returnJSON500(w, r, "Failed to generate random id")
		return
	}

	i.ID = id.String()
	i.CreatedAt = time.Now().Unix()
	i.TTL = time.Now().Add(time.Hour * 24).Unix()

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
		log.Printf("JSON Index: failed to save email: %v", err)
		returnJSON500(w, r, "Failed to save email")
		return
	}

	token := s.tg.NewToken(i.ID)

	res := struct {
		Inbox data.Inbox `json:"email"`
		Token string     `json:"token"`
	}{
		Inbox: i,
		Token: token,
	}

	resp.Success = true
	resp.Result = res

	jsonResp, err := json.Marshal(resp)

	if err != nil {
		log.Printf("JSON Index: failed to marshal result var: %v", err)
		returnJSON500(w, r, "Failed to marshal response")
		return
	}

	_, err = w.Write(jsonResp)

	if err != nil {
		log.Printf("NewInboxJSON: failed to write response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)

	// if we're using lambda then wait for our create route and update goroutine to finish before exiting the
	// func and therefore returning a response
	if s.usingLambda {
		wg.Wait()
	}
}

// GetInboxDetailsJSON returns details on a singular inbox by the given inbox id
func (s *Server) GetInboxDetailsJSON(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["inboxID"]

	e, err := s.db.GetInboxByID(id)

	if err != nil {
		log.Printf("GetInboxDetailsJSON: failed to retrieve email from db: %v", err)
		returnJSON500(w, r, "Failed to get email details")
		return
	}

	res := Response{
		Success: true,
		Result:  e,
		Meta:    GetMeta(),
	}

	resJSON, err := json.Marshal(res)

	if err != nil {
		log.Printf("GetInboxDetailsJSON: failed to marhsal json: %v", err)
		returnJSON500(w, r, "Failed to marshal response")
		return
	}

	_, err = w.Write(resJSON)

	if err != nil {
		log.Printf("GetInboxDetailsJSON: failed to write response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// GetAllMessagesJSON returns all messages in json
func (s *Server) GetAllMessagesJSON(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["inboxID"]

	m, err := s.db.GetMessagesByInboxID(id)

	if err != nil {
		log.Printf("GetAllMessagesJSON: failed to get messages with id %v: %v", id, err)
		returnJSON500(w, r, "Failed to get messages")
		return
	}

	res := Response{
		Success: true,
		Result:  m,
		Meta:    GetMeta(),
	}

	resJSON, err := json.Marshal(res)

	if err != nil {
		log.Printf("GetAllMessagesJSON: failed to marhsal json: %v", err)
		returnJSON500(w, r, "Failed to marshal response")
		return
	}

	_, err = w.Write(resJSON)

	if err != nil {
		log.Printf("GetAllMessagesJSON: failed to write response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// returnJSON500 returns json with custom error message
func returnJSON500(w http.ResponseWriter, r *http.Request, msg string) {
	resp := Response{}
	resp.Success = false
	resp.Result = nil
	resp.Meta = GetMeta()
	resp.Errors = Errors{
		Code: 500,
		Msg:  "Internal Server Error: " + msg,
	}

	jsonResp, err := json.Marshal(resp)

	if err != nil {
		log.Printf("JSON Index: failed to marshal error response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	_, err = w.Write(jsonResp)

	if err != nil {
		log.Printf("returnJSON500: failed to write response: %v", err)
		return
	}
}

func returnJSONError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	resp := Response{}
	resp.Success = false
	resp.Result = nil
	resp.Meta = GetMeta()
	resp.Errors = Errors{
		Code: status,
		Msg:  msg,
	}

	jsonResp, err := json.Marshal(resp)

	if err != nil {
		returnJSON500(w, r, "Failed to marshal error response")
		return
	}

	w.WriteHeader(status)

	_, err = w.Write(jsonResp)

	if err != nil {
		log.Printf("returnJSONError: failed to write response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
