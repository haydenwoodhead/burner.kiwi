package generateemail

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var H = []string{
	"example.com",
	"example.net",
	"example.org",
}

func TestNewEmailGenerator(t *testing.T) {
	g := NewEmailGenerator(H, "testtesttest1234", 8)

	if !reflect.DeepEqual(g.hosts, H) {
		t.Fatalf("TestNewEmailGenerator: hosts not being saved correctly. Expected %v, got %v", H, g.hosts)
	}
}

func TestEmailGenerator_NewRandom(t *testing.T) {
	g := NewEmailGenerator(H, "testtesttest1234", 8)

	s, err := g.NewRandom()

	if err != nil {
		t.Fatalf("TestEmailGenerator_NewRandom: err produced: %v", err)
	}

	sections := strings.Split(s, "@")

	inH := inArray(sections[1], H)

	if !inH {
		t.Fatalf("TestEmailGenerator_NewRandom: domain not in given hosts. Expected %v, got %v", H, sections[1])
	}

	match, err := regexp.Match("[a-zA-Z0-9]*", []byte(sections[0]))

	if err != nil {
		t.Fatalf("TestEmailGenerator_NewRandom: err produced: %v", err)
	}

	if !match {
		t.Fatalf("TestEmailGenerator_NewRandom: email contains illegal chars. Email: %v", s)
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
