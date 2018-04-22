package server

import (
	"fmt"
	"strings"
)

// GetHoursAndMinutes extracts only the hours and minutes from a duration as strings
func GetHoursAndMinutes(d fmt.Stringer) (string, string) {
	var gotHour bool

	var hour []byte
	var min []byte

	bytes := []byte(d.String())

	// loop over bytes in duration string
	for _, b := range bytes {
		if !gotHour {
			if b == []byte("h")[0] { // loop until we hit "h" signalling the end of the number of hours
				gotHour = true
				continue
			} else if b == []byte("m")[0] { // however if we hit "m" before we get the number of hours we know the number of hours is 0
				min = hour
				hour = nil
				break
			}

			hour = append(hour, b)
		} else {
			if b == []byte("m")[0] {
				break
			}

			min = append(min, b)
		}
	}

	h := string(hour)
	m := string(min)

	if strings.Compare(h, "") == 0 {
		h = "0"
	}

	return h, m
}
