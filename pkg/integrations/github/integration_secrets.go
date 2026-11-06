package github

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	githubauth "github.com/google/go-github/v75/github"
	"github.com/google/go-github/v84/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	integrationSecretGitHubToken        = "GITHUB_TOKEN"
	integrationSecretGitHubContextToken = "SUPERPLANE_GITHUB_CONTEXT_TOKEN"
	githubSetupName                     = "Set up GitHub"
)

func githubSecretUsage(host common.Host) string {
	return host.SecretUsage()
}

func githubSetupScript(host common.Host) string {
	return host.SetupGitScript()
}

type httpContextTransport struct {
	http core.HTTPContext
}

func (t *httpContextTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.http.Do(req)
}

func resolveAccessToken(httpCtx core.HTTPContext, integrationCtx core.IntegrationContext) (string, error) {
	if integrationCtx.LegacySetup() {
		return legacyAccessToken(httpCtx, integrationCtx)
	}

	authMethod, err := integrationCtx.Properties().GetString(common.PropertyAuthMethod)
	if err != nil {
		return "", fmt.Errorf("failed to get authentication method: %v", err)
	}

	switch authMethod {
	case common.AuthMethodPAT:
		pat, err := integrationCtx.Secrets().Get(common.SecretPAT)
		if err != nil {
			return "", fmt.Errorf("failed to get PAT: %v", err)
		}

		pat = strings.TrimSpace(pat)
		if pat == "" {
			return "", fmt.Errorf("PAT is required")
		}

		return pat, nil

	case common.AuthMethodApp:
		return appInstallationAccessToken(httpCtx, integrationCtx)

	default:
		return "", fmt.Errorf("invalid authentication method: %s", authMethod)
	}
}

func legacyAccessToken(httpCtx core.HTTPContext, integrationCtx core.IntegrationContext) (string, error) {
	var metadata common.Metadata
	if err := mapstructure.Decode(integrationCtx.GetMetadata(), &metadata); err != nil {
		return "", fmt.Errorf("failed to decode metadata: %v", err)
	}

	installationID := 0
	appID := metadata.GitHubApp.ID
	var repositoryIDs []int64
	if metadata.HostedApp {
		hosted, err := common.ResolveHostedAppBinding(integrationCtx)
		if err != nil {
			return "", err
		}
		installationID = int(hosted.ID)
		appID = hosted.App.ID
		repositoryIDs = hosted.RepositoryIDs
	} else {
		var err error
		installationID, err = strconv.Atoi(metadata.InstallationID)
		if err != nil {
			return "", fmt.Errorf("failed to parse installation ID: %v", err)
		}
	}

	client, err := newClientForApp(httpCtx, integrationCtx, appID)
	if err != nil {
		return "", err
	}
	return mintRunnerInstallationToken(client, int64(installationID), repositoryIDs)
}

// resolveContextAccessToken mints a contents:read installation token for
// enabled read-access GitHub context repositories on workspaces that use this
// integration. Write-access context repositories use the write token.
// Names the catalog does not know yet schedule a full installation refresh.
func resolveContextAccessToken(httpCtx core.HTTPContext, integrationCtx core.IntegrationContext) (string, error) {
	if !integrationCtx.LegacySetup() {
		return "", nil
	}
	var metadata common.Metadata
	if err := mapstructure.Decode(integrationCtx.GetMetadata(), &metadata); err != nil || !metadata.HostedApp {
		return "", nil
	}
	hosted, err := common.ResolveHostedAppBinding(integrationCtx)
	if err != nil {
		return "", err
	}
	names, err := models.ListEnabledGitHubContextRepositoryNames(database.Conn(), integrationCtx.ID(), models.FactoryContextRepoAccessRead)
	if err != nil {
		return "", fmt.Errorf("failed to list context repositories: %w", err)
	}
	if len(names) == 0 {
		return "", nil
	}
	repositoryIDs, missing, err := models.MatchVCSProviderRepositoryIDs(
		database.Conn(),
		models.ProviderGitHub,
		hosted.ID,
		names,
	)
	if err != nil {
		return "", fmt.Errorf("failed to match context repositories: %w", err)
	}
	if len(missing) > 0 {
		binding, bindErr := models.FindVCSProviderIntegrationBinding(database.Conn(), integrationCtx.ID())
		if bindErr != nil {
			return "", fmt.Errorf("failed to find GitHub App binding: %w", bindErr)
		}
		enqueueErr := models.EnqueueVCSProviderInstallationReconciliation(
			database.Conn(),
			models.ProviderGitHub,
			hosted.ID,
			binding.OrganizationID,
			time.Now(),
		)
		if enqueueErr != nil && len(repositoryIDs) == 0 {
			return "", fmt.Errorf("failed to refresh GitHub repositories: %w", enqueueErr)
		}
	}
	if len(repositoryIDs) == 0 {
		return "", nil
	}
	slices.Sort(repositoryIDs)
	return mintReadOnlyInstallationToken(httpCtx, hosted.App.ID, hosted.ID, hosted.App.PrivateKey, repositoryIDs)
}

func mintReadOnlyInstallationToken(
	httpCtx core.HTTPContext,
	appID, installationID int64,
	pem string,
	repositoryIDs []int64,
) (string, error) {
	transport, err := ghinstallation.New(
		&httpContextTransport{http: httpCtx},
		appID,
		installationID,
		[]byte(pem),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create apps transport: %v", err)
	}
	common.ApplyHostToInstallationTransport(transport, common.DotCom())
	read := "read"
	transport.InstallationTokenOptions = &githubauth.InstallationTokenOptions{
		RepositoryIDs: repositoryIDs,
		Permissions: &githubauth.InstallationPermissions{
			Contents: &read,
			Metadata: &read,
		},
	}
	token, err := transport.Token(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to create read-only installation access token: %v", err)
	}
	return token, nil
}

func appInstallationAccessToken(
	httpCtx core.HTTPContext,
	integrationCtx core.IntegrationContext,
) (string, error) {
	properties := integrationCtx.Properties()
	appID, err := properties.GetString(common.PropertyAppID)
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub app ID: %v", err)
	}

	appNumber, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse GitHub app ID: %v", err)
	}

	installationID, err := properties.GetString(common.PropertyAppInstallationID)
	if err != nil {
		return "", fmt.Errorf("failed to get installation ID: %v", err)
	}

	installationNumber, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse installation ID: %v", err)
	}

	client, err := newClientForApp(httpCtx, integrationCtx, appNumber)
	if err != nil {
		return "", err
	}
	return mintRunnerInstallationToken(client, installationNumber, nil)
}

// Runner credentials cannot publish checks or commit statuses. Either can
// satisfy a required GitHub gate, so both remain under server control.
func mintRunnerInstallationToken(client *github.Client, installationID int64, repositoryIDs []int64) (string, error) {
	path := fmt.Sprintf("app/installations/%d", installationID)
	request, err := client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	var installation struct {
		Permissions map[string]string `json:"permissions"`
	}
	if _, err := client.Do(context.Background(), request, &installation); err != nil {
		return "", fmt.Errorf("failed to get installation permissions: %w", err)
	}
	if len(installation.Permissions) == 0 {
		return "", fmt.Errorf("GitHub installation did not return permissions")
	}
	for _, permission := range []string{"checks", "statuses"} {
		if installation.Permissions[permission] == "write" {
			installation.Permissions[permission] = "read"
		}
	}
	options := struct {
		RepositoryIDs []int64           `json:"repository_ids,omitempty"`
		Permissions   map[string]string `json:"permissions"`
	}{RepositoryIDs: repositoryIDs, Permissions: installation.Permissions}
	request, err = client.NewRequest(http.MethodPost, path+"/access_tokens", options)
	if err != nil {
		return "", err
	}
	var token github.InstallationToken
	if _, err := client.Do(context.Background(), request, &token); err != nil {
		return "", fmt.Errorf("failed to create runner installation access token: %w", err)
	}
	if token.GetToken() == "" {
		return "", fmt.Errorf("GitHub did not return a runner installation access token")
	}
	return token.GetToken(), nil
}
