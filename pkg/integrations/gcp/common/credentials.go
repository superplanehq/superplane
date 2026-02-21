package common

import (
	"context"
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

func TokenSourceFromIntegration(ctx core.IntegrationContext, scopes ...string) (oauth2.TokenSource, error) {
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to get integration secrets: %w", err)
	}

	authMethod := AuthMethodFromMetadata(ctx.GetMetadata())

	keyJSON := FindSecretValue(secrets, SecretNameServiceAccountKey)
	if authMethod != AuthMethodWIF && len(keyJSON) > 0 {
		if len(scopes) == 0 {
			scopes = []string{ScopeCloudPlatform}
		}
		creds, err := google.CredentialsFromJSONWithType(context.Background(), keyJSON, google.ServiceAccount, scopes...)
		if err != nil {
			return nil, fmt.Errorf("failed to create credentials from service account key: %w", err)
		}
		return creds.TokenSource, nil
	}

	accessToken := FindSecretValue(secrets, SecretNameAccessToken)
	if authMethod != AuthMethodWIF || len(accessToken) == 0 {
		return nil, fmt.Errorf("no GCP credentials found: add a service account key or use Workload Identity Federation and resync")
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

func CredentialsFromIntegration(ctx core.IntegrationContext, scopes ...string) (*google.Credentials, error) {
	ts, err := TokenSourceFromIntegration(ctx, scopes...)
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

func AuthMethodFromMetadata(meta any) string {
	if meta == nil {
		return AuthMethodServiceAccountKey
	}
	var m struct {
		AuthMethod string `mapstructure:"authMethod"`
	}
	_ = mapstructure.Decode(meta, &m)
	switch m.AuthMethod {
	case AuthMethodWIF:
		return AuthMethodWIF
	default:
		return AuthMethodServiceAccountKey
	}
}
