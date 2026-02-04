package common

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFloatTimeUnmarshalFloat(t *testing.T) {
	base := time.Now().UTC().Add(-3 * time.Second).Truncate(time.Nanosecond)
	value := float64(base.Unix()) + float64(base.Nanosecond())/float64(time.Second)
	payload := fmt.Sprintf(`%0.9f`, value)

	var parsed FloatTime
	err := json.Unmarshal([]byte(payload), &parsed)
	require.NoError(t, err)
	require.True(t, parsed.Equal(base))
}

func TestFloatTimeUnmarshalStringRFC3339(t *testing.T) {
	base := time.Now().UTC().Add(-5 * time.Second).Truncate(time.Second)
	payload := fmt.Sprintf(`"%s"`, base.Format(time.RFC3339))

	var parsed FloatTime
	err := json.Unmarshal([]byte(payload), &parsed)
	require.NoError(t, err)
	require.True(t, parsed.Equal(base))
}

func TestFloatTimeUnmarshalStringNumeric(t *testing.T) {
	base := time.Now().UTC().Add(-7 * time.Second).Truncate(time.Nanosecond)
	value := float64(base.Unix()) + float64(base.Nanosecond())/float64(time.Second)
	payload := fmt.Sprintf(`"%0.9f"`, value)

	var parsed FloatTime
	err := json.Unmarshal([]byte(payload), &parsed)
	require.NoError(t, err)
	require.True(t, parsed.Equal(base))
}

func TestFloatTimeMarshal(t *testing.T) {
	base := time.Now().UTC().Add(-9 * time.Second).Truncate(time.Nanosecond)
	value := FloatTime{Time: base}

	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf(`"%s"`, base.Format(time.RFC3339)), string(encoded))
}

func TestFloatTimeMarshalZero(t *testing.T) {
	value := FloatTime{}

	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	require.Equal(t, "null", string(encoded))
}
