package canvases

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
	"github.com/superplanehq/superplane/pkg/triggers/start"
)

type runCommand struct {
	trigger  *string
	template *string
	payload  *string
	replay   *string
}

func (c *runCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) != 1 {
		return fmt.Errorf("canvas name or id is required as the first argument")
	}

	trigger := strings.TrimSpace(ptrVal(c.trigger))
	if trigger == "" {
		return fmt.Errorf("--trigger is required")
	}

	replay := strings.TrimSpace(ptrVal(c.replay))
	templateName := strings.TrimSpace(ptrVal(c.template))
	if replay != "" && templateName != "" {
		return fmt.Errorf("cannot use --template together with --replay")
	}
	if replay == "" && templateName == "" {
		return fmt.Errorf("either --template or --replay is required")
	}

	payloadArg := strings.TrimSpace(ptrVal(c.payload))
	if replay != "" && payloadArg != "" {
		return fmt.Errorf("cannot use --payload with --replay")
	}

	canvasID, err := findCanvasID(ctx, ctx.API, ctx.Args[0])
	if err != nil {
		return err
	}

	if replay != "" {
		return c.executeReplay(ctx, canvasID, trigger, replay)
	}

	return c.executeManualHook(ctx, canvasID, trigger, templateName, payloadArg)
}

func (c *runCommand) executeReplay(ctx core.CommandContext, canvasID, triggerNodeID, eventID string) error {
	response, _, err := ctx.API.CanvasNodeAPI.
		CanvasesReemitTriggerEvent(ctx.Context, canvasID, triggerNodeID, eventID).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Event re-emitted: %s\n", response.GetEventId())
		return err
	})
}

func (c *runCommand) executeManualHook(
	ctx core.CommandContext,
	canvasID, triggerNodeID, templateName, payloadArg string,
) error {
	parameters := map[string]interface{}{
		"template": templateName,
	}

	if payloadArg != "" {
		payload, err := parseJSONObjectPayload(payloadArg)
		if err != nil {
			return err
		}
		parameters["payload"] = payload
	}

	body := openapi_client.NewCanvasesInvokeNodeTriggerHookBody()
	body.SetParameters(parameters)

	response, _, err := ctx.API.CanvasNodeAPI.
		CanvasesInvokeNodeTriggerHook(ctx.Context, canvasID, triggerNodeID, manual.HookRun).
		Body(*body).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if !response.HasResult() {
			_, err := fmt.Fprintln(stdout, "Manual run hook completed.")
			return err
		}
		encoded, err := json.MarshalIndent(response.GetResult(), "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "%s\n", encoded)
		return err
	})
}

// parseJSONObjectPayload accepts:
//   - Inline JSON object, e.g. '{"message":"hi"}'
//   - A path prefixed with @, e.g. '@payload.json' (reads that file)
//   - Otherwise a filesystem path to a JSON file (same contents rules as @)
func parseJSONObjectPayload(s string) (map[string]interface{}, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("payload value is empty")
	}

	if strings.HasPrefix(s, "@") {
		path := strings.TrimSpace(strings.TrimPrefix(s, "@"))
		if path == "" {
			return nil, fmt.Errorf("payload @-path is empty")
		}
		return readJSONObjectFromFile(path)
	}

	if strings.HasPrefix(s, "{") {
		var decoded interface{}
		if err := json.Unmarshal([]byte(s), &decoded); err != nil {
			return nil, fmt.Errorf("parse payload as JSON: %w", err)
		}
		obj, ok := decoded.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("payload JSON must be a single object at the top level")
		}
		return obj, nil
	}

	return readJSONObjectFromFile(s)
}

func readJSONObjectFromFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read payload from file %q: %w", path, err)
	}

	var decoded interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("parse payload file %q as JSON: %w", path, err)
	}

	obj, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("payload file %q must contain a JSON object at the top level", path)
	}

	return obj, nil
}

func ptrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
