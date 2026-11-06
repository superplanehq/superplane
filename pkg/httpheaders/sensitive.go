package httpheaders

import "strings"

// IsSensitive identifies credential-bearing headers that must not appear in
// diagnostic output. Match custom header names as well as standard headers.
func IsSensitive(name string) bool {
	name = strings.ReplaceAll(strings.ToLower(name), "_", "-")
	for _, part := range []string{"authorization", "cookie", "token", "secret", "password", "api-key", "apikey"} {
		if strings.Contains(name, part) {
			return true
		}
	}
	return false
}
