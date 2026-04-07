package grafana

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__parseSilenceInstant__acceptsRFC3339(t *testing.T) {
	ref := time.Now().UTC().Truncate(time.Minute)
	s := ref.Format(time.RFC3339)
	tm, err := parseSilenceInstant(s)
	require.NoError(t, err)
	require.True(t, tm.Equal(ref))
}

func Test__parseSilenceInstant__acceptsTimeValue(t *testing.T) {
	ref := time.Now().UTC().Truncate(time.Minute)
	tm, err := parseSilenceInstant(ref)
	require.NoError(t, err)
	require.True(t, tm.Equal(ref))
}

func Test__parseSilenceInstant__acceptsNowExpressions(t *testing.T) {
	start := time.Now().UTC()

	now, err := parseSilenceInstant("now")
	require.NoError(t, err)
	require.WithinDuration(t, start, now, 2*time.Second)

	in5h, err := parseSilenceInstant("now+5h")
	require.NoError(t, err)
	require.WithinDuration(t, start.Add(5*time.Hour), in5h, 2*time.Second)

	minus30m, err := parseSilenceInstant("now-30m")
	require.NoError(t, err)
	require.WithinDuration(t, start.Add(-30*time.Minute), minus30m, 2*time.Second)
}

func Test__validateCreateSilenceSpec__matcherValueRequired(t *testing.T) {
	start := time.Now().UTC().Truncate(time.Minute)
	end := start.Add(time.Hour)
	err := validateCreateSilenceSpec(CreateSilenceSpec{
		Matchers: []SilenceMatcherInput{{Name: "alertname", Value: "  "}},
		StartsAt: start.Format(time.RFC3339),
		EndsAt:   end.Format(time.RFC3339),
		Comment:  "deploy",
	})
	require.ErrorContains(t, err, "value is required")
}

func Test__silenceMatcherFromInput__operators(t *testing.T) {
	cases := []struct {
		in   SilenceMatcherInput
		want SilenceMatcher
	}{
		{
			in:   SilenceMatcherInput{Name: "alertname", Value: "CPU", Operator: "="},
			want: SilenceMatcher{Name: "alertname", Value: "CPU", IsRegex: false, IsEqual: true},
		},
		{
			in:   SilenceMatcherInput{Name: "env", Value: "staging", Operator: "!="},
			want: SilenceMatcher{Name: "env", Value: "staging", IsRegex: false, IsEqual: false},
		},
		{
			in:   SilenceMatcherInput{Name: "alertname", Value: "High.*", Operator: "=~"},
			want: SilenceMatcher{Name: "alertname", Value: "High.*", IsRegex: true, IsEqual: true},
		},
		{
			in:   SilenceMatcherInput{Name: "team", Value: "infra-.*", Operator: "!~"},
			want: SilenceMatcher{Name: "team", Value: "infra-.*", IsRegex: true, IsEqual: false},
		},
		{
			in:   SilenceMatcherInput{Name: "a", Value: "b", Operator: "  =~  "},
			want: SilenceMatcher{Name: "a", Value: "b", IsRegex: true, IsEqual: true},
		},
		{
			in:   SilenceMatcherInput{Name: "x", Value: "y", IsRegex: true},
			want: SilenceMatcher{Name: "x", Value: "y", IsRegex: true, IsEqual: true},
		},
		{
			in:   SilenceMatcherInput{Name: "x", Value: "y", IsRegex: false},
			want: SilenceMatcher{Name: "x", Value: "y", IsRegex: false, IsEqual: true},
		},
	}
	for _, tc := range cases {
		got := silenceMatcherFromInput(tc.in)
		require.Equal(t, tc.want, got, "input %#v", tc.in)
	}
}

func Test__buildSilenceCreatedByFromOrgName__sanitizesAndPrefixes(t *testing.T) {
	require.Equal(t, "SuperPlane-Acme-Inc", buildSilenceCreatedByFromOrgName("Acme Inc", ""))
	require.Equal(t, "SuperPlane-acme.org", buildSilenceCreatedByFromOrgName("acme.org", ""))
	require.Equal(t, "SuperPlane-Acme-Org-Prod", buildSilenceCreatedByFromOrgName("Acme/Org (Prod)", ""))
	require.Equal(t, "SuperPlane-123e4567-e89b-12d3-a456-426614174000", buildSilenceCreatedByFromOrgName("", "123e4567-e89b-12d3-a456-426614174000"))
	require.Equal(t, "SuperPlane-unknown", buildSilenceCreatedByFromOrgName("", ""))
}

func Test__CreateSilence__Execute__endsAtMustBeAfterStartsAt(t *testing.T) {
	component := CreateSilence{}
	execCtx := &contexts.ExecutionStateContext{}

	start := time.Now().UTC().Truncate(time.Minute)
	end := start.Add(-time.Hour)

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"matchers": []any{map[string]any{"name": "alertname", "value": "HighCPU"}},
			"startsAt": start.Format(time.RFC3339),
			"endsAt":   end.Format(time.RFC3339),
			"comment":  "bad window",
		},
		HTTP:           &contexts.HTTPContext{},
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://grafana.example.com", "apiToken": "t"}},
		ExecutionState: execCtx,
	})

	require.ErrorContains(t, err, "endsAt must be after startsAt")
	require.False(t, execCtx.Finished)
}
