package burner

import (
	"fmt"
	"strings"
	"time"

	"github.com/haydenwoodhead/burner.kiwi/stringduration"
)

type staticDetails struct {
	FontPath string
	CSS      string
	Logo     string
}

func (s *Server) getStaticDetails() staticDetails {
	return staticDetails{
		FontPath: s.cfg.StaticURL,
		CSS:      fmt.Sprintf("%s/%s", s.cfg.StaticURL, css),
		Logo:     fmt.Sprintf("%s/%s", s.cfg.StaticURL, "roger.svg"),
	}
}

type templateMessage struct {
	Message
	ReceivedAt   string
	AvatarLetter string
	AvatarColor  string
}

type templateInbox struct {
	Inbox
	Expires expires
}

type expires struct {
	Hours   string
	Minutes string
}

type inboxOut struct {
	Static             staticDetails
	Messages           []templateMessage
	Inbox              templateInbox
	SelectedMessage    templateMessage
	HasSelectedMessage bool
}

type messageOut struct {
	Static           staticDetails
	ReceivedTimeDiff string
	ReceivedAt       string
	Message          Message
}

type editOut struct {
	Static staticDetails
	Hosts  []string
	Error  string
}

func transformMessagesForTemplate(msgs []Message) []templateMessage {
	transformedMsgs := make([]templateMessage, 0, len(msgs))

	// loop over all messages and calculate how long ago the message was received
	// then append that string to received to be passed to the template
	for _, m := range msgs {
		received := calculateReceivedAt(m.ReceivedAt)
		avatarLetter, avatarColor := getAvatarDetails(m.From)
		transformedMsgs = append(transformedMsgs, templateMessage{
			Message:      m,
			ReceivedAt:   received,
			AvatarLetter: avatarLetter,
			AvatarColor:  avatarColor,
		})
	}

	return transformedMsgs
}

func calculateReceivedAt(t int64) string {
	diff := time.Since(time.Unix(t, 0))

	// if we received the email less than 30 seconds ago then write that out
	// because rounding the duration when less than 30seconds will give us 0 seconds
	if diff.Seconds() < 30 {
		return "Less than 30s ago"
	}

	diff = diff.Round(time.Minute) // Round to nearest minute

	h, min := stringduration.GetHoursAndMinutes(diff)

	if h != "0" {
		return fmt.Sprintf("%vh %vm ago", h, min)
	}

	return fmt.Sprintf("%vm ago", min)
}

func getAvatarDetails(sender string) (string, string) {
	var letter string
	for _, runeInString := range sender {
		letter = strings.ToUpper(string(runeInString))
		break
	}

	// TOODO: normalize unicode so we get a better distribution of colors
	if letter < "E" {
		return letter, "bg-red"
	} else if letter >= "E" && letter < "I" {
		return letter, "bg-yellow"
	} else if letter >= "I" && letter < "M" {
		return letter, "bg-green"
	} else if letter >= "M" && letter < "Q" {
		return letter, "bg-indigo"
	} else if letter >= "Q" && letter < "U" {
		return letter, "bg-purple"
	}

	return letter, "bg-pink"
}

func transformInboxForTemplate(i Inbox) templateInbox {
	expiration := time.Until(time.Unix(i.TTL, 0))
	h, m := stringduration.GetHoursAndMinutes(expiration)

	return templateInbox{
		Inbox: i,
		Expires: expires{
			Hours:   h,
			Minutes: m,
		},
	}
}
