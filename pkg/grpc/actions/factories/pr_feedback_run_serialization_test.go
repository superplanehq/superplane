package factories

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestPRFeedbackRunStartedAt(t *testing.T) {
	created := time.Date(2026, 8, 26, 11, 0, 0, 0, time.UTC)
	runnerStart := created.Add(10 * time.Second)
	runnerUpdate := created.Add(5 * time.Minute)

	t.Run("uses the runner start, not the last runner update", func(t *testing.T) {
		started := prFeedbackRunStartedAt(
			models.CanvasRun{CreatedAt: &created},
			models.CanvasNodeExecution{CreatedAt: &runnerStart, UpdatedAt: &runnerUpdate},
		)
		require.NotNil(t, started)
		assert.True(t, started.AsTime().Equal(runnerStart))
	})

	t.Run("falls back to the canvas run created time", func(t *testing.T) {
		started := prFeedbackRunStartedAt(models.CanvasRun{CreatedAt: &created}, models.CanvasNodeExecution{})
		require.NotNil(t, started)
		assert.True(t, started.AsTime().Equal(created))
	})
}
