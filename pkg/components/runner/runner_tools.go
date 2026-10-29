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

	ScopeOwnerRead  RunnerToolScope = "owner:read"
	ScopeOwnerWrite RunnerToolScope = "owner:write"
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
}

var runnerFlowScopes = map[RunnerFlow][]RunnerToolScope{
	RunnerFlowPlanning:        {ScopeOwnerRead, ScopeOwnerWrite},
	RunnerFlowTask:            {ScopeOwnerRead, ScopeOwnerWrite},
	RunnerFlowMergeConfidence: {ScopeOwnerRead, ScopeOwnerWrite},
}

func (f RunnerFlow) Valid() bool {
	_, ok := runnerFlowScopes[f]
	return ok
}

// ScopesForFlow returns the default scopes that a run of this flow receives.
func ScopesForFlow(flow RunnerFlow) []RunnerToolScope {
	return slices.Clone(runnerFlowScopes[flow])
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
