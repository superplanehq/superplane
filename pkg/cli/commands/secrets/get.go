package secrets

import (
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type getCommand struct {
	appID *string
}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	domainType, domainID, err := core.ResolveSecretDomain(ctx, appIDFlag(c.appID))
	if err != nil {
		return err
	}

	response, _, err := ctx.API.SecretAPI.
		SecretsDescribeSecret(ctx.Context, ctx.Args[0]).
		DomainType(string(domainType)).
		DomainId(domainID).
		Execute()
	if err != nil {
		return err
	}

	secret := response.GetSecret()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(secret)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderSecretText(stdout, secret)
	})
}
