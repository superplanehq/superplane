package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllowedRunnerTools(t *testing.T) {
	assert.Equal(t,
		[]string{"assign_task_owner", "suggest_task_owner"},
		AllowedRunnerTools([]RunnerToolScope{ScopeOwnerWrite, ScopeOwnerRead}),
	)
	assert.Equal(t, []string{"suggest_task_owner"}, AllowedRunnerTools([]RunnerToolScope{ScopeOwnerRead}))
	assert.Empty(t, AllowedRunnerTools(nil))
}

func TestFlowTools(t *testing.T) {
	// The agent tools docs page lists these tools for every run.
	shared := []string{
		"list_members", "assign_task_owner", "suggest_task_owner", "list_tasks", "create_task", "close_task",
		"list_checks", "add_checks", "remove_checks",
		"list_artifacts", "get_artifact", "add_artifact", "remove_artifact",
	}
	for flow, extra := range map[RunnerFlow][]string{
		RunnerFlowPlanning:        {"propose_plan"},
		RunnerFlowTask:            {"open_pull_request", "list_pull_requests"},
		RunnerFlowMergeConfidence: {"report_merge_check"},
	} {
		assert.Equal(t, append(append([]string{}, shared...), extra...), AllowedRunnerTools(ScopesForFlow(flow)), flow)
	}
	assert.Empty(t, ScopesForFlow("deploy"))
}

func TestTaskCapabilityScopes(t *testing.T) {
	artifactToken := TaskEnvironmentVariable{Name: EnvSuperplaneArtifactToken, Value: "token"}
	visualEvidence := TaskEnvironmentVariable{Name: EnvSuperplaneVisualEvidence, Value: "yes"}
	publishLinks := TaskEnvironmentVariable{Name: EnvSuperplanePublishLinks, Value: "yes"}

	for name, test := range map[string]struct {
		environment []TaskEnvironmentVariable
		want        []RunnerToolScope
	}{
		"no artifact settings":          {nil, []RunnerToolScope{}},
		"visual evidence":               {[]TaskEnvironmentVariable{artifactToken, visualEvidence}, []RunnerToolScope{ScopeEvidenceWrite}},
		"artifact token without links":  {[]TaskEnvironmentVariable{artifactToken}, []RunnerToolScope{ScopeEvidenceWrite}},
		"links only":                    {[]TaskEnvironmentVariable{artifactToken, publishLinks}, []RunnerToolScope{}},
		"visual evidence and the links": {[]TaskEnvironmentVariable{artifactToken, visualEvidence, publishLinks}, []RunnerToolScope{ScopeEvidenceWrite}},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.want, TaskCapabilityScopes(test.environment))
		})
	}
}
