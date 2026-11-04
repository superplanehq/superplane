package tasks

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

var checkLevels = map[string]openapi_client.WorkOrderCheckLevel{
	"positive": openapi_client.WORKORDERCHECKLEVEL_LEVEL_POSITIVE,
	"caution":  openapi_client.WORKORDERCHECKLEVEL_LEVEL_CAUTION,
	"critical": openapi_client.WORKORDERCHECKLEVEL_LEVEL_CRITICAL,
}

type checkListCommand struct {
	workspace *string
	taskID    *string
}

func (c *checkListCommand) Execute(ctx core.CommandContext) error {
	workspaceID, taskID, err := resolveCheckTask(ctx, c.workspace, c.taskID)
	if err != nil {
		return err
	}
	response, _, err := ctx.API.FactoryAPI.
		FactoriesDescribeWorkOrder(ctx.Context, workspaceID, taskID).
		Execute()
	if err != nil {
		return err
	}
	task := response.GetOrder()
	checks := task.GetChecks()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(checks)
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderChecksText(stdout, checks)
	})
}

type checkAddCommand struct {
	workspace *string
	taskID    *string
	key       *string
	name      *string
	score     *float64
	maxScore  *float64
	level     *string
	reason    *string
}

func (c *checkAddCommand) Execute(ctx core.CommandContext) error {
	level, ok := checkLevels[strings.ToLower(strings.TrimSpace(stringValue(c.level)))]
	if !ok {
		return fmt.Errorf("--level must be positive, caution, or critical")
	}
	workspaceID, taskID, err := resolveCheckTask(ctx, c.workspace, c.taskID)
	if err != nil {
		return err
	}

	check := openapi_client.NewAddWorkOrderChecksRequestCheck()
	check.SetKey(strings.TrimSpace(stringValue(c.key)))
	check.SetName(strings.TrimSpace(stringValue(c.name)))
	check.SetScore(floatValue(c.score))
	check.SetMaxScore(floatValue(c.maxScore))
	check.SetLevel(level)
	check.SetReason(strings.TrimSpace(stringValue(c.reason)))
	body := openapi_client.NewFactoriesAddWorkOrderChecksBody()
	body.SetChecks([]openapi_client.AddWorkOrderChecksRequestCheck{*check})

	response, _, err := ctx.API.FactoryAPI.
		FactoriesAddWorkOrderChecks(ctx.Context, workspaceID, taskID).
		Body(*body).
		Execute()
	if err != nil {
		return err
	}
	checks := response.GetChecks()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(checks)
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderChecksText(stdout, checks)
	})
}

type checkRemoveCommand struct {
	workspace *string
	taskID    *string
	keys      *[]string
}

func (c *checkRemoveCommand) Execute(ctx core.CommandContext) error {
	if c.keys == nil || len(*c.keys) == 0 {
		return fmt.Errorf("--key is required")
	}
	workspaceID, taskID, err := resolveCheckTask(ctx, c.workspace, c.taskID)
	if err != nil {
		return err
	}

	body := openapi_client.NewFactoriesRemoveWorkOrderChecksBody()
	body.SetKeys(*c.keys)
	response, _, err := ctx.API.FactoryAPI.
		FactoriesRemoveWorkOrderChecks(ctx.Context, workspaceID, taskID).
		Body(*body).
		Execute()
	if err != nil {
		return err
	}
	removed := response.GetRemoved()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{"removed": removed})
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(removed) == 0 {
			_, err := fmt.Fprintln(stdout, "No checks removed")
			return err
		}
		_, err := fmt.Fprintf(stdout, "Removed %s.\n", strings.Join(removed, ", "))
		return err
	})
}

func resolveCheckTask(ctx core.CommandContext, workspace, rawTaskID *string) (string, string, error) {
	ref := taskRef(rawTaskID)
	if ref == "" {
		return "", "", errTaskRequired
	}
	workspaceID, err := resolveWorkspace(ctx, workspace)
	if err != nil {
		return "", "", err
	}
	taskID, err := resolveTaskID(ref)
	if err != nil {
		return "", "", err
	}
	return workspaceID, taskID, nil
}

func renderChecksText(stdout io.Writer, checks []openapi_client.FactoriesWorkOrderCheck) error {
	if len(checks) == 0 {
		_, err := fmt.Fprintln(stdout, "No checks")
		return err
	}
	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "KEY\tNAME\tSCORE\tLEVEL\tSOURCE")
	for _, check := range checks {
		score := formatCheckNumber(check.GetScore()) + "/" + formatCheckNumber(check.GetMaxScore())
		level := strings.ToLower(strings.TrimPrefix(string(check.GetLevel()), "LEVEL_"))
		_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", check.GetKey(), check.GetName(), score, level, check.GetSource())
	}
	return writer.Flush()
}

func formatCheckNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func floatValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
