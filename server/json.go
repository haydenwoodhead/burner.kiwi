package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
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

func GetMeta() Meta {
	return Meta{
		Version: "0.1 Alpha",
		By:      "Hayden Woodhead",
	}
}

// IndexJSON generates a new email address and returns it to the caller
func (s *Server) IndexJSON(w http.ResponseWriter, r *http.Request) {
	e := NewEmail()
	resp := Response{Meta: GetMeta()}

	// valid email
	ve := false

	// loop over our email generation section until we generate an email that's not already in our db
	// I'm being lazy here. I should really come up with a better email generation technique
	for i := 0; !ve; i++ {
		addr, err := s.eg.NewRandom()

		if err != nil {
			log.Printf("JSON Index: failed to generate new random email: %v", err)
			return500(w, r, "Failed to generate email")
			return
		}

		exist, err := s.emailExists(addr)

		if err != nil {
			log.Printf("JSON Index: failed to check if email exists: %v", err)
			return500(w, r, "Failed to generate email")
			return
		}

		// i.e our email is all fine
		if !exist {
			e.Address = addr
			ve = true
			break
		}

		// If we start looping too many times then return 500. Hopefully we shouldn't get here
		if i > 10 {
			log.Print("JSON Index: looped > 10 times in order to generate an email")
			return500(w, r, "Failed to generate email")
			return
		}
	}

	id, err := uuid.NewRandom()

	if err != nil {
		log.Printf("JSON Index: failed to generate new random id: %v", err)
		return500(w, r, "Failed to generate random id")
		return
	}

	e.ID = id.String()
	e.CreatedAt = time.Now().Unix()
	e.TTL = time.Now().Add(time.Hour * 24).Unix()

	// Mailgun takes a really long time to register a route (sometimes up to 2 seconds) so
	// we should do this out of the request thread and then update our db with the results
	go func(e Email) {
		err = s.createRoute(&e)

		if err != nil {
			log.Printf("Index JSON: failed to create route: %v", err)

			return
		}

		err := s.setEmailCreated(e)

		if err != nil {
			log.Printf("Index JSON: failed to update that route is created: %v", err)
		}
	}(e)

	err = s.saveNewEmail(e)

	if err != nil {
		log.Printf("JSON Index: failed to save email: %v", err)
		return500(w, r, "Failed to save email")
		return
	}

	token := s.signer.Sign([]byte(e.ID))

	res := struct {
		Email Email  `json:"email"`
		Token string `json:"token"`
	}{
		Email: e,
		Token: string(token),
	}

	resp.Success = true
	resp.Result = res

	jsonResp, err := json.Marshal(resp)

	if err != nil {
		log.Printf("JSON Index: failed to marshal result var: %v", err)
		return500(w, r, "Failed to marshal response")
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write(jsonResp)
}

func InboxJSON(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("success json"))
}

// return500 returns json with custom error message
func return500(w http.ResponseWriter, r *http.Request, msg string) {
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
		log.Println("JSON Index: failed to marshal error response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	w.Write(jsonResp)
}
