package main

import (
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/burner"
	"github.com/haydenwoodhead/gateway"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	cfg, db, email, listenAddr := mustParseConfig()
	s, err := burner.New(cfg, db, email)
	if err != nil {
		log.Fatalf("Failed to setup new burner: %v", err)
	}

	log.Info("Starting burner.kiwi")
	if cfg.UsingLambda {
		log.Fatal(gateway.ListenAndServe("", context.ClearHandler(s.Router))) // wrap mux in ClearHandler as per docs to prevent leaking memory
		return
	}

	if cfg.EmitMetrics {
		metricMux := mux.NewRouter()
		metricMux.Handle("/metrics", promhttp.Handler())
		go func() {
			log.Fatal(http.ListenAndServe(cfg.MetricPort, context.ClearHandler(metricMux)))
		}()
	}

	log.Fatal(http.ListenAndServe(listenAddr, context.ClearHandler(s.Router)))
}
