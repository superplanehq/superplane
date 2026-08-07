package secrets

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type deleteCommand struct {
	appID *string
}

func (c *deleteCommand) Execute(ctx core.CommandContext) error {
	domainType, domainID, err := core.ResolveSecretDomain(ctx, appIDFlag(c.appID))
	if err != nil {
		return err
	}

	response, _, err := ctx.API.SecretAPI.
		SecretsDeleteSecret(ctx.Context, ctx.Args[0]).
		DomainType(string(domainType)).
		DomainId(domainID).
		Execute()
	if err != nil {
		return err
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, err := fmt.Fprintf(stdout, "Secret deleted: %s\n", ctx.Args[0])
			return err
		})
	}

	return ctx.Renderer.Render(response)
}
