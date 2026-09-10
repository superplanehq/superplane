package openrouter

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestChildKeyExpiresAtUsesTimeoutPlusSlack(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	got := ChildKeyExpiresAt(now, 3600)
	assert.Equal(t, now.Add(time.Hour).Add(ChildKeyTTLSlack), got)
	assert.Equal(t, got.Format("2006-01-02T15:04:05Z"), FormatKeyExpiresAt(got))
}

func TestRunnerKeyNameIncludesRunID(t *testing.T) {
	runID := uuid.New()
	assert.Equal(t, "superplane-run-"+runID.String(), RunnerKeyName(runID))
}
