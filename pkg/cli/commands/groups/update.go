package groups

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type updateCommand struct {
	file        *string
	displayName *string
	description *string
	role        *string
}

func (c *updateCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	name, group, err := c.buildGroup(ctx)
	if err != nil {
		return err
	}

	body := openapi_client.GroupsUpdateGroupBody{}
	domain := organizationDomainType()
	body.SetDomainType(domain)
	body.SetDomainId(organizationID)
	body.SetGroup(group)

	response, _, err := ctx.API.GroupsAPI.
		GroupsUpdateGroup(ctx.Context, name).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	updated := response.GetGroup()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(updated)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderGroupText(stdout, updated)
	})
}

func (c *updateCommand) buildGroup(ctx core.CommandContext) (string, openapi_client.GroupsGroup, error) {
	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}

	if filePath != "" {
		if len(ctx.Args) > 0 {
			return "", openapi_client.GroupsGroup{}, fmt.Errorf("cannot combine a positional name with --file")
		}
		if ctx.Cmd.Flags().Changed("display-name") ||
			ctx.Cmd.Flags().Changed("description") ||
			ctx.Cmd.Flags().Changed("role") {
			return "", openapi_client.GroupsGroup{}, fmt.Errorf("cannot combine --display-name, --description, or --role with --file")
		}
		resource, err := parseGroupFile(filePath)
		if err != nil {
			return "", openapi_client.GroupsGroup{}, err
		}
		if resource.Metadata == nil || resource.Metadata.GetName() == "" {
			return "", openapi_client.GroupsGroup{}, fmt.Errorf("group metadata.name is required for update")
		}
		return resource.Metadata.GetName(), resourceToGroup(*resource), nil
	}

	if len(ctx.Args) == 0 {
		return "", openapi_client.GroupsGroup{}, fmt.Errorf("a group name positional argument is required (or use --file)")
	}
	name := ctx.Args[0]

	if !ctx.Cmd.Flags().Changed("display-name") &&
		!ctx.Cmd.Flags().Changed("description") &&
		!ctx.Cmd.Flags().Changed("role") {
		return "", openapi_client.GroupsGroup{}, fmt.Errorf("at least one of --display-name, --description, --role, or --file must be provided")
	}

	// Only set the fields the user explicitly provided. The backend merges
	// unspecified fields with the stored values (pkg/grpc/actions/auth/update_group.go).
	spec := openapi_client.GroupsGroupSpec{}
	if ctx.Cmd.Flags().Changed("display-name") {
		spec.SetDisplayName(*c.displayName)
	}
	if ctx.Cmd.Flags().Changed("description") {
		spec.SetDescription(*c.description)
	}
	if ctx.Cmd.Flags().Changed("role") {
		spec.SetRole(*c.role)
	}

	metadata := openapi_client.GroupsGroupMetadata{}
	metadata.SetName(name)

	group := openapi_client.GroupsGroup{}
	group.SetMetadata(metadata)
	group.SetSpec(spec)
	return name, group, nil
}
