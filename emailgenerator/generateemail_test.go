package emailgenerator

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var H = []string{
	"example.com",
	"example.net",
	"example.org",
}

func TestNewEmailGenerator(t *testing.T) {
	g := New(H, 8)

	if !reflect.DeepEqual(g.Hosts, H) {
		t.Fatalf("TestNewEmailGenerator: Hosts not being saved correctly. Expected %v, got %v", H, g.Hosts)
	}
}

func TestEmailGenerator_NewRandom(t *testing.T) {
	g := New(H, 8)

	s := g.NewRandom()

	sections := strings.Split(s, "@")

	inH := inArray(sections[1], H)

	if !inH {
		t.Fatalf("TestEmailGenerator_NewRandom: domain not in given Hosts. Expected %v, got %v", H, sections[1])
	}

	match, err := regexp.Match("[a-z0-9]*", []byte(sections[0]))

	if err != nil {
		t.Fatalf("TestEmailGenerator_NewRandom: err produced: %v", err)
	}

	if !match {
		t.Fatalf("TestEmailGenerator_NewRandom: email contains illegal chars. Email: %v", s)
	}
}

func TestEmailGenerator_NewFromRouteAndHost(t *testing.T) {
	g := New([]string{
		"example.com",
		"example.org",
	}, 8)

	tests := []struct {
		Name      string
		In        string
		ExpectErr bool
	}{
		{
			Name:      "less than 3 chars",
			In:        "a",
			ExpectErr: true,
		},
		{
			Name:      "longer than 64 chars",
			In:        alphabet + alphabet + alphabet + alphabet,
			ExpectErr: true,
		},
		{
			Name:      "contains space",
			In:        "firstname lastname",
			ExpectErr: true,
		},
		{
			Name:      "contains non alphanumeric chars",
			In:        "bob!@#",
			ExpectErr: true,
		},
		{
			Name:      "blacklisted email",
			In:        "postmaster",
			ExpectErr: true,
		},
		{
			Name:      "normal email",
			In:        "firstnamelastname",
			ExpectErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			err := g.verifyUser(test.In)
			if test.ExpectErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func inArray(n string, h []string) bool {
	for i := 0; i < len(h); i++ {
		if n == h[i] {
			return true
		}
	}

	return false
}
