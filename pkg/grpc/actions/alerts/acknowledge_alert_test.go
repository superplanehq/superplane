package alerts

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__AcknowledgeAlert(t *testing.T) {
	r := support.Setup(t)

	t.Run("successfully acknowledge an alert", func(t *testing.T) {
		alert, err := models.NewAlert(r.Canvas.ID, r.Stage.ID, "stage", "Test alert message", models.AlertTypeError, models.AlertOriginTypeEventRejection)
		require.NoError(t, err)
		require.NoError(t, alert.Create())

		res, err := AcknowledgeAlert(context.Background(), r.Canvas.ID.String(), alert.ID.String())
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Alert)
		assert.Equal(t, alert.ID.String(), res.Alert.Id)
		assert.True(t, res.Alert.Acknowledged)
		assert.NotEmpty(t, res.Alert.AcknowledgedAt)
	})

	t.Run("return error for invalid canvas ID", func(t *testing.T) {
		alert, err := models.NewAlert(r.Canvas.ID, r.Stage.ID, "stage", "Test alert message", models.AlertTypeError, models.AlertOriginTypeEventRejection)
		require.NoError(t, err)
		require.NoError(t, alert.Create())

		res, err := AcknowledgeAlert(context.Background(), "invalid-uuid", alert.ID.String())
		require.Error(t, err)
		require.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid canvas ID")
	})

	t.Run("return error for invalid alert ID", func(t *testing.T) {
		res, err := AcknowledgeAlert(context.Background(), r.Canvas.ID.String(), "invalid-uuid")
		require.Error(t, err)
		require.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid alert ID")
	})

	t.Run("return error for non-existent alert", func(t *testing.T) {
		nonExistentAlertID := uuid.New()
		res, err := AcknowledgeAlert(context.Background(), r.Canvas.ID.String(), nonExistentAlertID.String())
		require.Error(t, err)
		require.Nil(t, res)
		assert.Contains(t, err.Error(), "failed to find alert")
	})
}
