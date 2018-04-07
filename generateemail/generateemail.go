package generateemail

import (
	"math/rand"
	"time"
)

const ALPHABET = "abcdefghijklmnopqrstuvwxyz1234567890"

type EmailGenerator struct {
	hosts []string
	l     int
}

func NewEmailGenerator(h []string, l int) *EmailGenerator {
	rand.Seed(time.Now().UTC().UnixNano())
	return &EmailGenerator{hosts: h, l: l}
}

// NewRandom generates a new random email address. It is the callers responsibility to check for uniqueness
func (eg *EmailGenerator) NewRandom() string {
	a := []byte(ALPHABET)
	name := make([]byte, eg.l)

	for i := range name {
		name[i] = a[rand.Intn(len(a))]
	}

	domain := eg.hosts[rand.Intn(len(eg.hosts))]

	return string(name) + "@" + domain
}
