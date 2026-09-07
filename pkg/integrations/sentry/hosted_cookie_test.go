package sentry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__SignAndParseSetupCookie(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	secret := "client-secret"

	value, err := SignSetupCookie(secret, "int-1", "nonce-1", now)
	require.NoError(t, err)

	integrationID, nonce, err := ParseSetupCookie(secret, value, now)
	require.NoError(t, err)
	assert.Equal(t, "int-1", integrationID)
	assert.Equal(t, "nonce-1", nonce)
}

func Test__ParseSetupCookie_rejectsTamperedOrExpired(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	value, err := SignSetupCookie("secret", "int-1", "nonce-1", now)
	require.NoError(t, err)

	_, _, err = ParseSetupCookie("other-secret", value, now)
	require.Error(t, err)

	_, _, err = ParseSetupCookie("secret", value+"x", now)
	require.Error(t, err)

	_, _, err = ParseSetupCookie("secret", value, now.Add(16*time.Minute))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}
