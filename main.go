package main

import (
	"log"
	"net/http"

	"github.com/gorilla/context"
	"github.com/haydenwoodhead/burner.kiwi/burner"
	"github.com/haydenwoodhead/gateway"
)

func main() {
	nsi := mustParseNewServerInput()

	s, err := burner.New(nsi)
	if err != nil {
		log.Fatalf("Failed to setup new burner: %v", err)
	}

	if nsi.UsingLambda {
		log.Fatal(gateway.ListenAndServe("", context.ClearHandler(s.Router))) // wrap mux in ClearHandler as per docs to prevent leaking memory
	} else {
		log.Fatal(http.ListenAndServe(":8080", context.ClearHandler(s.Router)))
	}
}
