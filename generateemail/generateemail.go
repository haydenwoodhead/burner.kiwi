package generateemail

import (
	"fmt"
	"math/rand"
	"time"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz1234567890"

// EmailGenerator keeps track of hosts and min length needed by email generator methods
type EmailGenerator struct {
	hosts []string
	l     int
}

// NewEmailGenerator returns an email generator that creates emails with the given hosts and minimum length user part
func NewEmailGenerator(h []string, l int) *EmailGenerator {
	rand.Seed(time.Now().UTC().UnixNano())
	return &EmailGenerator{hosts: h, l: l}
}

// GetHosts returns all available hosts
func (eg *EmailGenerator) GetHosts() []string {
	return eg.hosts
}

// HostsContains tells whether host h is in eg.hosts
func (eg *EmailGenerator) HostsContains(h string) bool {
	for _, n := range eg.hosts {
		if h == n {
			return true
		}
	}
	return false
}

// NewRandom generates a new random email address. It is the callers responsibility to check for uniqueness
func (eg *EmailGenerator) NewRandom() string {
	a := []byte(alphabet)
	name := make([]byte, eg.l)

	for i := range name {
		name[i] = a[rand.Intn(len(a))]
	}

	domain := eg.hosts[rand.Intn(len(eg.hosts))]

	return string(name) + "@" + domain
}

// NewFromRouteAndHost generates a new email address from a string and host. It is the callers responsibility to check for uniqueness
func (eg *EmailGenerator) NewFromRouteAndHost(r string, h string) (string, error) {
	if eg.HostsContains(h) {
		return string(r) + "@" + h, nil
	}
	return "", fmt.Errorf("invalid host: %s", h)
}
