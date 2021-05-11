package burner

import (
	"testing"
)

func TestNewInbox(t *testing.T) {
	i := NewInbox()

	if i.FailedToCreate {
		t.Errorf("TestNewInbox: failed to create not true")
	}

	if i.EmailProviderRouteID != "-" {
		t.Errorf("TestNewInbox: route id not -")
	}
}
