package burner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
