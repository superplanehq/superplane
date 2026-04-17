package members

import (
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type getCommand struct {
	email *string
}

func (c *getCommand) Execute(ctx core.CommandContext) error {
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

	user, err := resolveMember(ctx, organizationID, positional, emailFlag)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(user)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderMemberText(stdout, user)
	})
}
