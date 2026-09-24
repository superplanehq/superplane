package openrouter

import (
	"time"

	"github.com/google/uuid"
)

// ChildKeyTTLSlack covers broker queue delay after SuperPlane mints a key.
const ChildKeyTTLSlack = 15 * time.Minute

func ChildKeyExpiresAt(now time.Time, executionTimeoutSeconds int) time.Time {
	timeout := time.Duration(executionTimeoutSeconds) * time.Second
	return now.UTC().Truncate(time.Second).Add(timeout).Add(ChildKeyTTLSlack)
}

func FormatKeyExpiresAt(expiresAt time.Time) string {
	return expiresAt.UTC().Format("2006-01-02T15:04:05Z")
}

func RunnerKeyName(runID uuid.UUID) string {
	return "superplane-run-" + runID.String()
}
