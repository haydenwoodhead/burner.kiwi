package burner

import (
	"fmt"
	"time"

	"github.com/haydenwoodhead/burner.kiwi/stringduration"
)

// Inbox contains data on a temporary inbox including its address and ttl
type Inbox struct {
	Address              string `dynamodbav:"email_address" json:"address" db:"address"`
	ID                   string `dynamodbav:"id" json:"id" db:"id"`
	CreatedAt            int64  `dynamodbav:"created_at" json:"created_at" db:"created_at"`
	CreatedBy            string `dynamodbav:"created_by" json:"-" db:"created_by"`
	TTL                  int64  `dynamodbav:"ttl" json:"ttl" db:"ttl"`
	EmailProviderRouteID string `dynamodbav:"ep_routeid" json:"-" db:"ep_routeid"`
	FailedToCreate       bool   `dynamodbav:"failed_to_create" json:"-" db:"failed_to_create"`
}

// NewInbox returns an inbox with failed to create and route id set.
func NewInbox() Inbox {
	return Inbox{
		FailedToCreate:       false,
		EmailProviderRouteID: "-",
	}
}

// Message contains details of an individual email message received by the burner
type Message struct {
	InboxID         string `dynamodbav:"inbox_id" json:"-" db:"inbox_id"`
	ID              string `dynamodbav:"message_id" json:"id" db:"message_id"`
	ReceivedAt      int64  `dynamodbav:"received_at" json:"received_at" db:"received_at"`
	EmailProviderID string `dynamodbav:"ep_id" json:"-" db:"ep_id"`
	Sender          string `dynamodbav:"sender" json:"sender" db:"sender"`
	From            string `dynamodbav:"from" json:"from" db:"from_address"`
	Subject         string `dynamodbav:"subject" json:"subject" db:"subject"`
	BodyHTML        string `dynamodbav:"body_html" json:"body_html" db:"body_html"`
	BodyPlain       string `dynamodbav:"body_plain" json:"body_plain" db:"body_plain"`
	TTL             int64  `dynamodbav:"ttl" json:"ttl" db:"ttl"`
}

//GetReceivedDetails takes a slice of Message and returns a slice with a string corresponding to each msg
// with the details on when that message was received
func GetReceivedDetails(msgs []Message) []string {
	var received []string

	// loop over all messages and calculate how long ago the message was received
	// then append that string to received to be passed to the template
	for _, m := range msgs {
		diff := time.Since(time.Unix(m.ReceivedAt, 0))

		// if we received the email less than 30 seconds ago then write that out
		// because rounding the duration when less than 30seconds will give us 0 seconds
		if diff.Seconds() < 30 {
			received = append(received, "Less than 30s ago")
			continue
		}

		diff = diff.Round(time.Minute) // Round to nearest minute

		h, min := stringduration.GetHoursAndMinutes(diff)

		if h != "0" {
			received = append(received, fmt.Sprintf("%vh %vm ago", h, min))
			continue
		}

		received = append(received, fmt.Sprintf("%vm ago", min))
	}

	return received
}
