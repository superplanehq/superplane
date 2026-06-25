package oidc

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	spoidc "github.com/superplanehq/superplane/pkg/oidc"
)

type verifyCommand struct {
	token        *string
	apiURL       *string
	orgID        *string
	canvasID     *string
	nodeID       *string
	component    *string
	projectID    *string
	pipelineFile *string
	ref          *string
	commitSha    *string
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
		apiURL = "http://localhost:8000"
	}

	claims, err := spoidc.ValidateRemote(http.DefaultClient, token, apiURL)
	if err != nil {
		return fmt.Errorf("token verification failed")
	}

	if err := matchExpectedClaims(claims, c); err != nil {
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

		for _, key := range []string{
			"org_id", "canvas_id", "node_id", "execution_id", "component",
			"project_id", "pipeline_file", "ref", "commit_sha",
		} {
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

func matchExpectedClaims(claims map[string]any, c *verifyCommand) error {
	checks := map[string]string{
		"org_id":        flagValue(c.orgID),
		"canvas_id":     flagValue(c.canvasID),
		"node_id":       flagValue(c.nodeID),
		"component":     flagValue(c.component),
		"project_id":    flagValue(c.projectID),
		"pipeline_file": flagValue(c.pipelineFile),
		"ref":           flagValue(c.ref),
		"commit_sha":    flagValue(c.commitSha),
	}

	for key, expected := range checks {
		if expected == "" {
			continue
		}
		if claimString(claims, key) != expected {
			return fmt.Errorf("token verification failed")
		}
	}

	return nil
}

func flagValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
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
