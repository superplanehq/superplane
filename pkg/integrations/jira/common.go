package jira

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

// NodeMetadata stores metadata on trigger/component nodes.
type NodeMetadata struct {
	Project *Project `json:"project,omitempty"`
}

// ensureProjectInMetadata validates project exists and sets node metadata.
func ensureProjectInMetadata(ctx core.MetadataContext, app core.IntegrationContext, configuration any) error {
	var nodeMetadata NodeMetadata
	err := mapstructure.Decode(ctx.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	project := getProjectFromConfiguration(configuration)
	if project == "" {
		return fmt.Errorf("project is required")
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(app.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	projectIndex := slices.IndexFunc(appMetadata.Projects, func(p Project) bool {
		return p.Key == project
	})

	if projectIndex == -1 {
		return fmt.Errorf("project %s is not accessible", project)
	}

	if nodeMetadata.Project != nil && nodeMetadata.Project.Key == project {
		return nil
	}

	return ctx.Set(NodeMetadata{
		Project: &appMetadata.Projects[projectIndex],
	})
}

// getProjectFromConfiguration extracts project from config map.
func getProjectFromConfiguration(c any) string {
	configMap, ok := c.(map[string]any)
	if !ok {
		return ""
	}

	p, ok := configMap["project"]
	if !ok {
		return ""
	}

	project, ok := p.(string)
	if !ok {
		return ""
	}

	return project
}

// verifyJiraSignature verifies HMAC-SHA256 signature from Jira webhook.
// OAuth dynamic webhooks do not include a signature header, so we skip
// verification for those. The trust model relies on the webhook URL
// being known only to Jira.
func verifyJiraSignature(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Hub-Signature")
	if signature == "" {
		// OAuth dynamic webhooks do not send a signature header.
		return http.StatusOK, nil
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature format")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	err = crypto.VerifySignature(secret, ctx.Body, signature)
	if err != nil {
		return http.StatusForbidden, err
	}

	return http.StatusOK, nil
}

// whitelistedIssueType checks if issue type is in allowed list.
// If issueTypes is empty, all issue types are allowed.
func whitelistedIssueType(data map[string]any, issueTypes []string) bool {
	if len(issueTypes) == 0 {
		return true
	}

	issue, ok := data["issue"].(map[string]any)
	if !ok {
		return false
	}

	fields, ok := issue["fields"].(map[string]any)
	if !ok {
		return false
	}

	issuetype, ok := fields["issuetype"].(map[string]any)
	if !ok {
		return false
	}

	name, ok := issuetype["name"].(string)
	if !ok {
		return false
	}

	return slices.Contains(issueTypes, name)
}
