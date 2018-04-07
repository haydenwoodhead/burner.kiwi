package token

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/go-alone"
	"github.com/pkg/errors"
)

var ErrTokenExpired = errors.New("token: token has expired")
var ErrInvalidSig = errors.New("token: invalid signature")

type Generator struct {
	s      *goalone.Sword
	maxAge time.Duration
}

// NewTokenGenerator takes a key and a max age for the token then returns a new token generator
func NewGenerator(k string, m time.Duration) *Generator {
	return &Generator{s: goalone.New([]byte(k)), maxAge: m}
}

// NewToken returns a signed id using the TokenGenerators key and maxAge
func (tg *Generator) NewToken(id string) string {
	exp := time.Now().Add(tg.maxAge).UTC().Unix()
	tk := fmt.Sprintf("%v.%v", id, exp)

	return string(tg.s.Sign([]byte(tk)))
}

// VerifyToken returns an id from the given token or an error
func (tg *Generator) VerifyToken(t string) (string, error) {
	tByte, err := tg.s.Unsign([]byte(t))

	if err == goalone.ErrInvalidSignature {
		return "", ErrInvalidSig
	} else if err != nil {
		return "", err
	}

	parts := strings.Split(string(tByte), ".")

	tInt64, err := strconv.ParseInt(parts[1], 10, 64)

	if err != nil {
		return "", err
	}

	tme := time.Unix(tInt64, 0)

	if tme.Before(time.Now()) {
		return "", ErrTokenExpired
	}

	return parts[0], nil
}
