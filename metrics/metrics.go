package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "burner_kiwi"

var EmailsReceived = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "emails_received",
})

var InboxesCreated = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "inboxes_created",
}, []string{"content_type", "style"})
