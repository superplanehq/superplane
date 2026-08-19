package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const integrationSecretGitHubToken = "GITHUB_TOKEN"

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

	installationID, err := strconv.Atoi(metadata.InstallationID)
	if err != nil {
		return "", fmt.Errorf("failed to parse installation ID: %v", err)
	}

	pem, err := common.FindSecret(integrationCtx, common.GitHubAppPEM)
	if err != nil {
		return "", fmt.Errorf("failed to find PEM: %v", err)
	}

	itr, err := ghinstallation.New(
		&httpContextTransport{http: httpCtx},
		metadata.GitHubApp.ID,
		int64(installationID),
		[]byte(pem),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create apps transport: %v", err)
	}

	token, err := itr.Token(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to create installation access token: %v", err)
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
		return "", fmt.Errorf("failed to create installation access token: %v", err)
	}

	return token, nil
}
