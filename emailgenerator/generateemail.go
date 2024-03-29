package emailgenerator

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz1234567890"

// EmailGenerator keeps track of Hosts and min length needed by email generator methods
type EmailGenerator struct {
	Hosts []string
	L     int
}

// New returns an email generator that creates emails with the given Hosts and minimum length user part
func New(h []string, l int) *EmailGenerator {
	return &EmailGenerator{Hosts: h, L: l}
}

// GetHosts returns all available Hosts
func (eg *EmailGenerator) GetHosts() []string {
	return eg.Hosts
}

// HostsContains tells whether host h is in eg.Hosts
func (eg *EmailGenerator) hostsContains(h string) bool {
	for _, n := range eg.Hosts {
		if h == n {
			return true
		}
	}
	return false
}

// NewRandom generates a new random email address. It is the callers responsibility to check for uniqueness
func (eg *EmailGenerator) NewRandom() string {
	a := []byte(alphabet)
	name := make([]byte, eg.L)

	for i := range name {
		name[i] = a[rand.Intn(len(a))]
	}

	domain := eg.Hosts[rand.Intn(len(eg.Hosts))]

	return string(name) + "@" + domain
}

// NewFromUserAndHost generates a new email address from a string and host. It is the callers responsibility to check for uniqueness
func (eg *EmailGenerator) NewFromUserAndHost(user string, host string) (string, error) {
	user = strings.ToLower(user)
	host = strings.ToLower(host)

	if err := eg.verifyUser(user); err != nil {
		return "", err
	}
	if err := eg.verifyHost(host); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s@%s", user, host), nil
}

var isAlphaNumeric = regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString

// VerifyUser verifies the local part of an email address is between 3 and 64 alphanumeric characters
func (eg *EmailGenerator) verifyUser(r string) error {
	if len(r) < 3 {
		return fmt.Errorf("route must be at least three characters: %s", r)
	} else if len(r) > 64 {
		return fmt.Errorf("route must be fewer than 64 characters: %s", r)
	} else if !isAlphaNumeric(r) {
		return fmt.Errorf("route may only contain letters (a-z, A-Z) and numbers (0-9): %s", r)
	} else if r == "webmaster" || r == "admin" || r == "postmaster" || r == "administrator" || r == "root" {
		return fmt.Errorf("route is blacklisted: %s", r)
	}
	return nil
}

// VerifyHost verifies the host part of an email address is not empty and is known to the application
func (eg *EmailGenerator) verifyHost(h string) error {
	if h == "" {
		return errors.New("host must not be an empty string")
	} else if !eg.hostsContains(h) {
		return fmt.Errorf("host not in list of known Hosts: %s", h)
	}
	return nil
}
