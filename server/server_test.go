package server

import (
	"html/template"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMustParseTemplates(t *testing.T) {
	indexFile := template.Must(template.New("index").ParseFiles("../templates/base.html", "../templates/index.html"))
	indexPackr := MustParseTemplates(templates.String("base.html"), templates.String("index.html"))

	out := indexOut{}

	fRecorder := httptest.NewRecorder()
	pRecorder := httptest.NewRecorder()

	if err := indexFile.ExecuteTemplate(fRecorder, "base", out); err != nil {
		t.Fatal(err)
	}

	if err := indexPackr.ExecuteTemplate(pRecorder, "base", out); err != nil {
		t.Fatal(err)
	}

	if strings.Compare(fRecorder.Body.String(), pRecorder.Body.String()) != 0 {
		t.Fatal("rendered html doesn't match")
	}
}
