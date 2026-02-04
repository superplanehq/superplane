package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FloatTime decodes AWS timestamps that are represented as float64 seconds since epoch.
// It marshals to RFC3339 strings for consistency across outputs.
type FloatTime struct {
	time.Time
}

func (t FloatTime) IsZero() bool {
	return t.Time.IsZero()
}

func (t FloatTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.UTC().Format(time.RFC3339))
}

func (t *FloatTime) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		t.Time = time.Time{}
		return nil
	}

	if data[0] == '"' {
		parsed, err := t.parseString(strings.Trim(string(data), `"`))
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	}

	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("invalid float time value: %w", err)
	}

	t.Time = floatToTime(value)
	return nil
}

func (t FloatTime) parseString(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}

	if looksNumeric(value) {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid float time string: %w", err)
		}
		return floatToTime(parsed), nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid RFC3339 time string: %w", err)
	}

	return parsed.UTC(), nil
}

func floatToTime(value float64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(int64(value), 0).UTC()
}

func looksNumeric(value string) bool {
	for _, ch := range value {
		if (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' || ch == '+' || ch == 'e' || ch == 'E' {
			continue
		}
		return false
	}
	return true
}
