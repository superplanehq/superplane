package tasks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestFormatTaskState(t *testing.T) {
	cases := []struct {
		state openapi_client.FactoriesWorkOrderState
		want  string
	}{
		{openapi_client.FACTORIESWORKORDERSTATE_STATE_DRAFT, "Draft"},
		{openapi_client.FACTORIESWORKORDERSTATE_STATE_OPEN, "Open"},
		{openapi_client.FACTORIESWORKORDERSTATE_STATE_CLOSED, "Closed"},
		{openapi_client.FACTORIESWORKORDERSTATE_STATE_UNSPECIFIED, "-"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, formatTaskState(tc.state))
	}
}

func TestFormatTaskResult(t *testing.T) {
	cases := []struct {
		result openapi_client.FactoriesWorkOrderResult
		want   string
	}{
		{openapi_client.FACTORIESWORKORDERRESULT_RESULT_COMPLETED, "Completed"},
		{openapi_client.FACTORIESWORKORDERRESULT_RESULT_REJECTED, "Rejected"},
		{openapi_client.FACTORIESWORKORDERRESULT_RESULT_FAILED, "Failed"},
		{openapi_client.FACTORIESWORKORDERRESULT_RESULT_UNSPECIFIED, "-"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, formatTaskResult(tc.result))
	}
}

func TestFormatRelativeTimeAt(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		value time.Time
		want  string
	}{
		{name: "zero", value: time.Time{}, want: "-"},
		{name: "one second", value: now.Add(-1 * time.Second), want: "1s ago"},
		{name: "many seconds", value: now.Add(-45 * time.Second), want: "45s ago"},
		{name: "one minute", value: now.Add(-1 * time.Minute), want: "1m ago"},
		{name: "many minutes", value: now.Add(-30 * time.Minute), want: "30m ago"},
		{name: "one hour", value: now.Add(-1 * time.Hour), want: "1h ago"},
		{name: "many hours", value: now.Add(-6 * time.Hour), want: "6h ago"},
		{name: "one day", value: now.Add(-24 * time.Hour), want: "1d ago"},
		{name: "many days", value: now.Add(-72 * time.Hour), want: "3d ago"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatRelativeTimeAt(tc.value, now))
		})
	}
}

func TestDescribeStatusTransition(t *testing.T) {
	cases := []struct {
		from, to, result string
		want             string
	}{
		{"", "draft", "", "Task created"},
		{"draft", "open", "", "Task opened"},
		{"open", "draft", "", "Task moved back to Draft"},
		{"open", "closed", "completed", "Task closed as Completed"},
		{"open", "closed", "", "Task closed"},
		{"closed", "open", "", "Task reopened"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, describeStatusTransition(tc.from, tc.to, tc.result))
	}
}
