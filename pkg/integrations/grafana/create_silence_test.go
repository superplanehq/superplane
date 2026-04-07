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
