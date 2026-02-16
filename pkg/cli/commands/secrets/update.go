package secrets

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type updateCommand struct {
	file *string
}

func (c *updateCommand) Execute(ctx core.CommandContext) error {
	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}
	if filePath == "" {
		return fmt.Errorf("--file is required")
	}
	if len(ctx.Args) > 0 {
		return fmt.Errorf("update does not accept positional arguments")
	}

	organizationID, err := resolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	resource, err := parseSecretFile(filePath)
	if err != nil {
		return err
	}
	if resource.Metadata == nil || resource.Metadata.GetId() == "" {
		return fmt.Errorf("secret metadata.id is required for update")
	}

	secret := resourceToSecret(*resource)

	request := openapi_client.SecretsUpdateSecretBody{}
	request.SetSecret(secret)
	request.SetDomainType(organizationDomainType())
	request.SetDomainId(organizationID)

	response, _, err := ctx.API.SecretAPI.SecretsUpdateSecret(ctx.Context, resource.Metadata.GetId()).Body(request).Execute()
	if err != nil {
		return err
	}

	updatedSecret := response.GetSecret()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(updatedSecret)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderSecretText(stdout, updatedSecret)
	})
}
