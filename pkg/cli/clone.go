package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type CloneCommand struct{}

func (c *CloneCommand) Execute(ctx core.CommandContext) error {
	orgName, appName, err := parseOrgAppTarget(ctx.Args[0])
	if err != nil {
		return err
	}

	urlFlag, err := ctx.Cmd.Flags().GetString("url")
	if err != nil {
		return err
	}

	api, apiToken, err := apiClientAndTokenForClone(orgName, urlFlag)
	if err != nil {
		return err
	}

	cloneCtx := ctx
	cloneCtx.API = api

	canvasID, err := common.FindAppID(cloneCtx, api, appName)
	if err != nil {
		return err
	}

	repoURL, err := repositoryCloneURL(cloneCtx, canvasID)
	if err != nil {
		return err
	}

	return runGitClone(cloneCtx.Context, repoURL, apiToken, cloneRepoDir(appName, ctx.Args[1:]))
}

func parseOrgAppTarget(target string) (orgName string, appName string, err error) {
	target = strings.TrimSpace(target)
	orgName, appName, ok := strings.Cut(target, "/")
	if !ok {
		return "", "", fmt.Errorf("expected org-name/app-name, got %q", target)
	}

	orgName = strings.TrimSpace(orgName)
	appName = strings.TrimSpace(appName)
	if orgName == "" || appName == "" {
		return "", "", fmt.Errorf("expected org-name/app-name, got %q", target)
	}

	return orgName, appName, nil
}

func apiClientAndTokenForClone(orgName, urlFilter string) (*openapi_client.APIClient, string, error) {
	if env, ok := GetEnvironmentContext(); ok {
		client := NewAPIClient(&ClientConfig{
			BaseURL:  env.URL,
			APIToken: env.APIToken,
		})
		if err := ensureOrganizationMatches(context.Background(), client, orgName); err != nil {
			return nil, "", err
		}
		return client, env.APIToken, nil
	}

	cfg, err := FindConfigContextForOrganization(orgName, urlFilter)
	if err != nil {
		return nil, "", err
	}

	client := NewAPIClient(&ClientConfig{
		BaseURL:        cfg.URL,
		APIToken:       cfg.APIToken,
		OrganizationID: cfg.OrganizationID,
	})
	return client, cfg.APIToken, nil
}

func ensureOrganizationMatches(ctx context.Context, api *openapi_client.APIClient, orgName string) error {
	me, _, err := api.MeAPI.MeMe(ctx).Execute()
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	orgID := strings.TrimSpace(me.User.GetOrganizationId())
	if orgID == "" {
		return fmt.Errorf("organization id not found for authenticated user")
	}

	if strings.EqualFold(orgID, orgName) {
		return nil
	}

	response, _, err := api.OrganizationAPI.OrganizationsDescribeOrganization(ctx, orgID).Execute()
	if err != nil {
		return fmt.Errorf("failed to describe organization: %w", err)
	}

	if response.Organization == nil || response.Organization.Metadata == nil {
		return fmt.Errorf("organization metadata not found")
	}

	name := strings.TrimSpace(response.Organization.Metadata.GetName())
	if !strings.EqualFold(name, orgName) {
		return fmt.Errorf("credentials are for organization %q, not %q", name, orgName)
	}

	return nil
}

func repositoryCloneURL(ctx core.CommandContext, canvasID string) (string, error) {
	response, _, err := ctx.API.CanvasRepositoryAPI.
		CanvasesGetCanvasRepository(ctx.Context, canvasID).
		Execute()
	if err != nil {
		return "", err
	}

	repository, ok := response.GetRepositoryOk()
	if !ok || repository == nil {
		return "", fmt.Errorf("repository not found for app %q", canvasID)
	}

	metadata, ok := repository.GetMetadataOk()
	if !ok || metadata == nil {
		return "", fmt.Errorf("repository metadata not found for app %q", canvasID)
	}

	repoURL := strings.TrimSpace(metadata.GetUrl())
	if repoURL == "" {
		return "", fmt.Errorf("repository has no clone URL for app %q", canvasID)
	}

	return repoURL, nil
}

func runGitClone(ctx context.Context, repoURL, apiToken, repoDir string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}

	apiToken = strings.TrimSpace(apiToken)
	if apiToken == "" {
		return fmt.Errorf("no API token configured; run superplane connect or set %s", EnvToken)
	}

	authHeader := "Authorization: Bearer " + apiToken

	args := []string{
		"clone",
		repoURL,
		"-c",
		"http.extraHeader=" + authHeader,
		repoDir,
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return configureRepositoryHTTPAuth(ctx, repoDir, authHeader)
}

func configureRepositoryHTTPAuth(ctx context.Context, repoDir, authHeader string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "config", "--local", "http.extraHeader", authHeader)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure repository authentication: %w", err)
	}

	return nil
}

func cloneRepoDir(appName string, extraArgs []string) string {
	if len(extraArgs) > 0 {
		return extraArgs[len(extraArgs)-1]
	}
	return appName
}

var cloneCmd = &cobra.Command{
	Use:   "clone [org-name/app-name] [directory]",
	Short: "Clone an app's git repository",
	Long: `Clone the git repository for an app using the SuperPlane API.

The target must be org-name/app-name. Authentication uses the API token from
the saved context for that organization, or from SUPERPLANE_URL and SUPERPLANE_TOKEN.

By default the repository is cloned into a directory named after the app.
Git authentication for fetch and push is stored in the repository's local config.

Example:
  superplane clone acme/widget-app
  superplane clone acme/widget-app ./custom-dir`,
	Args: cobra.MinimumNArgs(1),
}

func init() {
	cloneCmd.Flags().String("url", "", "installation base URL when multiple saved contexts match the organization")
	core.Bind(cloneCmd, &CloneCommand{}, defaultBindOptions())
	RootCmd.AddCommand(cloneCmd)
}
