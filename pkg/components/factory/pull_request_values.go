package factory

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseOptionalRFC3339(raw string) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, fmt.Errorf("timestamp must be RFC3339 (got %q)", trimmed)
	}
	return &parsed, nil
}

func parseOptionalInt64(raw string) (int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("number must be an integer (got %q)", trimmed)
	}
	return value, nil
}

func optionalStringPointer(raw string, present bool) *string {
	if !present {
		return nil
	}
	trimmed := strings.TrimSpace(raw)
	return &trimmed
}
