package secrets

import (
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type getCommand struct{}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := resolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.SecretAPI.
		SecretsDescribeSecret(ctx.Context, ctx.Args[0]).
		DomainType(string(organizationDomainType())).
		DomainId(organizationID).
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
