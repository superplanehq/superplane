package digitalocean

import (
	"fmt"
	"strconv"
	"strings"
)

func resolveIntID(config any, fieldName string) (int, error) {
	m, ok := config.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("invalid configuration type")
	}

	raw, ok := m[fieldName]
	if !ok {
		return 0, fmt.Errorf("%s is required", fieldName)
	}

	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, fmt.Errorf("%s is required", fieldName)
		}
		id, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("invalid %s value: %v", fieldName, v)
		}
		return id, nil
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
		return 0, fmt.Errorf("invalid %s value: %v", fieldName, raw)
	}
}

func resolveStringField(config any, fieldName string) (string, error) {
	m, ok := config.(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid configuration type")
	}

	raw, ok := m[fieldName]
	if !ok {
		return "", fmt.Errorf("%s is required", fieldName)
	}

	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return "", fmt.Errorf("%s is required", fieldName)
		}
		return s, nil
	case float64:
		return fmt.Sprintf("%.0f", v), nil
	case int:
		return fmt.Sprintf("%d", v), nil
	default:
		return "", fmt.Errorf("invalid %s value: %v", fieldName, raw)
	}
}

func readStringFromAny(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%.0f", x)
	case int:
		return fmt.Sprintf("%d", x)
	default:
		return fmt.Sprintf("%v", v)
	}
}
