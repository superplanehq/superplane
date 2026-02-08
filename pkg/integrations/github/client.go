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

func NewTokenClient(accessToken string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	httpClient := oauth2.NewClient(context.Background(), ts)
	return github.NewClient(httpClient)
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

func findSecretOptional(ctx core.IntegrationContext, secretName string) (string, bool, error) {
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return "", false, err
	}

	for _, secret := range secrets {
		if secret.Name == secretName {
			v := strings.TrimSpace(string(secret.Value))
			if v == "" {
				return "", false, nil
			}
			return v, true, nil
		}
	}

	return "", false, nil
}
