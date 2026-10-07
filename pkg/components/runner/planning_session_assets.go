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

// Analysis sessions run every code runner (Claude, Codex, OpenCode/OpenRouter)
// in a read-only task-refinement mode. These assets keep the wait loop and MCP
// contract identical for every supported agent CLI.

//go:embed planning_session_mcp.js
var planningSessionMCPScript string

//go:embed analysis_protocol.js
var analysisProtocolScript string

//go:embed analysis_protocol_review.md
var analysisProtocolReviewMarkdown string

//go:embed analysis_user_prompt_review.md
var analysisUserPromptReviewMarkdown string

//go:embed mcp.json
var planningSessionMCPConfig string

//go:embed follow_up_loop.js
var followUpLoopScript string

//go:embed attachment_limit.js
var attachmentLimitScript string

// PlanningSessionMCPScript is the stdio MCP server exposing task-refinement
// tools. Runners that speak MCP (Claude, Codex, OpenCode) ship it under
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

// PlanningSessionProtocolMarkdown returns the hardcoded refinement system
// protocol. Runners append this. The canvas prompt must not replace it.
func PlanningSessionProtocolMarkdown() string {
	return strings.TrimSpace(analysisProtocolReviewMarkdown)
}

// PlanningSessionUserPromptMarkdown is the Refine Task prompt: tone, the
// 1 through 3 sub-parameters, and plan shape. Factories can edit that node
// prompt. A saved canvas prompt is not rewritten.
func PlanningSessionUserPromptMarkdown() string {
	return strings.TrimSpace(analysisUserPromptReviewMarkdown)
}

// PlanningSessionUserPromptReviewMarkdown is the Refine Task prompt.
func PlanningSessionUserPromptReviewMarkdown() string {
	return PlanningSessionUserPromptMarkdown()
}

// PlanningSessionProtocolMarkdownFile ships the review protocol to runners.
func PlanningSessionProtocolMarkdownFile() BrokerTaskFile {
	return BrokerTaskFile{Path: "analysis_protocol_review.md", Content: analysisProtocolReviewMarkdown, Mode: "0644"}
}

func PlanningSessionReviewProtocolMarkdownFile() BrokerTaskFile {
	return PlanningSessionProtocolMarkdownFile()
}

// PlanningSessionMCPFiles returns the MCP server, attachment limit, static
// config, and analysis protocol task files. Only attach these when
// HasPlanningSessionToken is true.
func PlanningSessionMCPFiles() []BrokerTaskFile {
	return []BrokerTaskFile{
		PlanningSessionMCPScriptFile(),
		AttachmentLimitFile(),
		PlanningSessionMCPConfigFile(),
		PlanningSessionProtocolFile(),
		PlanningSessionProtocolMarkdownFile(),
	}
}

func AttachmentLimitFile() BrokerTaskFile {
	return BrokerTaskFile{Path: "attachment_limit.js", Content: attachmentLimitScript, Mode: "0644"}
}

func AppendAttachmentLimitFile(files []BrokerTaskFile) []BrokerTaskFile {
	return appendUniqueTaskFiles(files, AttachmentLimitFile())
}

func AppendPlanningSessionMCPFiles(files []BrokerTaskFile) []BrokerTaskFile {
	return appendUniqueTaskFiles(files, PlanningSessionMCPFiles()...)
}

func appendUniqueTaskFiles(files []BrokerTaskFile, extra ...BrokerTaskFile) []BrokerTaskFile {
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		seen[file.Path] = struct{}{}
	}
	for _, file := range extra {
		if _, ok := seen[file.Path]; ok {
			continue
		}
		seen[file.Path] = struct{}{}
		files = append(files, file)
	}
	return files
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
	if err != nil || !session.IsAnalysisSession() {
		return nil
	}
	text, err := models.AnalysisContinuationText(db, session)
	if err != nil || strings.TrimSpace(text) == "" {
		return nil
	}
	return &BrokerTaskFile{Path: "analysis_continuation.md", Content: text, Mode: "0644"}
}
