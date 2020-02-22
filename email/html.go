package email

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

//AddTargetBlank finds all a tags and add a target="_blank" attr to them so they
// open links in a new tab rather than in the iframe
func AddTargetBlank(html string) (string, error) {
	sr := strings.NewReader(html)

	var doc *goquery.Document
	doc, err := goquery.NewDocumentFromReader(sr)
	if err != nil {
		return "", fmt.Errorf("AddTargetBlank: failed to create goquery doc: %v", err)
	}

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("target", "_blank")
	})

	var modifiedHTML string
	modifiedHTML, err = doc.Html()
	if err != nil {
		return "", fmt.Errorf("AddTargetBlank: failed to get html doc: %v", err)
	}

	return modifiedHTML, nil
}
