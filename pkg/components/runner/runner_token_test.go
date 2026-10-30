package runner

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/jwt"
)

func taskRunnerScope() RunnerScope {
	return RunnerScope{
		OrganizationID:  uuid.New(),
		FactoryID:       uuid.New(),
		WorkOrderID:     uuid.New(),
		CanvasID:        uuid.New(),
		CanvasRunID:     uuid.New(),
		NodeExecutionID: uuid.New(),
		NodeID:          "agent",
		Flow:            RunnerFlowTask,
		Scopes:          ScopesForFlow(RunnerFlowTask),
	}
}

func TestRunnerTokenRoundTrip(t *testing.T) {
	signer := jwt.NewSigner("test")

	t.Run("task flow", func(t *testing.T) {
		scope := taskRunnerScope()
		token, err := MintRunnerToken(signer, scope, time.Minute)
		require.NoError(t, err)

		parsed, err := ParseRunnerToken(signer, token)
		require.NoError(t, err)
		assert.Equal(t, scope, *parsed)
		assert.True(t, parsed.HasScope(ScopeOwnerWrite))
	})

	t.Run("planning flow has a session and no task", func(t *testing.T) {
		scope := taskRunnerScope()
		scope.Flow = RunnerFlowPlanning
		scope.WorkOrderID = uuid.Nil
		scope.PlanningSessionID = uuid.New()
		token, err := MintRunnerToken(signer, scope, time.Minute)
		require.NoError(t, err)

		parsed, err := ParseRunnerToken(signer, token)
		require.NoError(t, err)
		assert.Equal(t, scope, *parsed)
	})
}

func TestRunnerTokenRejectsIncompleteScopes(t *testing.T) {
	signer := jwt.NewSigner("test")
	for name, change := range map[string]func(*RunnerScope){
		"no execution":             func(s *RunnerScope) { s.NodeExecutionID = uuid.Nil },
		"no node":                  func(s *RunnerScope) { s.NodeID = " " },
		"unknown flow":             func(s *RunnerScope) { s.Flow = "deploy" },
		"task flow without a task": func(s *RunnerScope) { s.WorkOrderID = uuid.Nil },
		"planning without session": func(s *RunnerScope) { s.Flow = RunnerFlowPlanning },
	} {
		t.Run(name, func(t *testing.T) {
			scope := taskRunnerScope()
			change(&scope)
			_, err := MintRunnerToken(signer, scope, time.Minute)
			assert.Error(t, err)
		})
	}
}

func TestParseRunnerTokenRejectsOtherTokens(t *testing.T) {
	signer := jwt.NewSigner("test")
	scope := taskRunnerScope()
	legacy, err := MintTaskOwnerToken(signer, TaskOwnerScope{
		OrganizationID: scope.OrganizationID, FactoryID: scope.FactoryID, WorkOrderID: scope.WorkOrderID,
		CanvasID: scope.CanvasID, CanvasRunID: scope.CanvasRunID, NodeExecutionID: scope.NodeExecutionID, NodeID: scope.NodeID,
	}, time.Minute)
	require.NoError(t, err)
	_, err = ParseRunnerToken(signer, legacy)
	assert.ErrorContains(t, err, "purpose")

	token, err := MintRunnerToken(signer, scope, time.Minute)
	require.NoError(t, err)
	_, err = ParseRunnerToken(jwt.NewSigner("other"), token)
	assert.Error(t, err)
}
