package smtpmail

import (
	"net"
	"net/smtp"
	"strings"
	"testing"
	"time"

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

	s := SMTPMail{listener: &listener}

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
		"This is the email body.")
	err = mailHelper(listener.Addr().String(), "bob@example.com", to, smtpMsg)
	require.NoError(t, err)

	// I really hate needing to sleep in tests in order to coordinate goroutines.
	// However, I really want to test my full smtp implementation. This includes
	// my go-smtp interface functions not just my handleMessage func. I'm not
	// sure how i could do that without sleep.
	time.Sleep(2 * time.Second)

	mDB.AssertExpectations(t)
}

func TestSMTPMail_Multipart(t *testing.T) {
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
		From:      "Bob Simon <bob@example.com>",
		Subject:   "Please actually work please.....",
		BodyHTML:  `<html><head></head><body><div dir="ltr">Test content goes here</div></body></html>`,
		BodyPlain: "Test content goes here",
		TTL:       2,
	}

	mDB.On("SaveNewMessage", mock.MatchedBy(MessageMatcher(msg))).Return(nil)

	go func() {
		err := s.Start("example.com", mDB, nil, fakeIsBlackListed)
		require.NoError(t, err)
	}()

	to := []string{"test@example.com"}
	smtpMsg := []byte("MIME-Version: 1.0\r\n" +
		"Date: Sun, 5 Apr 2020 13:24:00 +1200\r\n" +
		"Message-ID: <some-really-long-id-with-lots-of-numbers@mail.example.com>\r\n" +
		"Subject: Please actually work please.....\r\n" +
		"From: Bob Simon <bob@example.com>\r\n" +
		"To: test@example.com\r\n" +
		"Content-Type: multipart/alternative; boundary=\"0000000000006ce1f305a281017b\"\r\n" +
		"\r\n" +
		"--0000000000006ce1f305a281017b\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
		"\r\n" +
		"Test content goes here\r\n" +
		"\r\n" +
		"--0000000000006ce1f305a281017b\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
		"\r\n" +
		"<div dir=\"ltr\">Test content goes here</div>\r\n" +
		"\r\n" +
		"--0000000000006ce1f305a281017b--")

	err = mailHelper(listener.Addr().String(), "bob@example.com", to, smtpMsg)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	mDB.AssertExpectations(t)
}

// https://github.com/golang/go/wiki/SendingMail
func mailHelper(addr, from string, rcpts []string, body []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()

	err = c.Mail(from)
	if err != nil {
		return err
	}

	for _, rcpt := range rcpts {
		err := c.Rcpt(rcpt)
		if err != nil {
			return err
		}
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
		return e.InboxID == message.InboxID &&
			e.Sender == message.Sender &&
			e.From == message.From &&
			e.Subject == message.Subject &&
			message.BodyHTML == e.BodyHTML &&
			strings.Contains(message.BodyPlain, e.BodyPlain) &&
			e.TTL == message.TTL
	}
}
