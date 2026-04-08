package grafana

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
