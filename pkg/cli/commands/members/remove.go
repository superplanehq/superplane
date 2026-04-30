package members

import (
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type removeCommand struct {
	email *string
}

func (c *removeCommand) Execute(ctx core.CommandContext) error {
	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	positional := ""
	if len(ctx.Args) > 0 {
		positional = strings.TrimSpace(ctx.Args[0])
	}

	emailFlag := ""
	if c.email != nil {
		emailFlag = strings.TrimSpace(*c.email)
	}

	if positional != "" && emailFlag != "" {
		return fmt.Errorf("pass either a positional user id or --email, not both")
	}

	// Fast path: positional is a UUID. Skip the list-users lookup and go
	// straight to DELETE with the id.
	if positional != "" && !strings.Contains(positional, "@") {
		if _, err := uuid.Parse(positional); err == nil {
			return c.deleteByID(ctx, organizationID, positional, positional)
		}
	}

	user, err := resolveMember(ctx, organizationID, positional, emailFlag)
	if err != nil {
		return err
	}

	metadata := user.GetMetadata()
	return c.deleteByID(ctx, organizationID, metadata.GetId(), metadata.GetEmail())
}

func (c *removeCommand) deleteByID(ctx core.CommandContext, organizationID, userID, displayName string) error {
	response, _, err := ctx.API.OrganizationAPI.
		OrganizationsRemoveUser(ctx.Context, organizationID, userID).
		Execute()
	if err != nil {
		return err
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, err := fmt.Fprintf(stdout, "Member removed: %s\n", displayName)
			return err
		})
	}

	return ctx.Renderer.Render(response)
}
