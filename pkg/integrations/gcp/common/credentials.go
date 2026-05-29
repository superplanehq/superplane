package common

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type wifMetadata struct {
	AccessTokenExpiresAt string `json:"accessTokenExpiresAt" mapstructure:"accessTokenExpiresAt"`
}

func accessTokenFromIntegration(ctx core.IntegrationContext) ([]byte, error) {
	if ctx.LegacySetup() {
		secrets, err := ctx.GetSecrets()
		if err != nil {
			return nil, err
		}
		return FindSecretValue(secrets, SecretNameAccessToken), nil
	}
	v, err := ctx.Secrets().Get(SecretNameAccessToken)
	if err != nil {
		if errors.Is(err, core.ErrSecretNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if strings.TrimSpace(v) == "" {
		return nil, nil
	}
	return []byte(v), nil
}

func TokenSourceFromIntegration(ctx core.IntegrationContext) (oauth2.TokenSource, error) {
	accessToken, err := accessTokenFromIntegration(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get integration secrets: %w", err)
	}

	if len(accessToken) == 0 {
		return nil, fmt.Errorf("no GCP credentials found: configure Workload Identity Federation and resync")
	}

	var expiry time.Time
	if meta := ctx.GetMetadata(); meta != nil {
		var wif wifMetadata
		_ = mapstructure.Decode(meta, &wif)
		if expStr := strings.TrimSpace(wif.AccessTokenExpiresAt); expStr != "" {
			if exp, err := time.Parse(time.RFC3339, expStr); err == nil {
				if time.Now().After(exp) {
					return nil, fmt.Errorf("GCP access token expired; please resync the integration")
				}
				expiry = exp
			}
		}
	}

	tok := &oauth2.Token{
		AccessToken: string(accessToken),
		TokenType:   "Bearer",
		Expiry:      expiry,
	}
	return oauth2.StaticTokenSource(tok), nil
}

func CredentialsFromIntegration(ctx core.IntegrationContext) (*google.Credentials, error) {
	ts, err := TokenSourceFromIntegration(ctx)
	if err != nil {
		return nil, err
	}
	return &google.Credentials{TokenSource: ts}, nil
}

func FindSecretValue(secrets []core.IntegrationSecret, name string) []byte {
	for _, s := range secrets {
		if s.Name == name {
			return s.Value
		}
	}
	return nil
}
