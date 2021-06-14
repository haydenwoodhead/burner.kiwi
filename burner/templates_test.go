package burner

import (
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
