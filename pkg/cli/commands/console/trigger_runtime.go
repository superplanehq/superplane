package console

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type triggerCommand struct {
	canvasID   *string
	node       *string
	hook       *string
	parameters *string
}

// Execute invokes a node trigger hook (typically `run`) so the CLI can
// kick off the same operation users start from a Console node panel.
//
// The CLI accepts node ids or node names, and parameters can be supplied
// inline, from a file (`@path`), or from stdin (`-`). Parameters are sent
// to the API as the `parameters` map matching the proto payload.
func (c *triggerCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, valueOf(c.canvasID))
	if err != nil {
		return err
	}

	nodeRef := strings.TrimSpace(valueOf(c.node))
	if nodeRef == "" {
		return fmt.Errorf("--node is required")
	}

	hook := strings.TrimSpace(valueOf(c.hook))
	if hook == "" {
		hook = "run"
	}

	// Parse parameters before any network calls so a malformed --parameters
	// flag fails fast without hitting the API.
	parameters, err := loadTriggerParameters(valueOf(c.parameters), ctx.Cmd.InOrStdin())
	if err != nil {
		return err
	}

	nodeID, err := resolveNodeID(ctx, canvasID, nodeRef)
	if err != nil {
		return err
	}
	if nodeID == "" {
		return fmt.Errorf("node %q not found", nodeRef)
	}

	body := openapi_client.CanvasesInvokeNodeTriggerHookBody{}
	if parameters != nil {
		body.SetParameters(parameters)
	}

	response, _, err := ctx.API.CanvasNodeAPI.
		CanvasesInvokeNodeTriggerHook(ctx.Context, canvasID, nodeID, hook).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Triggered hook %q on node %s (canvas %s).\n", hook, nodeID, canvasID)
		return err
	})
}

// loadTriggerParameters supports three sources for the JSON payload:
//   - `-`              read from stdin
//   - `@path/to.json`  read from file
//   - `{...}`          inline JSON
//
// Returns nil when no parameters are provided so the API call sends an
// empty object (matching the UI default).
func loadTriggerParameters(input string, stdin io.Reader) (map[string]any, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, nil
	}

	var raw []byte
	switch {
	case trimmed == "-":
		var err error
		raw, err = io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read parameters from stdin: %w", err)
		}
	case strings.HasPrefix(trimmed, "@"):
		path := strings.TrimPrefix(trimmed, "@")
		// #nosec G304 - path is supplied by the CLI user.
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read parameters file %s: %w", path, err)
		}
		raw = data
	default:
		raw = []byte(trimmed)
	}

	if len(raw) == 0 {
		return nil, nil
	}

	parameters := map[string]any{}
	if err := json.Unmarshal(raw, &parameters); err != nil {
		return nil, fmt.Errorf("invalid JSON parameters: %w", err)
	}
	return parameters, nil
}
