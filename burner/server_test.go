package burner

import (
	"html/template"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustParseTemplates(t *testing.T) {
	indexFile := template.Must(template.New("index").ParseFiles("../templates/base.html", "../templates/index.html"))
	indexPackr := mustParseTemplates(templates, "base.html", "index.html")

	out := indexOut{}

	fRecorder := httptest.NewRecorder()
	pRecorder := httptest.NewRecorder()

	if err := indexFile.ExecuteTemplate(fRecorder, "base", out); err != nil {
		t.Fatal(err)
	}

	if err := indexPackr.ExecuteTemplate(pRecorder, "base", out); err != nil {
		t.Fatal(err)
	}

	if fRecorder.Body.String() != pRecorder.Body.String() {
		t.Fatal("rendered html doesn't match")
	}
}
func TestIsBlackListed(t *testing.T) {
	s := Server{
		cfg: Config{
			BlacklistedDomains: []string{"example.com"},
		},
	}

	tests := []struct {
		Email    string
		Expected bool
	}{
		{
			Email:    "test@example.com",
			Expected: true,
		},
		{
			Email:    "test@example.org",
			Expected: false,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Expected, s.isBlacklistedDomain(test.Email))
	}
}
