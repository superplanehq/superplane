package factories

import (
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type artifactListCommand struct{}

func (c *artifactListCommand) Execute(ctx core.CommandContext) error {
	factoryArg := ctx.Args[0]
	orderID := strings.TrimSpace(ctx.Args[1])

	if _, err := uuid.Parse(orderID); err != nil {
		return fmt.Errorf("order-id must be a UUID")
	}

	factoryID, err := FindFactoryID(ctx, factoryArg)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.FactoryAPI.
		FactoriesListWorkOrderArtifacts(ctx.Context, factoryID, orderID).
		Execute()
	if err != nil {
		return err
	}

	artifacts := response.GetArtifacts()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(artifacts)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if len(artifacts) == 0 {
			_, err := fmt.Fprintln(stdout, "No artifacts")
			return err
		}
		for _, artifact := range artifacts {
			title := ""
			if data := artifact.GetData(); data != nil {
				if value, ok := data["title"].(string); ok {
					title = value
				} else if value, ok := data["name"].(string); ok {
					title = value
				} else if value, ok := data["url"].(string); ok {
					title = value
				}
			}
			line := fmt.Sprintf("%s\t%s", artifact.GetId(), artifact.GetType())
			if title != "" {
				line += "\t" + title
			}
			if _, err := fmt.Fprintln(stdout, line); err != nil {
				return err
			}
		}
		return nil
	})
}
