package scim

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/utils"
)

func stringFromAttributes(attrs map[string]interface{}, key string) (string, bool) {
	raw, ok := attrs[key]
	if !ok || raw == nil {
		return "", false
	}
	s, ok := raw.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

func primaryEmail(attrs map[string]interface{}) (string, error) {
	raw, ok := attrs["emails"]
	if !ok || raw == nil {
		return "", fmt.Errorf("emails required")
	}
	list, ok := raw.([]interface{})
	if !ok || len(list) == 0 {
		return "", fmt.Errorf("emails must be a non-empty array")
	}
	var fallback string
	for _, e := range list {
		m, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		v, ok := m["value"].(string)
		if !ok || strings.TrimSpace(v) == "" {
			continue
		}
		if primary, ok := m["primary"].(bool); ok && primary {
			return utils.NormalizeEmail(v), nil
		}
		if fallback == "" {
			fallback = utils.NormalizeEmail(v)
		}
	}
	if fallback == "" {
		return "", fmt.Errorf("no usable email in emails array")
	}
	return fallback, nil
}

func displayName(attrs map[string]interface{}, userName string) string {
	if s, ok := stringFromAttributes(attrs, "displayName"); ok {
		return s
	}
	raw, ok := attrs["name"]
	if !ok || raw == nil {
		return userName
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return userName
	}
	if s, ok := m["formatted"].(string); ok && strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	var parts []string
	if s, ok := m["givenName"].(string); ok && strings.TrimSpace(s) != "" {
		parts = append(parts, strings.TrimSpace(s))
	}
	if s, ok := m["familyName"].(string); ok && strings.TrimSpace(s) != "" {
		parts = append(parts, strings.TrimSpace(s))
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return userName
}

func activeBool(attrs map[string]interface{}, defaultActive bool) bool {
	raw, ok := attrs["active"]
	if !ok || raw == nil {
		return defaultActive
	}
	b, ok := raw.(bool)
	if !ok {
		return defaultActive
	}
	return b
}
