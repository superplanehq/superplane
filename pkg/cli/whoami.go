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

	organizationLabel := response.GetOrganizationId()
	var canvasSandboxModeEnabled *bool
	if response.HasOrganizationId() && response.GetOrganizationId() != "" {
		orgResponse, _, err := ctx.API.OrganizationAPI.
			OrganizationsDescribeOrganization(ctx.Context, response.GetOrganizationId()).
			Execute()

		if err == nil && orgResponse != nil && orgResponse.Organization != nil && orgResponse.Organization.Metadata != nil {
			metadata := orgResponse.Organization.Metadata
			if metadata.Name != nil && *metadata.Name != "" {
				organizationLabel = *metadata.Name
			}

			if enabled, ok := metadata.GetCanvasSandboxModeEnabledOk(); ok {
				canvasSandboxModeEnabled = enabled
			}
		}
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			sandboxModeLabel := "unknown"
			if canvasSandboxModeEnabled != nil {
				if *canvasSandboxModeEnabled {
					sandboxModeLabel = "enabled"
				} else {
					sandboxModeLabel = "disabled"
				}
			}

			_, _ = fmt.Fprintf(stdout, "ID: %s\n", response.GetId())
			_, _ = fmt.Fprintf(stdout, "Email: %s\n", response.GetEmail())
			_, _ = fmt.Fprintf(stdout, "Organization ID: %s\n", response.GetOrganizationId())
			_, _ = fmt.Fprintf(stdout, "Organization: %s\n", organizationLabel)
			_, _ = fmt.Fprintf(stdout, "Canvas Sandbox Mode: %s\n", sandboxModeLabel)
			return nil
		})
	}

	return ctx.Renderer.Render(map[string]any{
		"id":                       response.GetId(),
		"email":                    response.GetEmail(),
		"organizationId":           response.GetOrganizationId(),
		"organizationName":         organizationLabel,
		"canvasSandboxModeEnabled": canvasSandboxModeEnabled,
	})
}

var whoamiCmd = &cobra.Command{
	Use:     "whoami",
	Short:   "Get information about the currently authenticated user",
	Aliases: []string{"events"},
	Args:    cobra.NoArgs,
}

func init() {
	core.Bind(whoamiCmd, &whoamiCommand{}, defaultBindOptions())
	RootCmd.AddCommand(whoamiCmd)
}
