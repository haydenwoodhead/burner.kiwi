package goalone

import (
	"encoding/base64"
	"time"
)

// Token is used to parse out a []byte token provided by Sign()
type Token struct {
	Payload   []byte
	Timestamp time.Time
}

// Parse will parse the []byte token returned from Sign based on the Sword
// Options into a Token struct. For this to work corectly the Sword Options need
// to match that of what was used when the token was initially created.
func (s *Sword) Parse(t []byte) Token {

	tl := len(t)
	el := base64.RawURLEncoding.EncodedLen(s.hash.Size())

	token := Token{}

	if s.timestamp {
		// we need to find out how many bytes the timestamp is.
		// so lets start at the start of the hash, and work back looking for our
		// separator - XXX: I'm sure there's room for improvement here.
		for i := tl - (el + 2); i >= 0; i-- {
			if t[i] == '.' {
				token.Payload = t[0:i]
				token.Timestamp = time.Unix(decodeBase58(t[i+1:tl-(el+1)])+s.epoch, 0)
				break
			}
		}
	} else {
		token.Payload = t[0 : tl-(el+1)]
	}

	return token
}
