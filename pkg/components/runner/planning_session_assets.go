package runner

import _ "embed"

// Planning sessions run every code runner (Claude, Codex, OpenRouter) in a
// read-only "explore, then propose a draft" mode. These assets are shared by
// every runner so the wait loop and the MCP tool contract behave identically
// regardless of which agent CLI is executing the turn.

//go:embed planning_session_mcp.js
var planningSessionMCPScript string

//go:embed mcp.json
var planningSessionMCPConfig string

//go:embed follow_up_loop.js
var followUpLoopScript string

// PlanningSessionMCPScript is the stdio MCP server exposing propose_draft and
// survey. Runners that speak MCP (Claude, Codex) ship it under
// SUPERPLANE_TASK_DIR as planning_session_mcp.js.
func PlanningSessionMCPScript() string { return planningSessionMCPScript }

// PlanningSessionMCPConfigJSON is the static mcp.json reference config.
func PlanningSessionMCPConfigJSON() string { return planningSessionMCPConfig }

// FollowUpLoopScript waits on SuperPlane for the next user message and reruns
// run.js --continue (or the runner's equivalent) for each one.
func FollowUpLoopScript() string { return followUpLoopScript }

// PlanningSessionMCPFiles returns the MCP server + static config task files.
// Only attach these when HasPlanningSessionToken is true; line automations
// never receive them.
func PlanningSessionMCPFiles() []BrokerTaskFile {
	return []BrokerTaskFile{
		{Path: "planning_session_mcp.js", Content: planningSessionMCPScript, Mode: "0644"},
		{Path: "mcp.json", Content: planningSessionMCPConfig, Mode: "0644"},
	}
}

// FollowUpLoopFile returns the shared wait-loop task file. Only attach this
// when HasPlanningSessionToken is true.
func FollowUpLoopFile() BrokerTaskFile {
	return BrokerTaskFile{Path: "follow_up_loop.js", Content: followUpLoopScript, Mode: "0644"}
}
