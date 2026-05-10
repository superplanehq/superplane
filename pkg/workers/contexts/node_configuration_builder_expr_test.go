package contexts

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNodeConfigurationBuilder_ResolveExpression_DateWithTimezoneOption(t *testing.T) {
	// This is a regression test for expr runtime crashes when compiling with expr.Timezone("UTC")
	// and using date(...) in server-side expression resolution.
	b := NewNodeConfigurationBuilder(nil, uuid.Nil).WithInput(map[string]any{})

	out, err := b.ResolveTemplateExpressions(`{{ date("2026-03-17T01:02:03Z").Add(duration("1ns")).Format("2006-01-02T15:04:05.999999999Z07:00") }}`)
	require.NoError(t, err)
	require.Equal(t, "2026-03-17T01:02:03.000000001Z", out)
}

func TestNodeConfigurationBuilder_ResolveExpression_EventId(t *testing.T) {
	eventID := uuid.New()
	b := NewNodeConfigurationBuilder(nil, uuid.Nil).
		WithInput(map[string]any{}).
		WithRootEvent(&eventID)

	out, err := b.ResolveExpression("eventId()")
	require.NoError(t, err)
	require.Equal(t, eventID.String(), out)
}

func TestNodeConfigurationBuilder_ResolveExpression_EventIdNotAvailable(t *testing.T) {
	b := NewNodeConfigurationBuilder(nil, uuid.Nil).WithInput(map[string]any{})

	_, err := b.ResolveExpression("eventId()")
	require.Error(t, err)
	require.Contains(t, err.Error(), "eventId() is not available in this context")
}

func TestNodeConfigurationBuilder_ResolveExpression_ExecutionId(t *testing.T) {
	executionID := uuid.New()
	b := NewNodeConfigurationBuilder(nil, uuid.Nil).
		WithInput(map[string]any{}).
		WithCurrentExecution(&executionID)

	out, err := b.ResolveExpression("executionId()")
	require.NoError(t, err)
	require.Equal(t, executionID.String(), out)
}

func TestNodeConfigurationBuilder_ResolveExpression_ExecutionIdNotAvailable(t *testing.T) {
	b := NewNodeConfigurationBuilder(nil, uuid.Nil).WithInput(map[string]any{})

	_, err := b.ResolveExpression("executionId()")
	require.Error(t, err)
	require.Contains(t, err.Error(), "executionId() is not available in this context")
}

func TestNodeConfigurationBuilder_ResolveTemplateExpressions_RunIdentifiersInUrlTemplate(t *testing.T) {
	eventID := uuid.New()
	executionID := uuid.New()
	b := NewNodeConfigurationBuilder(nil, uuid.Nil).
		WithInput(map[string]any{}).
		WithRootEvent(&eventID).
		WithCurrentExecution(&executionID)

	out, err := b.ResolveTemplateExpressions("/canvases/x/runs/{{ executionId() }}?event={{ eventId() }}")
	require.NoError(t, err)
	require.Equal(t, "/canvases/x/runs/"+executionID.String()+"?event="+eventID.String(), out)
}
