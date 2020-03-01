package smtpmail

import (
	"net"
	"net/smtp"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/haydenwoodhead/burner.kiwi/burner"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func fakeIsBlackListed(address string) bool {
	return false
}

func TestSMTPMail_SimpleText(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := &SMTPMail{listener: &listener}

	mDB := new(MockDatabase)
	mDB.On("GetInboxByAddress", "test@example.com").Return(burner.Inbox{
		Address:              "test@example.com",
		ID:                   "1234",
		CreatedBy:            "192.168.1.1",
		TTL:                  2,
		EmailProviderRouteID: "smtp",
		FailedToCreate:       false,
	}, nil)
	mDB.On("EmailAddressExists", "test@example.com").Return(true, nil)

	msg := burner.Message{
		InboxID:   "1234",
		Sender:    "bob@example.com",
		From:      "bob@example.com",
		Subject:   "discount Gophers!",
		BodyHTML:  "",
		BodyPlain: "This is the email body.",
		TTL:       2,
	}

	mDB.On("SaveNewMessage", mock.MatchedBy(MessageMatcher(msg))).Return(nil)

	go func() {
		err := s.Start("example.com", mDB, nil, fakeIsBlackListed)
		require.NoError(t, err)
	}()

	to := []string{"test@example.com"}
	smtpMsg := []byte("To: test@example.com\r\n" +
		"From: bob@example.com\r\n" +
		"Subject: discount Gophers!\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"This is the email body.\r\n")
	err = mailHelper(listener.Addr().String(), "bob@example.com", to, smtpMsg)
	require.NoError(t, err)

	mDB.AssertExpectations(t)
}

// https://github.com/golang/go/wiki/SendingMail
func mailHelper(addr, from string, rcpts []string, body []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()

	c.Mail(from)

	for _, rcpt := range rcpts {
		c.Rcpt(rcpt)
	}

	wc, err := c.Data()
	if err != nil {
		return err
	}
	defer wc.Close()

	_, err = wc.Write(body)
	if err != nil {
		return err
	}
	defer wc.Close()

	return nil
}

func MessageMatcher(e burner.Message) func(burner.Message) bool {
	return func(message burner.Message) bool {
		spew.Dump(e)
		spew.Dump(message)
		return e.InboxID == message.InboxID &&
			e.Sender == message.Sender &&
			e.From == message.From &&
			e.Subject == message.Subject &&
			e.BodyHTML == message.BodyHTML &&
			e.BodyPlain == message.BodyPlain &&
			e.TTL == message.TTL
	}
}
