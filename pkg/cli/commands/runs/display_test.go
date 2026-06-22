package runs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestFormatRunDisplayValues(t *testing.T) {
	require.Equal(t, "-", formatRunCustomName(""))
	require.Equal(t, "My run", formatRunCustomName("My run"))
	require.Equal(t, "Started", formatRunState(openapi_client.CANVASESCANVASRUNSTATE_STATE_STARTED))
	require.Equal(t, "Finished", formatRunState(openapi_client.CANVASESCANVASRUNSTATE_STATE_FINISHED))
	require.Equal(t, "-", formatRunState(openapi_client.CANVASESCANVASRUNSTATE_STATE_UNKNOWN))
	require.Equal(t, "Passed", formatRunResult(openapi_client.CANVASESCANVASRUNRESULT_RESULT_PASSED))
	require.Equal(t, "Failed", formatRunResult(openapi_client.CANVASESCANVASRUNRESULT_RESULT_FAILED))
	require.Equal(t, "Cancelled", formatRunResult(openapi_client.CANVASESCANVASRUNRESULT_RESULT_CANCELLED))
	require.Equal(t, "-", formatRunResult(openapi_client.CANVASESCANVASRUNRESULT_RESULT_UNKNOWN))
	require.Equal(t, "Pending", formatExecutionState(openapi_client.CANVASESCANVASNODEEXECUTIONSTATE_STATE_PENDING))
	require.Equal(t, "Finished", formatExecutionState(openapi_client.CANVASESCANVASNODEEXECUTIONSTATE_STATE_FINISHED))
	require.Equal(t, "Passed", formatExecutionResult(openapi_client.CANVASESCANVASNODEEXECUTIONRESULT_RESULT_PASSED))
}

func TestFormatRelativeTimeAt(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)

	require.Equal(t, "-", formatRelativeTimeAt(time.Time{}, now))
	require.Equal(t, "1s ago", formatRelativeTimeAt(now.Add(-time.Second), now))
	require.Equal(t, "5m ago", formatRelativeTimeAt(now.Add(-5*time.Minute), now))
	require.Equal(t, "2h ago", formatRelativeTimeAt(now.Add(-2*time.Hour), now))
	require.Equal(t, "3d ago", formatRelativeTimeAt(now.Add(-72*time.Hour), now))
}

func TestFormatRunDurationAt(t *testing.T) {
	created := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)

	require.Equal(t, "-", formatRunDurationAt(time.Time{}, created, created))
	require.Equal(t, "5m", formatRunDurationAt(created, created.Add(5*time.Minute), created.Add(10*time.Minute)))
	require.Equal(t, "1s 500ms", formatRunDurationAt(created, created.Add(1500*time.Millisecond), created.Add(10*time.Minute)))
	require.Equal(t, "2m 5s", formatRunDurationAt(created, created.Add(125*time.Second), created.Add(10*time.Minute)))

	now := created.Add(30 * time.Second)
	require.Equal(t, "30s", formatRunDurationAt(created, time.Time{}, now))
}
