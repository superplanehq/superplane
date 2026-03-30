package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type ConnectCommand struct{}

func (c *ConnectCommand) Execute(ctx core.CommandContext) error {
	baseURL := normalizeBaseURL(ctx.Args[0])
	apiToken := strings.TrimSpace(ctx.Args[1])
	if baseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	if apiToken == "" {
		return fmt.Errorf("API token is required")
	}

	api := NewAPIClient(&ClientConfig{BaseURL: baseURL, APIToken: apiToken})

	me, _, err := api.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return fmt.Errorf("failed to authenticate with the provided token: %w", err)
	}

	organizationResponse, _, err := api.OrganizationAPI.
		OrganizationsDescribeOrganization(ctx.Context, me.GetOrganizationId()).
		Execute()

	if err != nil {
		return fmt.Errorf("failed to describe organization %s: %w", me.GetOrganizationId(), err)
	}

	orgName := *organizationResponse.Organization.Metadata.Name

	_, err = UpsertContext(ConfigContext{
		URL:          baseURL,
		Organization: orgName,
		APIToken:     apiToken,
	})

	if err != nil {
		return err
	}

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, err := fmt.Fprintf(stdout, "Connected to %q (%s)\n", orgName, baseURL)
			return err
		})
	}

	return ctx.Renderer.Render(map[string]any{
		"organization": orgName,
		"url":          baseURL,
	})
}

var connectCmd = &cobra.Command{
	Use:   "connect [BASE_URL] [API_TOKEN]",
	Short: "Connect to a SuperPlane organization",
	Long:  "Validates the provided API token and saves the organization as the current CLI context.",
	Args:  cobra.ExactArgs(2),
}

func init() {
	core.Bind(connectCmd, &ConnectCommand{}, defaultBindOptions())
	RootCmd.AddCommand(connectCmd)
}
