package alerts

import (
	"context"
	"fmt"
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
		res, err := ListAlerts(context.Background(), uuid.NewString(), false, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Empty(t, res.Alerts)
	})

	t.Run("return list of alerts in the canvas", func(t *testing.T) {
		alert, err := models.NewAlert(r.Canvas.ID, r.Stage.ID, "stage", "Test alert message", models.AlertTypeError, models.AlertOriginTypeEventRejection)
		require.NoError(t, err)
		require.NoError(t, alert.Create())

		res, err := ListAlerts(context.Background(), r.Canvas.ID.String(), true, nil, nil)
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
		alert, err := models.NewAlert(r.Canvas.ID, r.Stage.ID, "stage", "Test acknowledged alert", models.AlertTypeWarning, models.AlertOriginTypeEventRejection)
		require.NoError(t, err)
		require.NoError(t, alert.Create())
		alert.Acknowledge()
		require.NoError(t, alert.Update())

		res, err := ListAlerts(context.Background(), r.Canvas.ID.String(), false, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)

		for _, a := range res.Alerts {
			assert.False(t, a.Acknowledged, "Expected only unacknowledged alerts")
		}
	})

	t.Run("limit number of returned alerts", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			alert, err := models.NewAlert(r.Canvas.ID, r.Stage.ID, "stage", fmt.Sprintf("Test alert %d", i), models.AlertTypeInfo, models.AlertOriginTypeEventRejection)
			require.NoError(t, err)
			require.NoError(t, alert.Create())
		}

		limit := uint32(2)
		res, err := ListAlerts(context.Background(), r.Canvas.ID.String(), true, nil, &limit)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.LessOrEqual(t, len(res.Alerts), 2, "Expected at most 2 alerts")

		limit = uint32(0)
		res, err = ListAlerts(context.Background(), r.Canvas.ID.String(), true, nil, &limit)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.GreaterOrEqual(t, len(res.Alerts), 3, "Expected at least 3 alerts when limit is 0")
	})

	t.Run("validate limit constraints", func(t *testing.T) {
		for i := 0; i < 150; i++ {
			alert, err := models.NewAlert(r.Canvas.ID, r.Stage.ID, "stage", fmt.Sprintf("Test alert %d", i), models.AlertTypeInfo, models.AlertOriginTypeEventRejection)
			require.NoError(t, err)
			require.NoError(t, alert.Create())
		}

		limit := uint32(150)
		res, err := ListAlerts(context.Background(), r.Canvas.ID.String(), true, nil, &limit)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.LessOrEqual(t, len(res.Alerts), 100, "Expected at most MaxLimit (100) alerts")

		res, err = ListAlerts(context.Background(), r.Canvas.ID.String(), true, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.LessOrEqual(t, len(res.Alerts), 50, "Expected at most DefaultLimit (50) alerts when limit is nil")
	})
}
