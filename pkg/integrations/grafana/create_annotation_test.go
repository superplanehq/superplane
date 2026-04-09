package grafana

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__parseRelativeAnnotationTime(t *testing.T) {
	base := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "now",
			input:    "now",
			expected: base,
		},
		{
			name:     "offset forward",
			input:    "now+2h",
			expected: base.Add(2 * time.Hour),
		},
		{
			name:     "offset backward",
			input:    "now-30m",
			expected: base.Add(-30 * time.Minute),
		},
		{
			name:  "round day then offset",
			input: "now/d+8h",
			expected: time.Date(
				base.Year(),
				base.Month(),
				base.Day(),
				8,
				0,
				0,
				0,
				base.Location(),
			),
		},
		{
			name:  "previous month start",
			input: "now-1M/M",
			expected: time.Date(
				base.AddDate(0, -1, 0).Year(),
				base.AddDate(0, -1, 0).Month(),
				1,
				0,
				0,
				0,
				0,
				base.Location(),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, ok, err := parseRelativeAnnotationTime(tt.input, base)
			require.NoError(t, err)
			require.True(t, ok)
			require.Equal(t, tt.expected, parsed)
		})
	}
}

func Test__parseRelativeAnnotationTime__invalidUnit(t *testing.T) {
	_, ok, err := parseRelativeAnnotationTime("now+2x", time.Now().UTC())
	require.ErrorContains(t, err, `unsupported relative time unit "x"`)
	require.True(t, ok)
}

func Test__parseAnnotationTime__requiresExplicitTimezoneForLocalDateTime(t *testing.T) {
	_, err := parseAnnotationTime("2026-04-08T15:30")
	require.ErrorContains(t, err, "timezone is required for datetime-local values")
}

func Test__parseAnnotationTime__acceptsRFC3339(t *testing.T) {
	parsed, err := parseAnnotationTime("2026-04-08T15:30:00Z")
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, time.April, 8, 15, 30, 0, 0, time.UTC), parsed.UTC())
}

func Test__parseAnnotationTime__acceptsGoTimeStringOutput(t *testing.T) {
	parsed, err := parseAnnotationTime("2026-04-08 11:53:05.86655651 +0000 UTC")
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, time.April, 8, 11, 53, 5, 866556510, time.UTC), parsed.UTC())
}

func Test__CreateAnnotation__Configuration__timeFieldsAreStrings(t *testing.T) {
	component := &CreateAnnotation{}
	fields := component.Configuration()

	fieldTypes := map[string]string{}
	fieldDefaults := map[string]any{}
	fieldPlaceholders := map[string]string{}
	for _, field := range fields {
		fieldTypes[field.Name] = field.Type
		fieldDefaults[field.Name] = field.Default
		fieldPlaceholders[field.Name] = field.Placeholder
	}

	require.Equal(t, "string", fieldTypes["time"])
	require.Equal(t, "string", fieldTypes["timeEnd"])
	require.Equal(t, `{{ now() }}`, fieldDefaults["time"])
	require.Equal(t, `{{ now() }}`, fieldPlaceholders["time"])
	require.Nil(t, fieldDefaults["timeEnd"])
	require.Equal(t, `{{ now() + duration("24h") }}`, fieldPlaceholders["timeEnd"])
}

func Test__CreateAnnotation__Configuration__panelIsRequiredSingleResource(t *testing.T) {
	component := &CreateAnnotation{}
	fields := component.Configuration()

	for _, field := range fields {
		if field.Name != "panel" {
			continue
		}

		require.True(t, field.Required)
		require.False(t, field.Togglable)
		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.Resource)
		require.False(t, field.TypeOptions.Resource.Multi)
		return
	}

	t.Fatal("panel field not found")
}

func Test__resolveAnnotationPanelID(t *testing.T) {
	panelID, err := resolveAnnotationPanelID("7", nil)
	require.NoError(t, err)
	require.NotNil(t, panelID)
	require.Equal(t, int64(7), *panelID)

	legacyPanelID := int64(9)
	panelID, err = resolveAnnotationPanelID("", &legacyPanelID)
	require.NoError(t, err)
	require.NotNil(t, panelID)
	require.Equal(t, int64(9), *panelID)

	_, err = resolveAnnotationPanelID("abc", nil)
	require.ErrorContains(t, err, "panel must be a valid panel resource")
}

func Test__validateCreateAnnotationSpec__requiresPanel(t *testing.T) {
	err := validateCreateAnnotationSpec(CreateAnnotationSpec{
		Text: "deploy",
	})

	require.ErrorContains(t, err, "panel is required")
}

func Test__buildAnnotationURL(t *testing.T) {
	panelID := int64(7)
	url := buildAnnotationURL(
		"https://grafana.example.com/d/abc123/overview",
		&panelID,
		1712563200000,
		1712566800000,
	)

	require.Equal(
		t,
		"https://grafana.example.com/d/abc123/overview?from=1712563200000&to=1712566800000&viewPanel=7",
		url,
	)
}

func Test__buildAnnotationURL__pointAnnotationGetsContextWindow(t *testing.T) {
	url := buildAnnotationURL(
		"https://grafana.example.com/d/abc123/overview",
		nil,
		1712563200000,
		0,
	)

	require.Equal(
		t,
		"https://grafana.example.com/d/abc123/overview?from=1712562900000&to=1712563500000",
		url,
	)
}

func Test__CreateAnnotation__Execute__createsAnnotationForSelectedPanel(t *testing.T) {
	component := &CreateAnnotation{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":41}`)),
			},
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"dashboardUID": "dash-1",
			"panel":        "7",
			"text":         "deploy",
			"time":         "now",
		},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://grafana.example.com", "apiToken": "token"}},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)

	body, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, float64(7), payload["panelId"])

	require.Len(t, executionState.Payloads, 1)
	output := executionState.Payloads[0].(map[string]any)["data"].(CreateAnnotationOutput)
	require.Equal(t, int64(41), output.ID)
	require.NotEmpty(t, output.URL)
}
