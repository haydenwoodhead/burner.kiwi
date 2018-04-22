package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

var url string
var key string

func callDelete() error {
	c := http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		return err
	}

	req.Header.Add("X-Burner-Delete-Key", key)

	resp, err := c.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("returned error code not 200. Actually %v", resp.StatusCode)
	}

	return nil
}

func main() {
	url = mustParseStringVar("URL")
	key = mustParseStringVar("KEY")
	lambda.Start(callDelete)
}

func mustParseStringVar(key string) (v string) {
	v = os.Getenv(key)

	if strings.Compare(v, "") == 0 {
		log.Fatalf("Env var %v cannot be empty", key)
	}

	return
}
