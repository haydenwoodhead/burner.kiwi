package token

import (
	"strings"
	"testing"
	"time"
)

const PARSE_RETURNED = "tk"

func TestTokenGenerator_Parse(t *testing.T) {
	tests := []struct {
		ID          string
		Time        time.Duration
		ToParse     string
		ExpectedRes string
		ExpectedErr error
	}{
		{
			ID:          "dafd5606-8aa8-4724-a2c5-f66110aba536",
			Time:        1 * time.Hour,
			ToParse:     PARSE_RETURNED,
			ExpectedRes: "dafd5606-8aa8-4724-a2c5-f66110aba536",
			ExpectedErr: nil,
		},
		{
			ID:          "f0870b33-03de-4223-8418-f01f2fcacf04",
			Time:        1 * time.Second,
			ToParse:     PARSE_RETURNED,
			ExpectedRes: "",
			ExpectedErr: ErrTokenExpired,
		},
		{
			ID:          "dafd5606-8aa8-4724-a2c5-f66110aba536",
			Time:        time.Hour,
			ToParse:     "invalid-for-signature.1523080494.dxeP8ibFqKuCDDb28ourLgd88rJfw14JQt8vX0yL0dk",
			ExpectedRes: "",
			ExpectedErr: ErrInvalidSig,
		},
		{
			ID:          "dafd5606-8aa8-4724-a2c5-f66110aba536",
			Time:        time.Hour,
			ToParse:     "dafd5606-8aa8-4724-a2c5-f66110aba536.invalid-for-signature.dxeP8ibFqKuCDDb28ourLgd88rJfw14JQt8vX0yL0dk",
			ExpectedRes: "",
			ExpectedErr: ErrInvalidSig,
		},
	}

	for i, test := range tests {
		tg := NewGenerator("test1234", test.Time)

		tk := tg.NewToken(test.ID)

		time.Sleep(2 * time.Second)

		var p string
		var err error

		if strings.Compare(test.ToParse, PARSE_RETURNED) == 0 {
			p, err = tg.VerifyToken(tk)
		} else {
			p, err = tg.VerifyToken(test.ToParse)
		}

		if strings.Compare(p, test.ExpectedRes) != 0 {
			t.Errorf("%v - Expected result %v, got %v", i, test.ExpectedRes, p)
		}

		if err != test.ExpectedErr {
			t.Errorf("%v - Expected error %v, got %v", i, test.ExpectedErr, err)
		}
	}
}
