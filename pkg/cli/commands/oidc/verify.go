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

const defaultAPIURL = "https://app.superplane.com"

var (
	errTokenNotFound    = errors.New("token is required (use --token or SUPERPLANE_OIDC_TOKEN)")
	errAudienceRequired = errors.New("audience is required (use --audience)")
)

type verifyCommand struct {
	token          *string
	apiURL         *string
	audience       *string
	expectedClaims *map[string]string

	client      *http.Client
	runToken    string
	runAPIURL   string
	runAudience string
	issuer      string
}

func (c *verifyCommand) Execute(ctx core.CommandContext) error {
	var err error

	c.client = http.DefaultClient
	c.runAPIURL = c.lookupAPIURL(ctx)

	err = c.lookupToken()
	if err != nil {
		return err
	}

	err = c.lookupAudience()
	if err != nil {
		return err
	}

	err = c.parseExpectedClaims()
	if err != nil {
		return err
	}

	err = c.verifyToken(ctx.Context)
	if err != nil {
		return err
	}

	fmt.Println("Token verified")
	return nil
}

func (c *verifyCommand) verifyToken(ctx context.Context) error {
	ctx = gooidc.ClientContext(ctx, c.client)

	issuer, err := c.fetchIssuer(ctx, c.runAPIURL)
	if err != nil {
		return err
	}

	provider, err := gooidc.NewProvider(ctx, issuer)
	if err != nil {
		return err
	}

	verifier := provider.Verifier(&gooidc.Config{
		ClientID: c.runAudience,
	})

	idToken, err := verifier.Verify(ctx, c.runToken)
	if err != nil {
		return err
	}

	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return err
	}

	for key, want := range *c.expectedClaims {
		if claimString(claims, key) != want {
			return fmt.Errorf("token verification failed: expected claim %s to be %s, got %s", key, want, claimString(claims, key))
		}
	}

	return nil
}

func (c *verifyCommand) fetchIssuer(ctx context.Context, baseURL string) (string, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		baseURL+"/.well-known/openid-configuration",
		nil,
	)
	if err != nil {
		return "", err
	}

	response, err := c.client.Do(request)
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

func (c *verifyCommand) parseExpectedClaims() error {
	expected := make(map[string]string, len(*c.expectedClaims))

	for _, claim := range *c.expectedClaims {
		claim = strings.TrimSpace(claim)
		if claim == "" {
			continue
		}

		key, value, ok := strings.Cut(claim, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return fmt.Errorf("invalid claim %q (expected key=value)", claim)
		}

		expected[key] = value
	}

	c.expectedClaims = &expected

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

func (c *verifyCommand) lookupToken() error {
	token := strings.TrimSpace(*c.token)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("SUPERPLANE_OIDC_TOKEN"))
	}
	if token == "" {
		return errTokenNotFound
	}

	c.runToken = token

	return nil
}

func (c *verifyCommand) lookupAudience() error {
	audience := strings.TrimSpace(*c.audience)
	if audience == "" {
		return errAudienceRequired
	}

	c.runAudience = audience

	return nil
}

func (c *verifyCommand) lookupAPIURL(ctx core.CommandContext) string {
	apiURL := strings.TrimRight(strings.TrimSpace(*c.apiURL), "/")
	if apiURL == "" && ctx.Config != nil {
		apiURL = strings.TrimRight(ctx.Config.GetURL(), "/")
	}
	if apiURL == "" {
		apiURL = defaultAPIURL
	}

	return apiURL
}
