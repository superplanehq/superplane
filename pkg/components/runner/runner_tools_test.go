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

func TestFlowTools(t *testing.T) {
	shared := []string{"suggest_owners", "assign_owner", "list_tasks", "create_task", "close_task", "remove_artifact"}
	for flow, extra := range map[RunnerFlow][]string{
		RunnerFlowPlanning:        {"propose_update", "inspect_attachment"},
		RunnerFlowTask:            nil,
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
		"links only":                    {[]TaskEnvironmentVariable{artifactToken, publishLinks}, []RunnerToolScope{ScopeLinksWrite}},
		"visual evidence and the links": {[]TaskEnvironmentVariable{artifactToken, visualEvidence, publishLinks}, []RunnerToolScope{ScopeEvidenceWrite, ScopeLinksWrite}},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.want, TaskCapabilityScopes(test.environment))
		})
	}
}
