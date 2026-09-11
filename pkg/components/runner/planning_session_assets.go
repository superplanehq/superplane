package runner

import (
	"context"
	"strings"

	_ "embed"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

// Planning sessions run every code runner (Claude, Codex, OpenCode/OpenRouter)
// in a read-only "explore, then propose a draft" mode. These assets are shared
// by every runner so the wait loop and the MCP tool contract behave identically
// regardless of which agent CLI is executing the turn.

//go:embed planning_session_mcp.js
var planningSessionMCPScript string

//go:embed analysis_protocol.js
var analysisProtocolScript string

//go:embed mcp.json
var planningSessionMCPConfig string

//go:embed follow_up_loop.js
var followUpLoopScript string

// PlanningSessionMCPScript is the stdio MCP server exposing propose_draft and
// survey. Runners that speak MCP (Claude, Codex, OpenCode) ship it under
// SUPERPLANE_TASK_DIR as planning_session_mcp.js.
func PlanningSessionMCPScript() string { return planningSessionMCPScript }

// PlanningSessionMCPConfigJSON is the static mcp.json reference config.
func PlanningSessionMCPConfigJSON() string { return planningSessionMCPConfig }

// FollowUpLoopScript waits on SuperPlane for the next user message and reruns
// run.js --continue (or the runner's equivalent) for each one.
func FollowUpLoopScript() string { return followUpLoopScript }

// PlanningSessionMCPScriptFile is just the MCP server task file. Attach it
// together with PlanningSessionMCPConfigFile for runners that speak MCP.
func PlanningSessionMCPScriptFile() BrokerTaskFile {
	return BrokerTaskFile{Path: "planning_session_mcp.js", Content: planningSessionMCPScript, Mode: "0644"}
}

// PlanningSessionMCPConfigFile is the static mcp.json reference config file.
func PlanningSessionMCPConfigFile() BrokerTaskFile {
	return BrokerTaskFile{Path: "mcp.json", Content: planningSessionMCPConfig, Mode: "0644"}
}

func PlanningSessionProtocolFile() BrokerTaskFile {
	return BrokerTaskFile{Path: "analysis_protocol.js", Content: analysisProtocolScript, Mode: "0644"}
}

// PlanningSessionMCPFiles returns the MCP server, static config, and analysis
// protocol task files. Only attach these when HasPlanningSessionToken is true.
func PlanningSessionMCPFiles() []BrokerTaskFile {
	return []BrokerTaskFile{PlanningSessionMCPScriptFile(), PlanningSessionMCPConfigFile(), PlanningSessionProtocolFile()}
}

// FollowUpLoopFile returns the shared wait-loop task file. Only attach this
// when HasPlanningSessionToken is true.
func FollowUpLoopFile() BrokerTaskFile {
	return BrokerTaskFile{Path: "follow_up_loop.js", Content: followUpLoopScript, Mode: "0644"}
}

func AppendPlanningSessionContinuation(ctx core.ExecutionContext, environment []BrokerEnvironmentVariable, files []BrokerTaskFile) []BrokerTaskFile {
	if !HasPlanningSessionToken(environment) {
		return files
	}
	if file := PlanningSessionContinuationFile(ctx); file != nil {
		return append(files, *file)
	}
	return files
}

func PlanningSessionContinuationFile(ctx core.ExecutionContext) *BrokerTaskFile {
	if ctx.RunID == uuid.Nil {
		return nil
	}
	db := database.DB(context.Background())
	session, err := models.FindPlanningSessionByRun(db, ctx.RunID)
	if err != nil {
		return nil
	}
	text, err := models.AnalysisContinuationText(db, session)
	if err != nil || strings.TrimSpace(text) == "" {
		return nil
	}
	return &BrokerTaskFile{Path: "analysis_continuation.md", Content: text, Mode: "0644"}
}
