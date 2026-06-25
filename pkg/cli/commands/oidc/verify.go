package oidc

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

const defaultAPIURL = "https://app.superplane.com"

type verifyCommand struct {
	token          *string
	apiURL         *string
	expectedClaims *[]string
}

func (c *verifyCommand) Execute(ctx core.CommandContext) error {
	token := strings.TrimSpace(*c.token)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("SUPERPLANE_OIDC_TOKEN"))
	}
	if token == "" {
		return fmt.Errorf("token is required (use --token or SUPERPLANE_OIDC_TOKEN)")
	}

	apiURL := strings.TrimRight(strings.TrimSpace(*c.apiURL), "/")
	if apiURL == "" {
		if ctx.Config != nil {
			apiURL = strings.TrimRight(ctx.Config.GetURL(), "/")
		}
	}
	if apiURL == "" {
		apiURL = defaultAPIURL
	}

	claims, err := validateRemote(http.DefaultClient, token, apiURL)
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
