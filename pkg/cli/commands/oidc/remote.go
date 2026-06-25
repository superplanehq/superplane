package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
)

const executionTokenAudience = "semaphore"

func validateRemote(ctx context.Context, client *http.Client, token, baseURL string) (map[string]any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if client != nil {
		ctx = gooidc.ClientContext(ctx, client)
	}

	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	issuer, err := fetchIssuer(ctx, client, baseURL)
	if err != nil {
		return nil, err
	}

	provider, err := gooidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	verifier := provider.Verifier(&gooidc.Config{
		ClientID: executionTokenAudience,
	})

	idToken, err := verifier.Verify(ctx, token)
	if err != nil {
		return nil, err
	}

	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}

	return claims, nil
}

func fetchIssuer(ctx context.Context, client *http.Client, baseURL string) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		baseURL+"/.well-known/openid-configuration",
		nil,
	)
	if err != nil {
		return "", err
	}

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("fetch OIDC discovery document: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch OIDC discovery document: unexpected status %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("read OIDC discovery document: %w", err)
	}

	var document struct {
		Issuer string `json:"issuer"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		return "", fmt.Errorf("parse OIDC discovery document: %w", err)
	}

	issuer := strings.TrimSpace(document.Issuer)
	if issuer == "" {
		return "", fmt.Errorf("OIDC discovery document is missing issuer")
	}

	return issuer, nil
}
