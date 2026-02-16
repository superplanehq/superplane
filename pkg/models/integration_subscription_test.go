package models

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

func TestFindIntegrationSubscriptionByConfigFields(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("should return error for invalid field name", func(t *testing.T) {
		installationID := uuid.New()
		// Try to use an invalid/unsafe field name
		_, err := FindIntegrationSubscriptionByConfigFields(database.Conn(), installationID, map[string]string{
			"invalid_field": "some_value",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported configuration field")
	})

	t.Run("should reject SQL injection attempts", func(t *testing.T) {
		installationID := uuid.New()
		// Try to inject SQL via field name
		_, err := FindIntegrationSubscriptionByConfigFields(database.Conn(), installationID, map[string]string{
			"'; DROP TABLE app_installation_subscriptions; --": "malicious",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported configuration field")

		// Verify table still exists by querying it
		var count int64
		err = database.Conn().Model(&IntegrationSubscription{}).Count(&count).Error
		require.NoError(t, err, "Table should still exist after injection attempt")
	})

	t.Run("should reject attempts to access arbitrary columns", func(t *testing.T) {
		installationID := uuid.New()
		// Try to access a different column
		_, err := FindIntegrationSubscriptionByConfigFields(database.Conn(), installationID, map[string]string{
			"installation_id": installationID.String(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported configuration field")
	})

	t.Run("should only allow allowlisted field names", func(t *testing.T) {
		installationID := uuid.New()

		// Test that each allowed field name works (even if no record is found)
		allowedFields := []string{"message_ts", "channel_id", "type"}
		for _, field := range allowedFields {
			_, err := FindIntegrationSubscriptionByConfigFields(database.Conn(), installationID, map[string]string{
				field: "test_value",
			})
			// Should return ErrRecordNotFound, not unsupported field error
			assert.True(t, errors.Is(err, gorm.ErrRecordNotFound),
				"Field %s should be allowed but got error: %v", field, err)
		}
	})
}
