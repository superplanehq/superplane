package openrouter

import (
	"time"
)

// ChildKeyTTLSlack covers delivery delay after credential preparation.
const ChildKeyTTLSlack = 15 * time.Minute

func FormatKeyExpiresAt(expiresAt time.Time) string {
	return expiresAt.UTC().Format("2006-01-02T15:04:05Z")
}
