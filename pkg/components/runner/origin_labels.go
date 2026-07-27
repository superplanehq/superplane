package runner

import "github.com/superplanehq/superplane/pkg/core"

// OriginLabelsForTask are optional labels for task-broker metrics / Dash0 alerts.
func OriginLabelsForTask(ctx core.ExecutionContext) map[string]string {
	canvas := ctx.CanvasName
	node := ctx.NodeName
	if canvas == "" && node == "" {
		return nil
	}
	labels := make(map[string]string, 2)
	if canvas != "" {
		labels["canvas_name"] = canvas
	}
	if node != "" {
		labels["node_name"] = node
	}
	return labels
}
