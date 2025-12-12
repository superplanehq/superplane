package github

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v74/github"
	"github.com/superplanehq/superplane/pkg/core"
)

func NewClient(ctx core.AppInstallationContext, ghAppID int64, installationID string) (*github.Client, error) {
	ID, err := strconv.Atoi(installationID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse installation ID: %v", err)
	}

	pem, err := findSecret(ctx, GitHubAppPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to find PEM: %v", err)
	}

	f, err := os.CreateTemp("", "github-app.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}

	defer f.Close()
	defer os.Remove(f.Name())

	_, err = f.Write([]byte(pem))
	if err != nil {
		return nil, fmt.Errorf("failed to write temp file: %v", err)
	}

	itr, err := ghinstallation.NewKeyFromFile(
		http.DefaultTransport,
		ghAppID,
		int64(ID),
		f.Name(),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create apps transport: %v", err)
	}

	return github.NewClient(&http.Client{Transport: itr}), nil
}

func findSecret(ctx core.AppInstallationContext, secretName string) (string, error) {
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
