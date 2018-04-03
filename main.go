package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/apex/gateway"
	"github.com/gorilla/context"
	"github.com/haydenwoodhead/burnerkiwi/server"
)

func main() {

	useLambda, err := strconv.ParseBool(os.Getenv("LAMBDA"))

	if err != nil {
		log.Fatalf("Failed to parse LAMBDA env var. Err = %v", err)
	}

	key := os.Getenv("KEY")

	if strings.Compare(key, "") == 0 {
		log.Fatalf("Env var key cannot be empty")
	}

	websiteURL := os.Getenv("WEBSITE_URL")

	if strings.Compare(websiteURL, "") == 0 {
		log.Fatalf("Env var WEBSITE_URL cannot be empty")
	}

	mgKey := os.Getenv("MG_KEY")

	if strings.Compare(mgKey, "") == 0 {
		log.Fatalf("Env var MG_KEY cannot be empty")
	}

	mgDomain := os.Getenv("MG_DOMAIN")

	if strings.Compare(mgKey, "") == 0 {
		log.Fatalf("Env var MG_KEY cannot be empty")
	}

	s, err := server.NewServer(key, websiteURL, mgDomain, mgKey, []string{"example.com"})

	if useLambda {
		log.Fatal(gateway.ListenAndServe("", context.ClearHandler(s.Router))) // wrap mux in ClearHandler as per docs to prevent leaking memory
	} else {
		log.Fatal(http.ListenAndServe(":8080", context.ClearHandler(s.Router)))
	}
}
