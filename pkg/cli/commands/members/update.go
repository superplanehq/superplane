package members

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type updateCommand struct {
	email *string
	role  *string
}

func (c *updateCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	positional := ""
	if len(ctx.Args) > 0 {
		positional = ctx.Args[0]
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

	body := openapi_client.RolesAssignRoleBody{}
	domain := organizationDomainType()
	body.SetDomainType(domain)
	body.SetDomainId(organizationID)
	if userID != "" {
		body.SetUserId(userID)
	} else {
		body.SetUserEmail(userEmail)
	}

	_, _, err = ctx.API.RolesAPI.
		RolesAssignRole(ctx.Context, *c.role).
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
			_, err := fmt.Fprintf(stdout, "Role %q assigned to %s\n", *c.role, who)
			return err
		})
	}

	return ctx.Renderer.Render(map[string]string{
		"user":     who,
		"roleName": *c.role,
	})
}
