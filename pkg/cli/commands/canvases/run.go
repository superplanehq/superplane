package canvases

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type runCommand struct {
	node     *string
	template *string
	payload  *string
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

	params := map[string]any{"template": templateName}
	if c.payload != nil {
		raw := strings.TrimSpace(*c.payload)
		if raw != "" {
			parsed, err := loadPayload(raw)
			if err != nil {
				return err
			}
			params["payload"] = parsed
		}
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

	body := openapi_client.NewCanvasesInvokeNodeTriggerHookBody()
	body.SetParameters(params)

	resp, _, err := ctx.API.CanvasNodeAPI.
		CanvasesInvokeNodeTriggerHook(ctx.Context, canvasID, nodeID, "run").
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

// loadPayload accepts either an inline JSON object/array or a path to a file
// containing JSON. Inline JSON is detected by a leading '{' or '['; anything
// else is treated as a file path.
func loadPayload(raw string) (map[string]any, error) {
	trimmed := strings.TrimLeftFunc(raw, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' })
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		parsed := map[string]any{}
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			return nil, fmt.Errorf("invalid --payload (inline JSON): %w", err)
		}
		return parsed, nil
	}

	data, err := os.ReadFile(raw)
	if err != nil {
		return nil, fmt.Errorf("--payload: cannot read file %q: %w", raw, err)
	}

	parsed := map[string]any{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("--payload: %s does not contain a valid JSON object: %w", raw, err)
	}
	return parsed, nil
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

		triggerName := node.GetComponent()
		if triggerName != "start" {
			return openapi_client.SuperplaneComponentsNode{}, fmt.Errorf(
				"node %q is a %q trigger, not a Manual Run (start) trigger",
				nodeID, triggerName,
			)
		}

		return node, nil
	}

	return openapi_client.SuperplaneComponentsNode{}, fmt.Errorf("node %q not found on canvas", nodeID)
}
