package notary

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotary(t *testing.T) {
	c := Notary{
		SigningKey: "secret",
		Clock: func() time.Time {
			return time.Date(2021, 04, 20, 14, 41, 0, 0, time.UTC)
		},
	}

	type Payload struct {
		Foo string
		Bar string
	}

	t.Run("standard use and parse", func(t *testing.T) {
		in := Payload{
			Foo: "foo",
			Bar: "bar",
		}

		jwt, err := c.Sign("token", in, c.Clock().Add(1*time.Minute).Unix())
		require.NoError(t, err)

		var out Payload
		err = c.Verify(jwt, &out)
		require.NoError(t, err)

		assert.Equal(t, in, out)
	})

	t.Run("expired should error", func(t *testing.T) {
		expired := "eyJhbGciOiJIUzI1NiJ9.eyJCYXIiOiJiYXIiLCJGb28iOiJmb28iLCJfX3B1cnBvc2UiOiJ0b2tlbiIsImV4cCI6MTYxODkyOTA2MCwiaWF0IjoxNjE4OTI5NjYwLCJpc3MiOiJidXJuZXIua2l3aSJ9.rkBzTZxIMmFhzWyy93ClkCY0hRFCapQV_cRmAx5j-00"

		var out Payload
		err := c.Verify(expired, &out)
		assert.Equal(t, ErrExpired, err)
	})

	t.Run("modified JWT should error", func(t *testing.T) {
		var out Payload
		err := c.Verify("eyJhbGciOiJIUzI1NiJ9.eyJCYXIiOiJiZWUiLCJGb28iOiJmb28iLCJfX3B1cnBvc2UiOiJ0b2tlbiIsImV4cCI6MTYxODkyOTA2MCwiaWF0IjoxNjE4OTI5NjYwLCJpc3MiOiJidXJuZXIua2l3aSJ9.9XGiAB5fUEvlFk4B5Ncl2NFpDXlrIQmidMj-IbgA-hM", &out)
		assert.Equal(t, ErrInvalid, err)
	})
}
