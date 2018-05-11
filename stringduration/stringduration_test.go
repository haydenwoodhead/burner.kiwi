package stringduration

import (
	"testing"
	"time"
)

func TestGetHoursAndMinutes(t *testing.T) {
	tests := []struct {
		In      string
		Hours   string
		Minutes string
	}{
		{
			In:      "1h30m0s",
			Hours:   "1",
			Minutes: "30",
		},
		{
			In:      "12h30m0s",
			Hours:   "12",
			Minutes: "30",
		},
		{
			In:      "30m0s",
			Hours:   "0",
			Minutes: "30",
		},
		{
			In:      "3m0s",
			Hours:   "0",
			Minutes: "3",
		},
		{
			In:      "3000h9m0s",
			Hours:   "3000",
			Minutes: "9",
		},
	}

	for i, test := range tests {

		d, err := time.ParseDuration(test.In)

		if err != nil {
			t.Errorf("%v - TestGetHoursAndMinutes failed. Failed to parse duration: %v", i, err)
		}

		h, m := GetHoursAndMinutes(d)

		if h != test.Hours {
			t.Errorf("%v - TestGetHoursAndMinutes failed. Expected %v hours, got %v", i, test.Hours, h)
		}

		if m != test.Minutes {
			t.Errorf("%v - TestGetHoursAndMinutes failed. Expected %v minutes, got %v", i, test.Minutes, m)
		}
	}
}
