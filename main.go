package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/context"
	"github.com/haydenwoodhead/burner.kiwi/server"
	"github.com/haydenwoodhead/gateway"
)

var runDelete bool

func init() {
	flag.BoolVar(&runDelete, "delete-old-routes", false, "when true will not run the server only delete old routes")
	flag.Parse()
}

func main() {
	nsi := mustParseNewServerInput()

	s, err := server.NewServer(nsi)
	if err != nil {
		log.Fatalf("Failed to setup new server: %v", err)
	}

	// if we are just running route delete then do so and return. Otherwise run runDeleteFunc in a goroutine
	if runDelete {
		runDeleteFunc(s)
		return
	}

	go func(s *server.Server) {
		if nsi.UsingLambda {
			runDeleteFunc(s)
		} else {
			for {
				time.Sleep(1 * time.Hour)
				log.Println("calling run delete func")
				runDeleteFunc(s)
			}
		}
	}(s)

	if nsi.UsingLambda {
		log.Fatal(gateway.ListenAndServe("", context.ClearHandler(s.Router))) // wrap mux in ClearHandler as per docs to prevent leaking memory
	} else {
		log.Fatal(http.ListenAndServe(":8080", context.ClearHandler(s.Router)))
	}
}

func runDeleteFunc(s *server.Server) {
	err := s.DeleteOldRoutes()
	if err != nil {
		log.Printf("Failed to call deleteOldRoutes: %v", err)
	}

	log.Printf("Route Delete finished.")
}
