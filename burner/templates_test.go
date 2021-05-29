package burner

import (
	"html/template"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tj/assert"
)

func TestCalculateReceivedAt(t *testing.T) {
	tests := []struct {
		In       int64
		Expected string
	}{
		{
			In:       time.Now().Unix(),
			Expected: "Less than 30s ago",
		},
		{
			In:       time.Now().Add(-30 * time.Minute).Unix(),
			Expected: "30m ago",
		},
		{
			In:       time.Now().Add(-10 * time.Second).Add(-30 * time.Minute).Add(-2 * time.Hour).Unix(),
			Expected: "2h 30m ago",
		},
	}

	for _, test := range tests {
		out := calculateReceivedAt(test.In)
		assert.Equal(t, test.Expected, out)
	}
}

func TestGetAvatarDetails(t *testing.T) {
	tests := []struct {
		In             string
		ExpectedLetter string
		ExpectedColor  string
	}{
		{
			In:             "Hayden Woodhead <hayden@example.com>",
			ExpectedLetter: "H",
			ExpectedColor:  "bg-yellow",
		},
		{
			In:             "Adam Smith <adam@example.com>",
			ExpectedLetter: "A",
			ExpectedColor:  "bg-red",
		},
		{ // TODO: once unicode is normalised this should be the same as "N"
			In:             "Ñandú <rhea@example.com>",
			ExpectedLetter: "Ñ",
			ExpectedColor:  "bg-pink",
		},
	}

	for _, test := range tests {
		outLetter, outColor := getAvatarDetails(test.In)
		assert.Equal(t, test.ExpectedLetter, outLetter)
		assert.Equal(t, test.ExpectedColor, outColor)
	}
}

func TestMustParseTemplates(t *testing.T) {
	indexFile := template.Must(template.New("index").ParseFiles("../templates/base.html", "../templates/inbox.html"))
	indexPackr := mustParseTemplates(templates, "base.html", "inbox.html")

	out := inboxOut{}

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
