package secrets

import (
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type listCommand struct {
	appID *string
}

func (c *listCommand) Execute(ctx core.CommandContext) error {
	domainType, domainID, err := core.ResolveSecretDomain(ctx, appIDFlag(c.appID))
	if err != nil {
		return err
	}

	response, _, err := ctx.API.SecretAPI.
		SecretsListSecrets(ctx.Context).
		DomainType(string(domainType)).
		DomainId(domainID).
		Execute()
	if err != nil {
		return err
	}

	secrets := response.GetSecrets()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(secrets)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderSecretListText(stdout, secrets)
	})
}
