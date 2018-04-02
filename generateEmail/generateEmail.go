package generateEmail

import (
	"math/rand"
	"time"

	"github.com/speps/go-hashids"
)

const ALPHABET = "abcdefghijklmnopqrstuvwxyz1234567890"

type EmailGenerator struct {
	hosts []string
	h     *hashids.HashID
}

func NewEmailGenerator(hosts []string, salt string, ml int) *EmailGenerator {
	rand.Seed(time.Now().UTC().UnixNano())
	hd := hashids.NewData()
	hd.Salt = salt
	hd.MinLength = ml
	hd.Alphabet = ALPHABET
	h, _ := hashids.NewWithData(hd)
	return &EmailGenerator{hosts: hosts, h: h}
}

// NewRandom generates a new random email address. It is the callers responsibility to check for uniqueness
func (eg *EmailGenerator) NewRandom() (string, error) {
	n := rand.Intn(99999999)
	name, err := eg.h.Encode([]int{n})

	if err != nil {
		return "", err
	}

	domain := eg.hosts[rand.Intn(len(eg.hosts))]

	return name + "@" + domain, nil
}
