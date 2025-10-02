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

func Test__ListAlerts(t *testing.T) {
	r := support.Setup(t)

	t.Run("return empty list of alerts", func(t *testing.T) {
		res, err := ListAlerts(context.Background(), uuid.NewString(), false, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Empty(t, res.Alerts)
	})

	t.Run("return list of alerts in the canvas", func(t *testing.T) {
		alert, err := models.NewAlert(r.Canvas.ID, r.Stage.ID, "stage", "Test alert message", models.AlertTypeError)
		require.NoError(t, err)
		require.NoError(t, alert.Create())

		res, err := ListAlerts(context.Background(), r.Canvas.ID.String(), true, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Alerts, 1)
		assert.Equal(t, alert.ID.String(), res.Alerts[0].Id)
		assert.Equal(t, alert.Message, res.Alerts[0].Message)
		assert.Equal(t, alert.SourceID.String(), res.Alerts[0].SourceId)
		assert.False(t, res.Alerts[0].Acknowledged)
		assert.NotEmpty(t, res.Alerts[0].CreatedAt)
	})

	t.Run("filter out acknowledged alerts when includeAcked is false", func(t *testing.T) {
		alert, err := models.NewAlert(r.Canvas.ID, r.Stage.ID, "stage", "Test acknowledged alert", models.AlertTypeWarning)
		require.NoError(t, err)
		require.NoError(t, alert.Create())
		alert.Acknowledge()
		require.NoError(t, alert.Update())

		res, err := ListAlerts(context.Background(), r.Canvas.ID.String(), false, nil)
		require.NoError(t, err)
		require.NotNil(t, res)

		for _, a := range res.Alerts {
			assert.False(t, a.Acknowledged, "Expected only unacknowledged alerts")
		}
	})
}