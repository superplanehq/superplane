package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type whoamiCommand struct{}

func (w *whoamiCommand) Execute(ctx core.CommandContext) error {
	response, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return err
	}

	organizationLabel := response.User.GetOrganizationId()
	var versioningEnabled *bool
	if response.User.HasOrganizationId() && response.User.GetOrganizationId() != "" {
		orgResponse, _, err := ctx.API.OrganizationAPI.
			OrganizationsDescribeOrganization(ctx.Context, response.User.GetOrganizationId()).
			Execute()

		if err == nil && orgResponse != nil && orgResponse.Organization != nil && orgResponse.Organization.Metadata != nil {
			metadata := orgResponse.Organization.Metadata
			if metadata.Name != nil && *metadata.Name != "" {
				organizationLabel = *metadata.Name
			}

			if enabled, ok := metadata.GetVersioningEnabledOk(); ok {
				versioningEnabled = enabled
			}
		}
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			versioningLabel := "unknown"
			if versioningEnabled != nil {
				if *versioningEnabled {
					versioningLabel = "enabled"
				} else {
					versioningLabel = "disabled"
				}
			}

			_, _ = fmt.Fprintf(stdout, "ID: %s\n", response.User.GetId())
			_, _ = fmt.Fprintf(stdout, "Email: %s\n", response.User.GetEmail())
			_, _ = fmt.Fprintf(stdout, "Organization ID: %s\n", response.User.GetOrganizationId())
			_, _ = fmt.Fprintf(stdout, "Organization: %s\n", organizationLabel)
			_, _ = fmt.Fprintf(stdout, "Canvas Versioning: %s\n", versioningLabel)
			return nil
		})
	}

	return ctx.Renderer.Render(map[string]any{
		"id":                response.User.GetId(),
		"email":             response.User.GetEmail(),
		"organizationId":    response.User.GetOrganizationId(),
		"organizationName":  organizationLabel,
		"versioningEnabled": versioningEnabled,
	})
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Get information about the currently authenticated user",
	Args:  cobra.NoArgs,
}

func init() {
	core.Bind(whoamiCmd, &whoamiCommand{}, defaultBindOptions())
	RootCmd.AddCommand(whoamiCmd)
}
