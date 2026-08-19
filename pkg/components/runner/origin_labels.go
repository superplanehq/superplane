package runner

import "github.com/superplanehq/superplane/pkg/core"

// OriginLabelsForTask are optional labels for task-broker metrics / Dash0 alerts.
func OriginLabelsForTask(ctx core.ExecutionContext) map[string]string {
	canvasID := ctx.WorkflowID
	organizationID := ctx.OrganizationID
	canvas := ctx.CanvasName
	node := ctx.NodeName
	if canvasID == "" && organizationID == "" && canvas == "" && node == "" {
		return nil
	}

	labels := make(map[string]string, 4)
	if canvasID != "" {
		labels["canvas_id"] = canvasID
	}
	if organizationID != "" {
		labels["organization_id"] = organizationID
	}
	if canvas != "" {
		labels["canvas_name"] = canvas
	}
	if node != "" {
		labels["node_name"] = node
	}
	return labels
}
