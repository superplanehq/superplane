package common

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFloatTime(t *testing.T) {
	t.Run("unmarshal numeric JSON seconds since epoch", func(t *testing.T) {
		base := time.Now().UTC().Add(-3 * time.Second).Truncate(time.Second)
		payload := fmt.Sprintf(`%d`, base.Unix())

		var parsed FloatTime
		err := json.Unmarshal([]byte(payload), &parsed)
		require.NoError(t, err)
		require.True(t, parsed.Equal(base))
	})

	t.Run("unmarshal quoted RFC3339 string", func(t *testing.T) {
		base := time.Now().UTC().Add(-5 * time.Second).Truncate(time.Second)
		payload := fmt.Sprintf(`"%s"`, base.Format(time.RFC3339))

		var parsed FloatTime
		err := json.Unmarshal([]byte(payload), &parsed)
		require.NoError(t, err)
		require.True(t, parsed.Equal(base))
	})

	t.Run("unmarshal quoted numeric seconds since epoch", func(t *testing.T) {
		base := time.Now().UTC().Add(-7 * time.Second).Truncate(time.Second)
		payload := fmt.Sprintf(`"%d"`, base.Unix())

		var parsed FloatTime
		err := json.Unmarshal([]byte(payload), &parsed)
		require.NoError(t, err)
		require.True(t, parsed.Equal(base))
	})

	t.Run("marshal to RFC3339 string", func(t *testing.T) {
		base := time.Now().UTC().Add(-9 * time.Second).Truncate(time.Second)
		value := FloatTime{Time: base}

		encoded, err := json.Marshal(value)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf(`"%s"`, base.Format(time.RFC3339)), string(encoded))
	})

	t.Run("marshal zero time as null", func(t *testing.T) {
		value := FloatTime{}

		encoded, err := json.Marshal(value)
		require.NoError(t, err)
		require.Equal(t, "null", string(encoded))
	})
}
