package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllowedRunnerTools(t *testing.T) {
	assert.Equal(t, []string{"suggest_owners", "assign_owner"}, AllowedRunnerTools([]RunnerToolScope{ScopeOwnerWrite, ScopeOwnerRead}))
	assert.Equal(t, []string{"suggest_owners"}, AllowedRunnerTools([]RunnerToolScope{ScopeOwnerRead}))
	assert.Empty(t, AllowedRunnerTools(nil))
}

func TestEveryFlowCanUseEveryGlobalTool(t *testing.T) {
	want := []string{"suggest_owners", "assign_owner", "list_tasks", "create_task", "close_task", "remove_artifact"}
	for _, flow := range []RunnerFlow{RunnerFlowPlanning, RunnerFlowTask, RunnerFlowMergeConfidence} {
		assert.Equal(t, want, AllowedRunnerTools(ScopesForFlow(flow)), flow)
	}
	assert.Empty(t, ScopesForFlow("deploy"))
}
