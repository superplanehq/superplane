package mcpserver

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/logging"
)

func (rt *Runtime) runMergeConfidence(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	factory, err := rt.beginWorkspaceCall(ctx, claims, args, "work_orders:update")
	if err != nil {
		return ToolResult{}, err
	}
	task, err := stringArg(args, "task")
	if err != nil {
		return ToolResult{}, err
	}
	result, err := factories.RunMergeConfidence(toolContext(ctx, claims), rt.Intake, claims.OrgID.String(), factories.RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: task})
	if err != nil {
		return ToolResult{}, actionError(err)
	}
	logging.AddRequestLogFields(ctx, log.Fields{"object": "task", "object_id": result.Order.ID.String()})
	return TextResult(mustJSON(map[string]any{
		"id":      result.Order.ID.String(),
		"key":     factory.WorkOrderKey(result.Order.Number),
		"url":     newTaskLinks(rt.BaseURL, factory).url(result.Order.Number),
		"runs":    result.Runs,
		"skipped": result.Skipped,
	})), nil
}
