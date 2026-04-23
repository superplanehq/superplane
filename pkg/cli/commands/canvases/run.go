package canvases

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type runCommand struct {
	node        *string
	template    *string
	payloadJSON *string
}

func (c *runCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) != 1 {
		return fmt.Errorf("<name-or-id> of the canvas is required")
	}

	nodeID := ""
	if c.node != nil {
		nodeID = strings.TrimSpace(*c.node)
	}
	if nodeID == "" {
		return fmt.Errorf("--node is required")
	}

	templateName := ""
	if c.template != nil {
		templateName = strings.TrimSpace(*c.template)
	}
	if templateName == "" {
		return fmt.Errorf("--template is required")
	}

	payloadOverride := ""
	if c.payloadJSON != nil {
		payloadOverride = strings.TrimSpace(*c.payloadJSON)
	}

	canvasID, err := findCanvasID(ctx, ctx.API, ctx.Args[0])
	if err != nil {
		return err
	}

	describeResp, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return err
	}
	if describeResp == nil || describeResp.Canvas == nil {
		return fmt.Errorf("canvas %q not found", ctx.Args[0])
	}

	_, err = findStartTriggerNode(*describeResp.Canvas, nodeID)
	if err != nil {
		return err
	}

	params := map[string]any{"template": templateName}
	if payloadOverride != "" {
		parsed := map[string]any{}
		if err := json.Unmarshal([]byte(payloadOverride), &parsed); err != nil {
			return fmt.Errorf("invalid --payload-json: %w", err)
		}
		params["payload"] = parsed
	}

	body := openapi_client.NewCanvasesInvokeNodeTriggerActionBody()
	body.SetParameters(params)

	resp, _, err := ctx.API.CanvasNodeAPI.
		CanvasesInvokeNodeTriggerAction(ctx.Context, canvasID, nodeID, "run").
		Body(*body).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(resp)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Run started\n")
		return err
	})
}

func findStartTriggerNode(
	canvas openapi_client.CanvasesCanvas,
	nodeID string,
) (openapi_client.SuperplaneComponentsNode, error) {
	spec := canvas.GetSpec()
	for _, node := range spec.GetNodes() {
		if node.GetId() != nodeID {
			continue
		}

		if node.GetType() != openapi_client.COMPONENTSNODETYPE_TYPE_TRIGGER {
			return openapi_client.SuperplaneComponentsNode{}, fmt.Errorf(
				"node %q is not a trigger (type=%s); only Manual Run (start) triggers can be run from the CLI",
				nodeID, node.GetType(),
			)
		}

		trigger := node.GetTrigger()
		if trigger.GetName() != "start" {
			return openapi_client.SuperplaneComponentsNode{}, fmt.Errorf(
				"node %q is a %q trigger, not a Manual Run (start) trigger",
				nodeID, trigger.GetName(),
			)
		}

		return node, nil
	}

	return openapi_client.SuperplaneComponentsNode{}, fmt.Errorf("node %q not found on canvas", nodeID)
}
