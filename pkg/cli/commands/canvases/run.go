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
	trigger     *string
	template    *string
	payloadFile *string
	replay      *string
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

	payloadPath := strings.TrimSpace(ptrVal(c.payloadFile))
	if replay != "" && payloadPath != "" {
		return fmt.Errorf("cannot use --payload-file with --replay")
	}

	canvasID, err := findCanvasID(ctx, ctx.API, ctx.Args[0])
	if err != nil {
		return err
	}

	if replay != "" {
		return c.executeReplay(ctx, canvasID, trigger, replay)
	}

	return c.executeManualHook(ctx, canvasID, trigger, templateName, payloadPath)
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
	canvasID, triggerNodeID, templateName, payloadPath string,
) error {
	parameters := map[string]interface{}{
		"template": templateName,
	}

	if payloadPath != "" {
		payload, err := readJSONObjectFile(payloadPath)
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

func readJSONObjectFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read payload file: %w", err)
	}

	var decoded interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("parse payload file as JSON: %w", err)
	}

	obj, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("payload file must contain a JSON object at the top level")
	}

	return obj, nil
}

func ptrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
