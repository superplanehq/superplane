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

	ScopeOwnerRead        RunnerToolScope = "owner:read"
	ScopeOwnerWrite       RunnerToolScope = "owner:write"
	ScopeTasksRead        RunnerToolScope = "tasks:read"
	ScopeTasksCreate      RunnerToolScope = "tasks:create"
	ScopeTasksClose       RunnerToolScope = "tasks:close"
	ScopeArtifactsWrite   RunnerToolScope = "artifacts:write"
	ScopePlanWrite        RunnerToolScope = "plan:write"
	ScopeAttachmentsRead  RunnerToolScope = "attachments:read"
	ScopeEvidenceWrite    RunnerToolScope = "evidence:write"
	ScopeLinksWrite       RunnerToolScope = "links:write"
	ScopeMergeChecksWrite RunnerToolScope = "merge_checks:write"
)

// RunnerTool is one tool of the global runner MCP server and the scope that
// the backend requires on every call.
type RunnerTool struct {
	Name  string
	Scope RunnerToolScope
}

var runnerTools = []RunnerTool{
	{Name: "suggest_owners", Scope: ScopeOwnerRead},
	{Name: "assign_owner", Scope: ScopeOwnerWrite},
	{Name: "list_tasks", Scope: ScopeTasksRead},
	{Name: "create_task", Scope: ScopeTasksCreate},
	{Name: "close_task", Scope: ScopeTasksClose},
	{Name: "remove_artifact", Scope: ScopeArtifactsWrite},
	{Name: "propose_update", Scope: ScopePlanWrite},
	{Name: "inspect_attachment", Scope: ScopeAttachmentsRead},
	{Name: "inspect_screenshot", Scope: ScopeEvidenceWrite},
	{Name: "upload_artifact", Scope: ScopeEvidenceWrite},
	{Name: "report_visual_evidence_unavailable", Scope: ScopeEvidenceWrite},
	{Name: "publish_link", Scope: ScopeLinksWrite},
	{Name: "report_merge_check", Scope: ScopeMergeChecksWrite},
}

// Every flow receives these tools for now. Narrow the list here later.
var sharedRunnerScopes = []RunnerToolScope{
	ScopeOwnerRead,
	ScopeOwnerWrite,
	ScopeTasksRead,
	ScopeTasksCreate,
	ScopeTasksClose,
	ScopeArtifactsWrite,
}

// Flow tools work only in their flow. Task runs get the evidence and link
// scopes from the run settings, see TaskCapabilityScopes.
var runnerFlowScopes = map[RunnerFlow][]RunnerToolScope{
	RunnerFlowPlanning:        withScopes(ScopePlanWrite, ScopeAttachmentsRead),
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

// TaskCapabilityScopes returns the evidence and link scopes that the artifact
// attach call turned on for this run. The rule matches the legacy task
// artifact MCP server, so both servers list the same tools.
func TaskCapabilityScopes(environment []TaskEnvironmentVariable) []RunnerToolScope {
	scopes := []RunnerToolScope{}
	publishLinks := HasPublishLinks(environment)
	if HasVisualEvidence(environment) || (HasArtifactUploadToken(environment) && !publishLinks) {
		scopes = append(scopes, ScopeEvidenceWrite)
	}
	if publishLinks {
		scopes = append(scopes, ScopeLinksWrite)
	}
	return scopes
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
