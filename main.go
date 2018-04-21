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
	useLambda := mustParseBoolVar("LAMBDA")
	key := mustParseStringVar("KEY")
	websiteURL := mustParseStringVar("WEBSITE_URL")
	staticURL := mustParseStringVar("STATIC_URL")
	mgKey := mustParseStringVar("MG_KEY")
	mgDomain := mustParseStringVar("MG_DOMAIN")
	developing := mustParseBoolVar("DEVELOPING")

	s, err := server.NewServer(key, websiteURL, staticURL, mgDomain, mgKey, []string{"rogerin.space"}, developing)

	if err != nil {
		log.Fatalf("Failed to setup new server: %v", err)
	}

	if useLambda {
		log.Fatal(gateway.ListenAndServe("", context.ClearHandler(s.Router))) // wrap mux in ClearHandler as per docs to prevent leaking memory
	} else {
		log.Fatal(http.ListenAndServe(":8080", context.ClearHandler(s.Router)))
	}
}

func mustParseStringVar(key string) (v string) {
	v = os.Getenv(key)

	if strings.Compare(v, "") == 0 {
		log.Fatalf("Env var %v cannot be empty", key)
	}

	return
}

func mustParseBoolVar(key string) (v bool) {
	val := mustParseStringVar(key)

	v, err := strconv.ParseBool(val)

	if err != nil {
		log.Fatalf("Failed to parse %v. It must be either true or false", key)
	}

	return
}
