package burner

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
	FromName        string `dynamodbav:"fromName" json:"fromName" db:"from_name"`
	FromAddress     string `dynamodbav:"fromEmail" json:"fromAddress" db:"from_address"`
	Subject         string `dynamodbav:"subject" json:"subject" db:"subject"`
	BodyHTML        string `dynamodbav:"body_html" json:"body_html" db:"body_html"`
	BodyPlain       string `dynamodbav:"body_plain" json:"body_plain" db:"body_plain"`
	TTL             int64  `dynamodbav:"ttl" json:"ttl" db:"ttl"`
}
