package runner

import "slices"

// RunnerFlow names the kind of agent run that holds a runner token.
type RunnerFlow string

// RunnerToolScope names one permission that a runner token can carry.
type RunnerToolScope string

const (
	RunnerFlowPlanning        RunnerFlow = "planning"
	RunnerFlowTask            RunnerFlow = "task"
	RunnerFlowMergeConfidence RunnerFlow = "merge_confidence"

	ScopeMembersRead      RunnerToolScope = "members:read"
	ScopeOwnerRead        RunnerToolScope = "owner:read"
	ScopeOwnerWrite       RunnerToolScope = "owner:write"
	ScopeTasksRead        RunnerToolScope = "tasks:read"
	ScopeTasksCreate      RunnerToolScope = "tasks:create"
	ScopeTasksClose       RunnerToolScope = "tasks:close"
	ScopeChecksRead       RunnerToolScope = "checks:read"
	ScopeChecksWrite      RunnerToolScope = "checks:write"
	ScopeArtifactsRead    RunnerToolScope = "artifacts:read"
	ScopeArtifactsWrite   RunnerToolScope = "artifacts:write"
	ScopePlanWrite        RunnerToolScope = "plan:write"
	ScopeEvidenceWrite    RunnerToolScope = "evidence:write"
	ScopeMergeChecksWrite RunnerToolScope = "merge_checks:write"
)

// RunnerTool is one tool of the global runner MCP server and the scope that
// the backend requires on every call.
type RunnerTool struct {
	Name  string
	Scope RunnerToolScope
}

var runnerTools = []RunnerTool{
	{Name: "list_members", Scope: ScopeMembersRead},
	{Name: "assign_task_owner", Scope: ScopeOwnerWrite},
	{Name: "suggest_task_owner", Scope: ScopeOwnerRead},
	{Name: "list_tasks", Scope: ScopeTasksRead},
	{Name: "create_task", Scope: ScopeTasksCreate},
	{Name: "close_task", Scope: ScopeTasksClose},
	{Name: "list_checks", Scope: ScopeChecksRead},
	{Name: "add_checks", Scope: ScopeChecksWrite},
	{Name: "remove_checks", Scope: ScopeChecksWrite},
	{Name: "list_artifacts", Scope: ScopeArtifactsRead},
	{Name: "get_artifact", Scope: ScopeArtifactsRead},
	{Name: "add_artifact", Scope: ScopeArtifactsWrite},
	{Name: "remove_artifact", Scope: ScopeArtifactsWrite},
	{Name: "propose_plan", Scope: ScopePlanWrite},
	{Name: "report_visual_evidence_unavailable", Scope: ScopeEvidenceWrite},
	{Name: "report_merge_check", Scope: ScopeMergeChecksWrite},
}

// Every flow receives these tools for now. Narrow the list here later.
var sharedRunnerScopes = []RunnerToolScope{
	ScopeMembersRead,
	ScopeOwnerRead,
	ScopeOwnerWrite,
	ScopeTasksRead,
	ScopeTasksCreate,
	ScopeTasksClose,
	ScopeChecksRead,
	ScopeChecksWrite,
	ScopeArtifactsRead,
	ScopeArtifactsWrite,
}

// Flow tools work only in their flow. Task runs get the evidence scope from
// the run settings, see TaskCapabilityScopes.
var runnerFlowScopes = map[RunnerFlow][]RunnerToolScope{
	RunnerFlowPlanning:        withScopes(ScopePlanWrite),
	RunnerFlowTask:            withScopes(),
	RunnerFlowMergeConfidence: withScopes(ScopeMergeChecksWrite),
}

func withScopes(extra ...RunnerToolScope) []RunnerToolScope {
	return append(slices.Clone(sharedRunnerScopes), extra...)
}

func (f RunnerFlow) Valid() bool {
	_, ok := runnerFlowScopes[f]
	return ok
}

// ScopesForFlow returns the default scopes that a run of this flow receives.
func ScopesForFlow(flow RunnerFlow) []RunnerToolScope {
	return slices.Clone(runnerFlowScopes[flow])
}

// TaskCapabilityScopes returns the evidence scope when the artifact attach
// call turned it on for this run.
func TaskCapabilityScopes(environment []TaskEnvironmentVariable) []RunnerToolScope {
	if HasVisualEvidence(environment) || (HasArtifactUploadToken(environment) && !HasPublishLinks(environment)) {
		return []RunnerToolScope{ScopeEvidenceWrite}
	}
	return []RunnerToolScope{}
}

// AllowedRunnerTools returns the tool names that the scopes allow, in catalog
// order.
func AllowedRunnerTools(scopes []RunnerToolScope) []string {
	names := []string{}
	for _, tool := range runnerTools {
		if slices.Contains(scopes, tool.Scope) {
			names = append(names, tool.Name)
		}
	}
	return names
}
