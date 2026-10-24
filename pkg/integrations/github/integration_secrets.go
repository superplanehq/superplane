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

const githubSecretUsage = `A GitHub token is available in the GITHUB_TOKEN environment variable.
The gh CLI is already installed on this runner. Use gh. Do not download or install gh.
gh reads GITHUB_TOKEN.
Use normal HTTPS repository URLs such as https://github.com/<owner>/<repo>.git.
Do not put the token in URLs, commands, Git configuration, or output.`

const githubSetupScript = `set -euo pipefail
: "${GITHUB_TOKEN:?GITHUB_TOKEN is required}"
gh auth setup-git --hostname github.com --force
`

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
		return appInstallationAccessToken(httpCtx, integrationCtx.Properties(), integrationCtx.Secrets())

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
	pem := ""
	var repositoryIDs []int64
	if metadata.HostedApp {
		hosted, err := common.ResolveHostedAppBinding(integrationCtx)
		if err != nil {
			return "", err
		}
		installationID = int(hosted.ID)
		appID = hosted.App.ID
		pem = hosted.App.PrivateKey
		repositoryIDs = hosted.RepositoryIDs
	} else {
		var err error
		installationID, err = strconv.Atoi(metadata.InstallationID)
		if err != nil {
			return "", fmt.Errorf("failed to parse installation ID: %v", err)
		}
		pem, err = common.LegacyAppPrivateKey(integrationCtx, metadata)
		if err != nil {
			return "", fmt.Errorf("failed to find PEM: %v", err)
		}
	}

	itr, err := ghinstallation.New(
		&httpContextTransport{http: httpCtx},
		appID,
		int64(installationID),
		[]byte(pem),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create apps transport: %v", err)
	}
	if metadata.HostedApp {
		if err := common.RestrictHostedAppTransport(itr, repositoryIDs); err != nil {
			return "", err
		}
	}

	token, err := itr.Token(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to create installation access token: %w", common.ExplainError(err))
	}

	return token, nil
}

// resolveContextAccessToken mints a contents:read installation token for
// enabled GitHub context repositories on workspaces that use this integration.
// The write token stays limited to the workspace and backlog repositories.
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
	names, err := models.ListEnabledGitHubContextRepositoryNames(database.Conn(), integrationCtx.ID())
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
	properties core.IntegrationPropertyStorageReader,
	secrets core.IntegrationSecretStorageReader,
) (string, error) {
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

	pem, err := secrets.Get(common.SecretAppPEM)
	if err != nil {
		return "", fmt.Errorf("failed to get PEM: %v", err)
	}

	itr, err := ghinstallation.New(
		&httpContextTransport{http: httpCtx},
		appNumber,
		installationNumber,
		[]byte(pem),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create apps transport: %v", err)
	}

	token, err := itr.Token(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to create installation access token: %w", common.ExplainError(err))
	}

	return token, nil
}
