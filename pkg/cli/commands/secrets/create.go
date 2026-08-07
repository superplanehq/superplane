package secrets

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type createCommand struct {
	file  *string
	appID *string
}

func (c *createCommand) Execute(ctx core.CommandContext) error {
	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}
	if filePath == "" {
		return fmt.Errorf("--file is required")
	}

	domainType, domainID, err := core.ResolveSecretDomain(ctx, appIDFlag(c.appID))
	if err != nil {
		return err
	}

	resource, err := parseSecretInput(filePath, ctx.Cmd.InOrStdin())
	if err != nil {
		return err
	}

	secret := resourceToSecret(*resource)

	request := openapi_client.SecretsCreateSecretRequest{}
	request.SetSecret(secret)
	request.SetDomainType(domainType)
	request.SetDomainId(domainID)

	response, _, err := ctx.API.SecretAPI.SecretsCreateSecret(ctx.Context).Body(request).Execute()
	if err != nil {
		return err
	}

	createdSecret := response.GetSecret()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(createdSecret)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderSecretText(stdout, createdSecret)
	})
}
