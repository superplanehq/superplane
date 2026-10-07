package jira

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const superplaneWebhookPathPrefix = "/api/v1/webhooks/"

var issueWebhookConflictURLPattern = regexp.MustCompile(`(?i)currently used URL:\s*(\S+)`)

func (h *JiraWebhookHandler) createIssueWebhookRecoveringURLConflict(
	client *Client,
	ctx core.WebhookHandlerContext,
	jqlFilter string,
	events []string,
) (int64, error) {
	webhookID, err := client.CreateIssueWebhook(ctx.Webhook.GetURL(), jqlFilter, events)
	if err == nil {
		return webhookID, nil
	}

	if _, isConflict := parseIssueWebhookConflictURL(err); !isConflict {
		return 0, err
	}

	if recoverErr := h.recoverStaleIssueWebhookRegistration(client, ctx, err); recoverErr != nil {
		return 0, recoverErr
	}

	return client.CreateIssueWebhook(ctx.Webhook.GetURL(), jqlFilter, events)
}

func (h *JiraWebhookHandler) recoverStaleIssueWebhookRegistration(
	client *Client,
	ctx core.WebhookHandlerContext,
	createErr error,
) error {
	blockerURL, ok := parseIssueWebhookConflictURL(createErr)
	if !ok {
		return createErr
	}

	if !sameOriginCallbacks(ctx.Webhook.GetURL(), blockerURL) {
		return fmt.Errorf("%w: blocking URL is not a SuperPlane callback on this origin", createErr)
	}

	webhookID, err := superplaneWebhookIDFromCallbackURL(blockerURL)
	if err != nil {
		return fmt.Errorf("%w: blocking URL is not a SuperPlane webhook callback", createErr)
	}

	active, err := ctx.Webhook.CallbackHasActiveNodes(webhookID)
	if err != nil {
		return fmt.Errorf("failed to check whether blocking webhook %s is still in use: %w", webhookID, err)
	}
	if active {
		return fmt.Errorf("%w: blocking webhook %s still has active consumers", createErr, webhookID)
	}

	registered, err := client.ListIssueWebhooks()
	if err != nil {
		return fmt.Errorf("failed to list Jira webhooks after URL conflict: %w", err)
	}

	ids := issueWebhookIDsMatchingURL(registered, blockerURL)
	if len(ids) == 0 {
		return fmt.Errorf("%w: no registered Jira webhook matches the blocking URL", createErr)
	}

	if err := client.DeleteIssueWebhooks(ids); err != nil {
		return fmt.Errorf("failed to delete stale Jira webhook: %w", err)
	}

	h.clearMirroredWebhookIDs(ctx, ids)
	return nil
}

func (h *JiraWebhookHandler) clearMirroredWebhookIDs(ctx core.WebhookHandlerContext, deleted []int64) {
	integrationMetadata := Metadata{}
	_ = mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata)
	if integrationMetadata.WebhookID == nil {
		return
	}
	if !slices.Contains(deleted, *integrationMetadata.WebhookID) {
		return
	}

	integrationMetadata.WebhookID = nil
	ctx.Integration.SetMetadata(integrationMetadata)
}

func parseIssueWebhookConflictURL(err error) (string, bool) {
	if err == nil {
		return "", false
	}

	message := err.Error()
	if !strings.Contains(strings.ToLower(message), "only a single url") {
		return "", false
	}

	match := issueWebhookConflictURLPattern.FindStringSubmatch(message)
	if len(match) < 2 {
		return "", false
	}

	blocker := strings.TrimRight(match[1], ".,;")
	if blocker == "" {
		return "", false
	}
	return blocker, true
}

func sameOriginCallbacks(requestedURL, blockerURL string) bool {
	requested, err := url.Parse(requestedURL)
	if err != nil || requested.Scheme == "" || requested.Host == "" {
		return false
	}
	blocker, err := url.Parse(blockerURL)
	if err != nil || blocker.Scheme == "" || blocker.Host == "" {
		return false
	}
	return strings.EqualFold(requested.Scheme, blocker.Scheme) && strings.EqualFold(requested.Host, blocker.Host)
}

func superplaneWebhookIDFromCallbackURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	path := strings.TrimSuffix(parsed.Path, "/")
	id := strings.TrimPrefix(path, superplaneWebhookPathPrefix)
	if id == path || id == "" || strings.Contains(id, "/") {
		return "", fmt.Errorf("callback path is not a SuperPlane webhook URL")
	}

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return "", fmt.Errorf("callback path is not a SuperPlane webhook URL")
	}
	return parsedID.String(), nil
}

func callbacksEqual(leftURL, rightURL string) bool {
	left, err := url.Parse(leftURL)
	if err != nil {
		return false
	}
	right, err := url.Parse(rightURL)
	if err != nil {
		return false
	}
	return strings.EqualFold(left.Scheme, right.Scheme) &&
		strings.EqualFold(left.Host, right.Host) &&
		strings.TrimSuffix(left.Path, "/") == strings.TrimSuffix(right.Path, "/")
}

func issueWebhookIDsMatchingURL(webhooks []IssueWebhook, blockerURL string) []int64 {
	var ids []int64
	for _, webhook := range webhooks {
		if webhook.URL == "" {
			continue
		}
		if callbacksEqual(webhook.URL, blockerURL) {
			ids = append(ids, webhook.ID)
		}
	}
	return ids
}
