package gitlab

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	EventType string `json:"eventType" mapstructure:"eventType"`
	ProjectID string `json:"projectId" mapstructure:"projectId"`
}

type WebhookMetadata struct {
	ID int `json:"id" mapstructure:"id"`
}

type NodeMetadata struct {
	Repository *Repository `json:"repository"`
}

// getAuthToken retrieves the appropriate authentication token based on auth type
func getAuthToken(ctx core.IntegrationContext, authType string) (string, error) {
	switch authType {
	case AuthTypePersonalAccessToken:
		tokenBytes, err := ctx.GetConfig("personalAccessToken")
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

// whitelistedAction checks if the action is in the allowed list.
func whitelistedAction(data map[string]any, allowedActions []string) bool {
	objAttrs, ok := data["object_attributes"].(map[string]any)
	if !ok {
		return false
	}

	actionStr, ok := objAttrs["action"].(string)
	if !ok {
		return false
	}

	return slices.Contains(allowedActions, actionStr)
}

func ensureRepoInMetadata(ctx core.MetadataContext, app core.IntegrationContext, projectID string) error {
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

	repoIndex := slices.IndexFunc(appMetadata.Repositories, func(r Repository) bool {
		return fmt.Sprintf("%d", r.ID) == projectID
	})

	if repoIndex == -1 {
		return fmt.Errorf("project %s is not accessible to integration", projectID)
	}

	if nodeMetadata.Repository != nil && fmt.Sprintf("%d", nodeMetadata.Repository.ID) == projectID {
		return nil
	}

	return ctx.Set(NodeMetadata{
		Repository: &appMetadata.Repositories[repoIndex],
	})
}
