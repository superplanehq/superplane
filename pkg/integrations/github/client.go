package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v74/github"
	"github.com/superplanehq/superplane/pkg/core"
	"golang.org/x/oauth2"
)

func NewClientV2(parameters core.IntegrationParameterStorage, secrets core.IntegrationSecretStorage) (*github.Client, error) {
	authMethod, err := getStringParameter(parameters, ParameterAuthMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication method: %v", err)
	}

	switch authMethod {
	case AuthMethodPAT:
		return NewPATClient(parameters, secrets)
	case AuthMethodGitHubApp:
		return NewGitHubAppClient(parameters, secrets)
	default:
		return nil, fmt.Errorf("invalid authentication method: %s", authMethod)
	}
}

func NewPATClient(parameters core.IntegrationParameterStorage, secrets core.IntegrationSecretStorage) (*github.Client, error) {
	pat, err := secrets.Get(SecretPAT)
	if err != nil {
		return nil, fmt.Errorf("failed to get PAT: %v", err)
	}

	pat = strings.TrimSpace(string(pat))
	if pat == "" {
		return nil, fmt.Errorf("PAT is required")
	}

	return github.NewClient(
		oauth2.NewClient(
			context.Background(),
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: string(pat)}),
		),
	), nil
}

func NewGitHubAppClient(parameters core.IntegrationParameterStorage, secrets core.IntegrationSecretStorage) (*github.Client, error) {
	appID, err := getStringParameter(parameters, ParameterGitHubAppID)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub app ID: %v", err)
	}

	appNumber, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub app ID: %v", err)
	}

	installationID, err := getStringParameter(parameters, ParameterGitHubAppInstallationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get installation ID: %v", err)
	}

	installationNumber, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse installation ID: %v", err)
	}

	pem, err := secrets.Get(SecretGitHubAppPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to get PEM: %v", err)
	}

	itr, err := ghinstallation.New(
		http.DefaultTransport,
		appNumber,
		installationNumber,
		[]byte(pem),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create apps transport: %v", err)
	}

	return github.NewClient(&http.Client{Transport: itr}), nil
}

func NewClient(ctx core.IntegrationContext, ghAppID int64, installationID string) (*github.Client, error) {
	ID, err := strconv.Atoi(installationID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse installation ID: %v", err)
	}

	pem, err := findSecret(ctx, GitHubAppPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to find PEM: %v", err)
	}

	itr, err := ghinstallation.New(
		http.DefaultTransport,
		ghAppID,
		int64(ID),
		[]byte(pem),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create apps transport: %v", err)
	}

	return github.NewClient(&http.Client{Transport: itr}), nil
}

func findSecret(ctx core.IntegrationContext, secretName string) (string, error) {
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, secret := range secrets {
		if secret.Name == secretName {
			return string(secret.Value), nil
		}
	}

	return "", fmt.Errorf("secret %s not found", secretName)
}
