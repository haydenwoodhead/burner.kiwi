package token

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/go-alone"
	"github.com/pkg/errors"
)

// ErrTokenExpired is returned when the given token's ttl in the past
var ErrTokenExpired = errors.New("token: token has expired")

// ErrInvalidToken is returned when the token has an invalid signature or is otherwise invalid
var ErrInvalidToken = errors.New("token: invalid token")

// Generator contains fields needed by NewToken and VerifyToken
type Generator struct {
	s      *goalone.Sword
	maxAge time.Duration
}

// NewGenerator takes a key and a max age for the token then returns a new token generator
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

	if err != nil {
		return "", ErrInvalidToken
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
