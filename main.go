package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/apex/gateway"
	"github.com/gorilla/context"
	"github.com/haydenwoodhead/burnerkiwi/database/dynamodb"
	"github.com/haydenwoodhead/burnerkiwi/server"
)

var runDelete bool

func init() {
	flag.BoolVar(&runDelete, "delete-old-routes", false, "when true will not run the server only delete old routes")
	flag.Parse()
}

func main() {
	useLambda := mustParseBoolVar("LAMBDA")

	nsi := server.NewServerInput{
		Key:         mustParseStringVar("KEY"),
		URL:         mustParseStringVar("WEBSITE_URL"),
		StaticURL:   mustParseStringVar("STATIC_URL"),
		MGKey:       mustParseStringVar("MG_KEY"),
		MGDomain:    mustParseStringVar("MG_DOMAIN"),
		Developing:  mustParseBoolVar("DEVELOPING"),
		Domains:     mustParseSliceVar("DOMAINS"),
		UsingLambda: useLambda,
		Database:    dynamodb.GetNewDynamoDB(),
	}

	s, err := server.NewServer(nsi)

	// if we are just running route delete then do so and return. Otherwise run runDeleteFunc in a goroutine
	if runDelete {
		runDeleteFunc(s)
		return
	}

	go func(s *server.Server) {
		runDeleteFunc(s)
	}(s)

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

func mustParseSliceVar(key string) (v []string) {
	val := mustParseStringVar(key)
	split := strings.Split(val, ",")

	for _, s := range split {
		v = append(v, strings.TrimSpace(s))
	}

	return
}

func runDeleteFunc(s *server.Server) {
	routes, err := s.DeleteOldRoutes()

	if err != nil {
		log.Printf("Failed to call deleteOldRoutes: %v", err)
	}

	for _, route := range routes {
		log.Printf("Failed to process route id: %v; email: %v; desc: %v", route.ID, route.Expression, route.Description)
	}

	log.Printf("Route Delete finished.")
}
