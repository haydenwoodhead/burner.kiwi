package notary

import (
	"errors"
	"fmt"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

var ErrExpired = errors.New("expired")
var ErrInvalid = errors.New("invalid")

type Notary struct {
	SigningKey string
	Clock      func() time.Time
}

func New(signingKey string) *Notary {
	return &Notary{SigningKey: signingKey, Clock: func() time.Time {
		return time.Now()
	}}
}

type purposeInfo struct {
	Purpose string `json:"__purpose"`
}

var Iss = "burner.kiwi"

// Sign produces a JWT with the given payload and name encoded. Valid until ttl
func (c *Notary) Sign(purpose string, payload interface{}, ttl int64) (string, error) {
	jwtClaims := &jwt.Claims{
		Issuer:   Iss,
		Expiry:   jwt.NewNumericDate(time.Unix(ttl, 0)),
		IssuedAt: jwt.NewNumericDate(c.Clock()),
	}

	beaconClaims := purposeInfo{Purpose: purpose}

	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.HS256,
		Key:       []byte(c.SigningKey),
	}, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create signer: %w", err)
	}

	return jwt.Signed(signer).Claims(jwtClaims).Claims(beaconClaims).Claims(payload).CompactSerialize()
}

func (c *Notary) Verify(signed string, out interface{}) error {
	tok, err := jwt.ParseSigned(signed)
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	var jwtClaims jwt.Claims
	if err := tok.Claims([]byte(c.SigningKey), &jwtClaims, &out); err != nil {
		if err == jose.ErrCryptoFailure {
			return ErrInvalid
		}
		return fmt.Errorf("failed to get token claims: %w", err)
	}

	err = jwtClaims.ValidateWithLeeway(jwt.Expected{
		Issuer: Iss,
		Time:   c.Clock(),
	}, 0)
	if err != nil {
		if err == jwt.ErrExpired {
			return ErrExpired
		}
		return fmt.Errorf("failed to validate token claims: %w", err)
	}

	return nil
}
