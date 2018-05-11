package data

import (
	"fmt"
	"strings"
	"time"

	"github.com/haydenwoodhead/burner.kiwi/stringduration"
)

// Inbox contains data on a temporary inbox including its address and ttl
type Inbox struct {
	Address        string `dynamodbav:"email_address" json:"address"`
	ID             string `dynamodbav:"id" json:"id"`
	CreatedAt      int64  `dynamodbav:"created_at" json:"created_at"`
	TTL            int64  `dynamodbav:"ttl" json:"ttl"`
	MGRouteID      string `dynamodbav:"mg_routeid" json:"-"`
	FailedToCreate bool   `dynamodbav:"failed_to_create" json:"-"`
}

// NewInbox returns an inbox with failed to create and route id set. Upon successful registration of the mailgun route we set these as true.
func NewInbox() Inbox {
	return Inbox{
		FailedToCreate: true,
		MGRouteID:      "-",
	}
}

// Message contains details of an individual email message received by the server
type Message struct {
	InboxID    string `dynamodbav:"inbox_id" json:"-"`
	ID         string `dynamodbav:"message_id" json:"id"`
	ReceivedAt int64  `dynamodbav:"received_at" json:"received_at"`
	MGID       string `dynamodbav:"mg_id" json:"-"`
	Sender     string `dynamodbav:"sender" json:"sender"`
	From       string `dynamodbav:"from" json:"from"`
	Subject    string `dynamodbav:"subject" json:"subject"`
	BodyHTML   string `dynamodbav:"body_html" json:"body_html"`
	BodyPlain  string `dynamodbav:"body_plain" json:"body_plain"`
	TTL        int64  `dynamodbav:"ttl" json:"ttl"`
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
			received = append(received, fmt.Sprintf("Less than 30s ago"))
			continue
		}

		diff = diff.Round(time.Minute) // Round to nearest minute

		h, min := stringduration.GetHoursAndMinutes(diff)

		if strings.Compare(h, "0") != 0 {
			received = append(received, fmt.Sprintf("%vh %vm ago", h, min))
			continue
		}

		received = append(received, fmt.Sprintf("%vm ago", min))
	}

	return received
}
