package groups

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type createCommand struct {
	file        *string
	displayName *string
	description *string
	role        *string
}

func (c *createCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	group, err := c.buildGroup(ctx)
	if err != nil {
		return err
	}

	request := openapi_client.GroupsCreateGroupRequest{}
	domain := core.OrganizationDomainType()
	request.SetDomainType(domain)
	request.SetDomainId(organizationID)
	request.SetGroup(group)

	response, _, err := ctx.API.GroupsAPI.
		GroupsCreateGroup(ctx.Context).
		Body(request).
		Execute()
	if err != nil {
		return err
	}

	created := response.GetGroup()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(created)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderGroupText(stdout, created)
	})
}

func (c *createCommand) buildGroup(ctx core.CommandContext) (openapi_client.GroupsGroup, error) {
	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}

	if filePath != "" {
		if len(ctx.Args) > 0 {
			return openapi_client.GroupsGroup{}, fmt.Errorf("cannot combine a positional name with --file")
		}
		if ctx.Cmd.Flags().Changed("display-name") ||
			ctx.Cmd.Flags().Changed("description") ||
			ctx.Cmd.Flags().Changed("role") {
			return openapi_client.GroupsGroup{}, fmt.Errorf("cannot combine --display-name, --description, or --role with --file")
		}
		resource, err := parseGroupFile(filePath)
		if err != nil {
			return openapi_client.GroupsGroup{}, err
		}
		if resource.Metadata == nil || resource.Metadata.GetName() == "" {
			return openapi_client.GroupsGroup{}, fmt.Errorf("group metadata.name is required")
		}
		return resourceToGroup(*resource), nil
	}

	if len(ctx.Args) == 0 {
		return openapi_client.GroupsGroup{}, fmt.Errorf("either a group name positional argument or --file is required")
	}
	name := ctx.Args[0]

	metadata := openapi_client.GroupsGroupMetadata{}
	metadata.SetName(name)

	spec := openapi_client.GroupsGroupSpec{}
	if c.displayName != nil && *c.displayName != "" {
		spec.SetDisplayName(*c.displayName)
	}
	if c.description != nil && *c.description != "" {
		spec.SetDescription(*c.description)
	}
	if c.role != nil && *c.role != "" {
		spec.SetRole(*c.role)
	}

	group := openapi_client.GroupsGroup{}
	group.SetMetadata(metadata)
	group.SetSpec(spec)
	return group, nil
}
