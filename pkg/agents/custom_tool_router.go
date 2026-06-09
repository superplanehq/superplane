package agents

import (
	"context"
	"fmt"
)

// CustomToolRouter dispatches Managed Agent custom tool calls by name. It keeps
// the stream worker independent from individual tool implementations.
type CustomToolRouter struct {
	executors map[string]CustomToolExecutor
}

func NewCustomToolRouter(executors ...CustomToolExecutor) *CustomToolRouter {
	router := &CustomToolRouter{executors: map[string]CustomToolExecutor{}}
	for _, executor := range executors {
		if named, ok := executor.(interface{ CustomToolName() string }); ok {
			router.executors[named.CustomToolName()] = executor
		}
	}
	return router
}

func (r *CustomToolRouter) ExecuteCustomTool(ctx context.Context, session AgentSessionContext, toolUse CustomToolUse) CustomToolResult {
	if r == nil {
		return customToolError(toolUse.ID, "custom tool router is not configured")
	}

	executor, ok := r.executors[toolUse.Name]
	if !ok {
		return customToolError(toolUse.ID, fmt.Sprintf("unsupported custom tool %q", toolUse.Name))
	}

	return executor.ExecuteCustomTool(ctx, session, toolUse)
}
