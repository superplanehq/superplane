package runner

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllowedRunnerTools(t *testing.T) {
	assert.Equal(t,
		[]string{"assign_task_owner", "suggest_task_owner", "suggest_owners", "assign_owner"},
		AllowedRunnerTools([]RunnerToolScope{ScopeOwnerWrite, ScopeOwnerRead}),
	)
	assert.Equal(t, []string{"suggest_task_owner", "suggest_owners"}, AllowedRunnerTools([]RunnerToolScope{ScopeOwnerRead}))
	assert.Empty(t, AllowedRunnerTools(nil))
}

func TestFlowTools(t *testing.T) {
	// The agent tools docs page lists these tools for every run.
	shared := []string{
		"list_members", "assign_task_owner", "suggest_task_owner", "list_tasks", "create_task", "close_task",
		"list_checks", "add_checks", "remove_checks",
		"list_artifacts", "get_artifact", "add_artifact", "remove_artifact",
	}
	for flow, test := range map[RunnerFlow]struct {
		current []string
		aliases []string
	}{
		RunnerFlowPlanning:        {[]string{"propose_plan"}, []string{"suggest_owners", "assign_owner", "propose_update", "inspect_attachment"}},
		RunnerFlowTask:            {nil, []string{"suggest_owners", "assign_owner"}},
		RunnerFlowMergeConfidence: {[]string{"report_merge_check"}, []string{"suggest_owners", "assign_owner"}},
	} {
		want := append(append(append([]string{}, shared...), test.current...), test.aliases...)
		assert.Equal(t, want, AllowedRunnerTools(ScopesForFlow(flow)), flow)
	}
	assert.Empty(t, ScopesForFlow("deploy"))
}

func TestRunnerToolAliasesNameACurrentToolWithTheirPreviousScope(t *testing.T) {
	for _, tool := range runnerTools {
		if tool.AliasOf == "" {
			continue
		}
		index := slices.IndexFunc(runnerTools, func(candidate RunnerTool) bool { return candidate.Name == tool.AliasOf })
		if assert.GreaterOrEqual(t, index, 0, tool.Name) {
			assert.Empty(t, runnerTools[index].AliasOf, tool.Name)
		}
	}
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
		"links only":                    {[]TaskEnvironmentVariable{artifactToken, publishLinks}, []RunnerToolScope{ScopeLinksWrite}},
		"visual evidence and the links": {[]TaskEnvironmentVariable{artifactToken, visualEvidence, publishLinks}, []RunnerToolScope{ScopeEvidenceWrite, ScopeLinksWrite}},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.want, TaskCapabilityScopes(test.environment))
		})
	}
}
