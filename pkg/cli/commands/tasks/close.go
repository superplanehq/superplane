package tasks

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type taskCloseCommand struct {
	workspace *string
	taskID    *string
	result    *string
}

func (c *taskCloseCommand) Execute(ctx core.CommandContext) error {
	rawTaskID := strings.TrimSpace(stringValue(c.taskID))
	if rawTaskID == "" {
		return fmt.Errorf("--task is required")
	}

	result, err := closeTaskResult(stringValue(c.result))
	if err != nil {
		return err
	}

	workspaceID, err := resolveWorkspace(ctx, c.workspace)
	if err != nil {
		return err
	}

	taskID, err := resolveTaskID(rawTaskID)
	if err != nil {
		return err
	}

	body := openapi_client.NewFactoriesCloseWorkOrderBody()
	body.SetResult(result)

	response, _, err := ctx.API.FactoryAPI.
		FactoriesCloseWorkOrder(ctx.Context, workspaceID, taskID).
		Body(*body).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		order := response.GetOrder()
		_, err := fmt.Fprintf(stdout, "Closed %s (%s).\n", order.GetKey(), formatTaskResult(order.GetResult()))
		return err
	})
}

func closeTaskResult(raw string) (openapi_client.FactoriesWorkOrderResult, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "completed":
		return openapi_client.FACTORIESWORKORDERRESULT_RESULT_COMPLETED, nil
	case "rejected":
		return openapi_client.FACTORIESWORKORDERRESULT_RESULT_REJECTED, nil
	case "failed":
		return openapi_client.FACTORIESWORKORDERRESULT_RESULT_FAILED, nil
	default:
		return "", fmt.Errorf("--result must be completed, rejected, or failed")
	}
}
