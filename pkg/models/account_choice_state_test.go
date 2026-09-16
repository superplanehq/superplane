package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestAccountChoiceState(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	now := time.Now()

	newState := func(token, providerID string, expiresAt time.Time) *models.AccountChoiceState {
		return &models.AccountChoiceState{
			TokenHash:   crypto.HashToken(token),
			Provider:    models.ProviderGitHub,
			ProviderID:  providerID,
			Redirect:    "/canvases",
			Email:       "choice@example.com",
			Name:        "Choice User",
			AccessToken: []byte("sealed-access"),
			ExpiresAt:   expiresAt,
		}
	}

	t.Run("finds a valid unused state", func(t *testing.T) {
		state := newState("valid-token", "github-1", now.Add(10*time.Minute))
		require.NoError(t, models.CreateAccountChoiceState(db, state))

		found, err := models.FindValidAccountChoiceState(db, crypto.HashToken("valid-token"), now)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "github-1", found.ProviderID)
		assert.Equal(t, "/canvases", found.Redirect)
		assert.Equal(t, []byte("sealed-access"), found.AccessToken)
	})

	t.Run("does not find an expired state", func(t *testing.T) {
		state := newState("expired-token", "github-2", now.Add(-time.Minute))
		require.NoError(t, models.CreateAccountChoiceState(db, state))

		found, err := models.FindValidAccountChoiceState(db, crypto.HashToken("expired-token"), now)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, found)
	})

	t.Run("claims a state one time", func(t *testing.T) {
		state := newState("once-token", "github-3", now.Add(10*time.Minute))
		require.NoError(t, models.CreateAccountChoiceState(db, state))

		claimed, err := models.ClaimAccountChoiceState(db, crypto.HashToken("once-token"), now)
		require.NoError(t, err)
		require.NotNil(t, claimed)
		assert.Equal(t, "github-3", claimed.ProviderID)
		require.NotNil(t, claimed.UsedAt)

		again, err := models.ClaimAccountChoiceState(db, crypto.HashToken("once-token"), now)
		require.NoError(t, err)
		assert.Nil(t, again)

		found, err := models.FindValidAccountChoiceState(db, crypto.HashToken("once-token"), now)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, found)
	})

	t.Run("does not claim an expired state", func(t *testing.T) {
		state := newState("expired-claim", "github-4", now.Add(-time.Minute))
		require.NoError(t, models.CreateAccountChoiceState(db, state))

		claimed, err := models.ClaimAccountChoiceState(db, crypto.HashToken("expired-claim"), now)
		require.NoError(t, err)
		assert.Nil(t, claimed)
	})
}
