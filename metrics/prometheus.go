package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// IncomingEmails is the metric for incoming emails
	IncomingEmails = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "burner_kiwi_incoming_emails",
			Help: "number of incoming emails",
		},
		[]string{"action"},
	)
	// ActiveInboxes is the metric for currently active inboxes
	ActiveInboxes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "burner_kiwi_active_inboxes",
			Help: "number of inboxes able to receive emails",
		},
	)
)
