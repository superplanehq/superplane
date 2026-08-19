package cli

import (
	"errors"
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

	orgID := me.User.GetOrganizationId()
	organizationResponse, _, err := api.OrganizationAPI.
		OrganizationsDescribeOrganization(ctx.Context, orgID).
		Execute()

	if err != nil {
		return fmt.Errorf("failed to describe organization %s: %w", orgID, err)
	}

	orgName := *organizationResponse.Organization.Metadata.Name

	saved, err := UpsertContext(ConfigContext{
		URL:            baseURL,
		Organization:   orgName,
		OrganizationID: orgID,
		APIToken:       apiToken,
	})
	if err != nil {
		return err
	}

	serverVersion, serverVersionErr := ServerVersion()

	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, err := fmt.Fprintf(stdout, "Connected to %q (%s)\n", orgName, saved.URL)
			if err != nil {
				return err
			}
			if err := writeServerVersionInfo(stdout, serverVersion, serverVersionErr); err != nil {
				return err
			}
			_, err = fmt.Fprintf(stdout, "%s\n", core.AgentSkillsHint())
			return err
		})
	}

	result := map[string]any{
		"organization":   orgName,
		"organizationId": orgID,
		"url":            baseURL,
	}
	if serverVersionErr == nil {
		result["serverVersion"] = serverVersion
	}

	return ctx.Renderer.Render(result)
}

func writeServerVersionInfo(stdout io.Writer, serverVersion string, serverVersionErr error) error {
	if errors.Is(serverVersionErr, core.ErrServerVersionUnavailable) {
		_, err := fmt.Fprintf(stdout,
			"Server version: unknown (the server predates version reporting). If commands fail with \"not found\", use a CLI release matching your server.\n")
		return err
	}

	if _, err := fmt.Fprintf(stdout, "Server version: %s\n", serverVersion); err != nil {
		return err
	}

	if !isDevBuild() && serverVersion != "dev" && isNewerVersion(serverVersion, Version) {
		_, err := fmt.Fprintf(stdout,
			"Warning: CLI version %s is newer than server version %s. Commands added after %s may fail with \"not found\"; use a matching CLI release for best results.\n",
			Version, serverVersion, serverVersion)
		return err
	}
	return nil
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
