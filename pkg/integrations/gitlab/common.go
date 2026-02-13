package gitlab

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	PipelineStatusSuccess   = "success"
	PipelineStatusFailed    = "failed"
	PipelineStatusCanceled  = "canceled"
	PipelineStatusCancelled = "cancelled"
	PipelineStatusSkipped   = "skipped"
	PipelineStatusManual    = "manual"
	PipelineStatusBlocked   = "blocked"
)

type WebhookConfiguration struct {
	EventType string `json:"eventType" mapstructure:"eventType"`
	ProjectID string `json:"projectId" mapstructure:"projectId"`
}

type WebhookMetadata struct {
	ID int `json:"id" mapstructure:"id"`
}

type NodeMetadata struct {
	Project *ProjectMetadata `json:"project"`
}

// getAuthToken retrieves the appropriate authentication token based on auth type
func getAuthToken(ctx core.IntegrationContext, authType string) (string, error) {
	switch authType {
	case AuthTypePersonalAccessToken:
		tokenBytes, err := ctx.GetConfig("accessToken")
		if err != nil {
			return "", err
		}
		token := string(tokenBytes)
		if token == "" {
			return "", fmt.Errorf("personal access token not found")
		}
		return token, nil

	case AuthTypeAppOAuth:
		token, err := findSecret(ctx, OAuthAccessToken)
		if err != nil {
			return "", err
		}
		if token == "" {
			return "", fmt.Errorf("OAuth access token not found")
		}
		return token, nil

	default:
		return "", fmt.Errorf("unknown auth type: %s", authType)
	}
}

func setAuthHeaders(req *http.Request, authType, token string) {
	if authType == AuthTypePersonalAccessToken {
		req.Header.Set("PRIVATE-TOKEN", token)
	} else {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

// verifyWebhookToken verifies the X-Gitlab-Token header matches the expected secret.
func verifyWebhookToken(ctx core.WebhookRequestContext) (int, error) {
	token := ctx.Headers.Get("X-Gitlab-Token")
	if token == "" {
		return http.StatusForbidden, fmt.Errorf("missing X-Gitlab-Token header")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting webhook secret: %v", err)
	}

	if subtle.ConstantTimeCompare([]byte(token), secret) != 1 {
		return http.StatusForbidden, fmt.Errorf("invalid webhook token")
	}

	return http.StatusOK, nil
}

func ensureProjectInMetadata(ctx core.MetadataContext, app core.IntegrationContext, projectID string) error {
	var nodeMetadata NodeMetadata
	err := mapstructure.Decode(ctx.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	if projectID == "" {
		return fmt.Errorf("project is required")
	}

	//
	// Validate that the app has access to this repository
	//
	var appMetadata Metadata
	if err := mapstructure.Decode(app.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	repoIndex := slices.IndexFunc(appMetadata.Projects, func(r ProjectMetadata) bool {
		return fmt.Sprintf("%d", r.ID) == projectID
	})

	if repoIndex == -1 {
		return fmt.Errorf("project %s is not accessible to integration", projectID)
	}

	if nodeMetadata.Project != nil && fmt.Sprintf("%d", nodeMetadata.Project.ID) == projectID {
		return nil
	}

	return ctx.Set(NodeMetadata{
		Project: &appMetadata.Projects[repoIndex],
	})
}

func normalizePipelineRef(ref string) string {
	if strings.HasPrefix(ref, "refs/heads/") {
		return strings.TrimPrefix(ref, "refs/heads/")
	}

	if strings.HasPrefix(ref, "refs/tags/") {
		return strings.TrimPrefix(ref, "refs/tags/")
	}

	return ref
}
