package factories

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type orderDispatchCommand struct {
	factory *string
	orderID *string
	line    *string
}

func (c *orderDispatchCommand) Execute(ctx core.CommandContext) error {
	orderID := strings.TrimSpace(stringValue(c.orderID))
	if orderID == "" {
		return fmt.Errorf("--order is required")
	}

	lineName := strings.TrimSpace(stringValue(c.line))
	if lineName == "" {
		return fmt.Errorf("--line is required")
	}

	factoryID, err := ResolveFactoryID(ctx, stringValue(c.factory))
	if err != nil {
		return err
	}

	body := openapi_client.NewFactoriesDispatchWorkOrderBody()
	body.SetLineName(lineName)

	response, _, err := ctx.API.FactoryAPI.
		FactoriesDispatchWorkOrder(ctx.Context, factoryID, orderID).
		Body(*body).
		Execute()
	if err != nil {
		return err
	}

	order := response.GetOrder()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(order)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(
			stdout,
			"Work order dispatched: %s -> line %q (state: %s)\n",
			order.GetId(),
			lineName,
			formatOrderState(order.GetState()),
		)
		return err
	})
}
