package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddTargetBlank(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{
			in:  `<html><body><a href="https://example.com">Hello there</a></body></html>`,
			out: `<html><head></head><body><a href="https://example.com" target="_blank">Hello there</a></body></html>`,
		},
	}

	for _, test := range tests {
		out, err := AddTargetBlank(test.in)
		require.NoError(t, err)
		assert.Equal(t, test.out, out)
	}
}
