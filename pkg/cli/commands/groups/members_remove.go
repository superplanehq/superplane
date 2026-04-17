package groups

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type membersRemoveCommand struct {
	email *string
}

func (c *membersRemoveCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	groupName := ctx.Args[0]

	positional := ""
	if len(ctx.Args) > 1 {
		positional = ctx.Args[1]
	}

	emailFlag := ""
	if c.email != nil {
		emailFlag = *c.email
	}

	userID, userEmail, err := splitUserIdentifier(positional, emailFlag)
	if err != nil {
		return err
	}
	if userID == "" && userEmail == "" {
		return fmt.Errorf("a user id positional argument or --email is required")
	}

	body := openapi_client.GroupsRemoveUserFromGroupBody{}
	domain := organizationDomainType()
	body.SetDomainType(domain)
	body.SetDomainId(organizationID)
	if userID != "" {
		body.SetUserId(userID)
	} else {
		body.SetUserEmail(userEmail)
	}

	response, _, err := ctx.API.GroupsAPI.
		GroupsRemoveUserFromGroup(ctx.Context, groupName).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	who := userID
	if who == "" {
		who = userEmail
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, err := fmt.Fprintf(stdout, "Removed %s from group %s\n", who, groupName)
			return err
		})
	}

	return ctx.Renderer.Render(response)
}
