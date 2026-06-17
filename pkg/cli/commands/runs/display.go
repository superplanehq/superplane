package runs

import (
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func formatRunCustomName(customName string) string {
	if customName == "" {
		return "-"
	}

	return customName
}

func formatRunState(state openapi_client.CanvasesCanvasRunState) string {
	switch state {
	case openapi_client.CANVASESCANVASRUNSTATE_STATE_STARTED:
		return "Started"
	case openapi_client.CANVASESCANVASRUNSTATE_STATE_FINISHED:
		return "Finished"
	default:
		return "-"
	}
}

func formatRunResult(result openapi_client.CanvasesCanvasRunResult) string {
	switch result {
	case openapi_client.CANVASESCANVASRUNRESULT_RESULT_PASSED:
		return "Passed"
	case openapi_client.CANVASESCANVASRUNRESULT_RESULT_FAILED:
		return "Failed"
	case openapi_client.CANVASESCANVASRUNRESULT_RESULT_CANCELLED:
		return "Cancelled"
	default:
		return "-"
	}
}

func formatExecutionState(state openapi_client.CanvasesCanvasNodeExecutionState) string {
	switch state {
	case openapi_client.CANVASESCANVASNODEEXECUTIONSTATE_STATE_PENDING:
		return "Pending"
	case openapi_client.CANVASESCANVASNODEEXECUTIONSTATE_STATE_STARTED:
		return "Started"
	case openapi_client.CANVASESCANVASNODEEXECUTIONSTATE_STATE_FINISHED:
		return "Finished"
	default:
		return "-"
	}
}

func formatExecutionResult(result openapi_client.CanvasesCanvasNodeExecutionResult) string {
	switch result {
	case openapi_client.CANVASESCANVASNODEEXECUTIONRESULT_RESULT_PASSED:
		return "Passed"
	case openapi_client.CANVASESCANVASNODEEXECUTIONRESULT_RESULT_FAILED:
		return "Failed"
	case openapi_client.CANVASESCANVASNODEEXECUTIONRESULT_RESULT_CANCELLED:
		return "Cancelled"
	default:
		return "-"
	}
}

func formatRelativeTime(value time.Time) string {
	return formatRelativeTimeAt(value, time.Now())
}

func formatRelativeTimeAt(value time.Time, now time.Time) string {
	if value.IsZero() {
		return "-"
	}

	elapsed := now.Sub(value)
	if elapsed < 0 {
		elapsed = 0
	}

	switch {
	case elapsed < time.Minute:
		seconds := int(elapsed.Seconds())
		if seconds <= 1 {
			return "1s ago"
		}
		return fmt.Sprintf("%ds ago", seconds)
	case elapsed < time.Hour:
		minutes := int(elapsed.Minutes())
		if minutes <= 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", minutes)
	case elapsed < 24*time.Hour:
		hours := int(elapsed.Hours())
		if hours <= 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	default:
		days := int(elapsed.Hours() / 24)
		if days <= 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

func formatRunDuration(created, finished time.Time) string {
	return formatRunDurationAt(created, finished, time.Now())
}

func formatRunDurationAt(created, finished, now time.Time) string {
	if created.IsZero() {
		return "-"
	}

	end := finished
	if end.IsZero() {
		end = now
	}

	duration := end.Sub(created)
	if duration < 0 {
		return "-"
	}

	if duration == 0 {
		return "0ms"
	}

	days := int(duration.Hours() / 24)
	duration -= time.Duration(days) * 24 * time.Hour

	hours := int(duration.Hours())
	duration -= time.Duration(hours) * time.Hour

	minutes := int(duration.Minutes())
	duration -= time.Duration(minutes) * time.Minute

	seconds := int(duration.Seconds())
	duration -= time.Duration(seconds) * time.Second

	milliseconds := int(duration.Milliseconds())

	parts := make([]string, 0, 5)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}
	if len(parts) == 0 || milliseconds > 0 {
		parts = append(parts, fmt.Sprintf("%dms", milliseconds))
	}

	return strings.Join(parts, " ")
}
