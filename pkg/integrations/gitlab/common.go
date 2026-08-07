package gitlab

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

var expressionPlaceholderRegex = regexp.MustCompile(`(?s)\{\{.*?\}\}`)

// isExpression reports whether the given string contains an expression
// placeholder (e.g. `{{ ... }}`). Useful in Setup paths to skip strict
// validation of values that will only be known at execution time.
func isExpression(s string) bool {
	return expressionPlaceholderRegex.MatchString(s)
}

// ensureConcreteProject rejects expression-based project values for components
// that provision a project webhook during setup (the triggers and Run Pipeline).
// Those webhooks must be registered against a concrete GitLab project, so the
// project cannot be deferred to runtime the way it can for components that only
// read it when they execute.
func ensureConcreteProject(project string) error {
	if isExpression(project) {
		return fmt.Errorf("project does not support expressions: this component provisions a project webhook, so it needs a concrete project")
	}
	return nil
}

const (
	PipelineStatusSuccess   = "success"
	PipelineStatusFailed    = "failed"
	PipelineStatusCanceled  = "canceled"
	PipelineStatusCancelled = "cancelled"
	PipelineStatusSkipped   = "skipped"
	PipelineStatusManual    = "manual"
	PipelineStatusBlocked   = "blocked"
)

// zeroSHA is the all-zero commit SHA GitLab sends in push events to signal a
// ref that did not exist before the push (branch creation, where "before" is
// zeroed) or no longer exists after it (branch deletion, where "after" is
// zeroed).
const zeroSHA = "0000000000000000000000000000000000000000"

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

func ensureProjectInMetadata(ctx core.MetadataWriter, app core.IntegrationContext, projectID string) error {
	var nodeMetadata NodeMetadata
	err := mapstructure.Decode(ctx.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	if projectID == "" {
		return fmt.Errorf("project is required")
	}

	//
	// Expression values are only known at execution time, so skip the
	// accessibility check and node metadata caching until then.
	//
	if isExpression(projectID) {
		return nil
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

// matchesNoteContentFilter checks whether a Note Hook payload's comment body
// matches the given regex filter. An empty filter always matches.
func matchesNoteContentFilter(filter string, data map[string]any) (bool, error) {
	if filter == "" {
		return true, nil
	}

	attrs, ok := data["object_attributes"].(map[string]any)
	if !ok {
		return false, nil
	}

	note, ok := attrs["note"].(string)
	if !ok {
		return false, nil
	}

	matched, err := regexp.MatchString(filter, note)
	if err != nil {
		return false, fmt.Errorf("invalid content filter pattern: %w", err)
	}

	return matched, nil
}

// parseUserIDs converts a list of stringified GitLab user IDs (as produced by
// member resource selectors) into ints, skipping any that are not numeric.
func parseUserIDs(ids []string) []int {
	var result []int
	for _, s := range ids {
		var id int
		if _, err := fmt.Sscanf(s, "%d", &id); err == nil {
			result = append(result, id)
		}
	}
	return result
}

// reviewerIDsOf returns the user IDs of a merge request's current reviewers.
func reviewerIDsOf(mr *MergeRequest) []int {
	ids := make([]int, 0, len(mr.Reviewers))
	for _, r := range mr.Reviewers {
		ids = append(ids, r.ID)
	}
	return ids
}

// mergeReviewerIDs returns the union of existing and added IDs, preserving the
// existing order and appending new IDs that are not already present.
func mergeReviewerIDs(existing, add []int) []int {
	seen := make(map[int]struct{}, len(existing))
	result := make([]int, 0, len(existing)+len(add))
	for _, id := range existing {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	for _, id := range add {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

// removeReviewerIDs returns the existing IDs with any of the given IDs removed.
func removeReviewerIDs(existing, remove []int) []int {
	toRemove := make(map[int]struct{}, len(remove))
	for _, id := range remove {
		toRemove[id] = struct{}{}
	}
	result := make([]int, 0, len(existing))
	for _, id := range existing {
		if _, ok := toRemove[id]; ok {
			continue
		}
		result = append(result, id)
	}
	return result
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
