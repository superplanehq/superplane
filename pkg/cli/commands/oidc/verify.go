package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

const (
	defaultAPIURL          = "https://app.superplane.com"
	executionTokenAudience = "semaphore"
	errTokenNotFound       = errors.New("token is required (use --token or SUPERPLANE_OIDC_TOKEN)")
)

type verifyCommand struct {
	token          *string
	apiURL         *string
	expectedClaims *[]string
}

func (c *verifyCommand) Execute(ctx core.CommandContext) error {
	apiURL := c.lookupAPIURL(ctx)

	token, err := c.lookupToken(ctx)
	if err != nil {
		return err
	}

	claims, err := validateRemote(ctx.Context, http.DefaultClient, token, apiURL)
	if err != nil {
		return fmt.Errorf("token verification failed")
	}

	expected, err := parseExpectedClaims(*c.expectedClaims)
	if err != nil {
		return err
	}

	if err := matchExpectedClaims(claims, expected); err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(claims)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Token verified\n")
		if err != nil {
			return err
		}

		for key := range claims {
			value := claimString(claims, key)
			if value == "" {
				continue
			}
			_, err = fmt.Fprintf(stdout, "%s: %s\n", key, value)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

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

func parseExpectedClaims(flags []string) (map[string]string, error) {
	expected := make(map[string]string, len(flags))

	for _, flag := range flags {
		flag = strings.TrimSpace(flag)
		if flag == "" {
			continue
		}

		key, value, ok := strings.Cut(flag, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid claim %q (expected key=value)", flag)
		}

		expected[key] = value
	}

	return expected, nil
}

func matchExpectedClaims(claims map[string]any, expected map[string]string) error {
	for key, want := range expected {
		if claimString(claims, key) != want {
			return fmt.Errorf("token verification failed")
		}
	}

	return nil
}

func claimString(claims map[string]any, key string) string {
	value, ok := claims[key]
	if !ok || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func (c *verifyCommand) lookupToken(ctx core.CommandContext) (string, error) {
	token := strings.TrimSpace(*c.token)

	if token == "" {
		token = strings.TrimSpace(os.Getenv("SUPERPLANE_OIDC_TOKEN"))
	}

	if token == "" {
		return "", errTokenNotFound
	}

	return token, nil
}

func (c *verifyCommand) lookupAPIURL(ctx core.CommandContext) string {
	apiURL := strings.TrimRight(strings.TrimSpace(*c.apiURL), "/")

	if apiURL == "" {
		if ctx.Config != nil {
			apiURL = strings.TrimRight(ctx.Config.GetURL(), "/")
		}
	}

	if apiURL == "" {
		apiURL = defaultAPIURL
	}

	return apiURL
}
