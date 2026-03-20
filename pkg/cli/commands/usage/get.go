package usage

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const unlimitedValue = "-1"

type getCommand struct{}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := resolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.OrganizationAPI.
		OrganizationsDescribeUsage(ctx.Context, organizationID).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderUsageText(stdout, response)
	})
}

func resolveOrganizationID(ctx core.CommandContext) (string, error) {
	me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return "", err
	}

	if !me.HasOrganizationId() || strings.TrimSpace(me.GetOrganizationId()) == "" {
		return "", fmt.Errorf("organization id not found for authenticated user")
	}

	return me.GetOrganizationId(), nil
}

func renderUsageText(stdout io.Writer, response *openapi_client.OrganizationsDescribeUsageResponse) error {
	if response == nil {
		return fmt.Errorf("usage response was empty")
	}

	_, _ = fmt.Fprintf(stdout, "Enabled: %t\n", response.GetEnabled())
	if response.HasStatusMessage() && strings.TrimSpace(response.GetStatusMessage()) != "" {
		_, _ = fmt.Fprintf(stdout, "Status: %s\n", response.GetStatusMessage())
	}

	if !response.GetEnabled() {
		return nil
	}

	if response.HasUsage() {
		usage := response.GetUsage()
		_, _ = fmt.Fprintln(stdout, "Usage:")

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintf(writer, "Canvases\t%s\n", formatInt32Value(usage.GetCanvasesOk()))
		_, _ = fmt.Fprintf(
			writer,
			"Event bucket\t%s / %s\n",
			formatFloat64Value(usage.GetEventBucketLevelOk()),
			formatFloat64Value(usage.GetEventBucketCapacityOk()),
		)
		if updatedAt, ok := usage.GetEventBucketLastUpdatedAtOk(); ok {
			_, _ = fmt.Fprintf(writer, "Event bucket updated\t%s\n", updatedAt.Format(time.RFC3339))
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	}

	if response.HasLimits() {
		limits := response.GetLimits()
		_, _ = fmt.Fprintln(stdout, "Limits:")

		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintf(writer, "Max canvases\t%s\n", formatInt32Limit(limits.GetMaxCanvasesOk()))
		_, _ = fmt.Fprintf(writer, "Max nodes per canvas\t%s\n", formatInt32Limit(limits.GetMaxNodesPerCanvasOk()))
		_, _ = fmt.Fprintf(writer, "Max users\t%s\n", formatInt32Limit(limits.GetMaxUsersOk()))
		_, _ = fmt.Fprintf(writer, "Retention window days\t%s\n", formatInt32Limit(limits.GetRetentionWindowDaysOk()))
		_, _ = fmt.Fprintf(writer, "Max events per month\t%s\n", formatStringLimit(limits.GetMaxEventsPerMonthOk()))
		_, _ = fmt.Fprintf(writer, "Max integrations\t%s\n", formatInt32Limit(limits.GetMaxIntegrationsOk()))
		return writer.Flush()
	}

	return nil
}

func formatInt32Value(value *int32, ok bool) string {
	if !ok {
		return "n/a"
	}

	return strconv.FormatInt(int64(*value), 10)
}

func formatInt32Limit(value *int32, ok bool) string {
	if !ok {
		return "n/a"
	}

	if *value == -1 {
		return "unlimited"
	}

	return strconv.FormatInt(int64(*value), 10)
}

func formatStringLimit(value *string, ok bool) string {
	if !ok || strings.TrimSpace(*value) == "" {
		return "n/a"
	}

	if strings.TrimSpace(*value) == unlimitedValue {
		return "unlimited"
	}

	return *value
}

func formatFloat64Value(value *float64, ok bool) string {
	if !ok {
		return "n/a"
	}

	if *value == -1 {
		return "unlimited"
	}

	if math.Mod(*value, 1) == 0 {
		return strconv.FormatInt(int64(*value), 10)
	}

	return strconv.FormatFloat(*value, 'f', 2, 64)
}
