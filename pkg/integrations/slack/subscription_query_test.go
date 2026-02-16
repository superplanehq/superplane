package slack

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newDryRunPostgresDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=localhost user=postgres dbname=superplane sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	require.NoError(t, err)

	return db
}

func TestFindButtonClickSubscriptionQuery(t *testing.T) {
	t.Run("uses fixed predicates and parameterized values", func(t *testing.T) {
		db := newDryRunPostgresDB(t)
		installationID := uuid.New()
		messageTS := "1234567890.1234"
		channelID := "C12345"

		stmt := db.
			Where("installation_id = ?", installationID).
			Where("configuration->>'type' = ?", "button_click").
			Where("configuration->>'message_ts' = ?", messageTS).
			Where("configuration->>'channel_id' = ?", channelID).
			First(&models.IntegrationSubscription{}).
			Statement

		sql := stmt.SQL.String()
		assert.Contains(t, sql, "installation_id = $1")
		assert.Contains(t, sql, "configuration->>'type' = $2")
		assert.Contains(t, sql, "configuration->>'message_ts' = $3")
		assert.Contains(t, sql, "configuration->>'channel_id' = $4")

		// Ensure values are bound parameters and not interpolated directly into SQL
		assert.NotContains(t, sql, messageTS)
		assert.NotContains(t, sql, channelID)

		require.GreaterOrEqual(t, len(stmt.Vars), 4)
		assert.Equal(t, installationID, stmt.Vars[0])
		assert.Equal(t, "button_click", stmt.Vars[1])
		assert.Equal(t, messageTS, stmt.Vars[2])
		assert.Equal(t, channelID, stmt.Vars[3])
	})

	t.Run("keeps suspicious input as bound variables", func(t *testing.T) {
		db := newDryRunPostgresDB(t)
		installationID := uuid.New()
		messageTS := "1234' OR 1=1 --"
		channelID := "C123'; DROP TABLE app_installation_subscriptions; --"

		stmt := db.
			Where("installation_id = ?", installationID).
			Where("configuration->>'type' = ?", "button_click").
			Where("configuration->>'message_ts' = ?", messageTS).
			Where("configuration->>'channel_id' = ?", channelID).
			First(&models.IntegrationSubscription{}).
			Statement

		sql := stmt.SQL.String()
		assert.NotContains(t, sql, "DROP TABLE")
		assert.NotContains(t, sql, "OR 1=1")

		require.GreaterOrEqual(t, len(stmt.Vars), 4)
		assert.Equal(t, messageTS, stmt.Vars[2])
		assert.Equal(t, channelID, stmt.Vars[3])
	})
}
