package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiToken, err := requiredConfig(ctx, "apiToken")
	if err != nil {
		return nil, err
	}

	baseURL, err := optionalConfig(ctx, "baseURL")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}

	return &Client{
		APIToken: apiToken,
		BaseURL:  strings.TrimSuffix(baseURL, "/"),
		http:     httpClient,
	}, nil
}

func requiredConfig(ctx core.IntegrationContext, name string) (string, error) {
	value, err := ctx.GetConfig(name)
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSpace(string(value))
	if trimmed == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	return trimmed, nil
}

func optionalConfig(ctx core.IntegrationContext, name string) (string, error) {
	value, err := ctx.GetConfig(name)
	if err != nil {
		errText := strings.ToLower(err.Error())
		if strings.Contains(errText, "not found") {
			return "", nil
		}

		// Optional fields can be stored as null in configuration.
		if strings.Contains(errText, "not a string") {
			return "", nil
		}

		return "", err
	}

	return strings.TrimSpace(string(value)), nil
}

func (c *Client) ensureAccountID() error {
	if strings.TrimSpace(c.AccountID) != "" {
		return nil
	}

	_, err := c.ResolveAccountID()
	return err
}

func (c *Client) ensureProjectScope() error {
	if err := c.ensureAccountID(); err != nil {
		return err
	}

	if strings.TrimSpace(c.OrgID) == "" {
		return fmt.Errorf("orgId is required")
	}

	if strings.TrimSpace(c.ProjectID) == "" {
		return fmt.Errorf("projectId is required")
	}

	return nil
}

func (c *Client) withScope(orgID, projectID string) *Client {
	cloned := *c
	cloned.OrgID = strings.TrimSpace(orgID)
	cloned.ProjectID = strings.TrimSpace(projectID)
	return &cloned
}

func (c *Client) accountQuery() url.Values {
	query := url.Values{}
	if strings.TrimSpace(c.AccountID) != "" {
		query.Set("accountIdentifier", c.AccountID)
	}
	return query
}

func (c *Client) scopeQuery() url.Values {
	query := c.accountQuery()

	if c.OrgID != "" {
		query.Set("orgIdentifier", c.OrgID)
		if c.ProjectID != "" {
			query.Set("projectIdentifier", c.ProjectID)
		}
	}

	return query
}

func (c *Client) Verify() error {
	if _, err := c.ResolveAccountID(); err != nil {
		return fmt.Errorf("failed to verify api key: %s", summarizeVerificationError(err))
	}

	return c.ensureAccountScope()
}

func (c *Client) ResolveAccountID() (string, error) {
	accountIDFromToken := parseAccountIDFromToken(c.APIToken)
	if strings.TrimSpace(accountIDFromToken) != "" {
		c.AccountID = strings.TrimSpace(accountIDFromToken)
		return c.AccountID, nil
	}

	if strings.TrimSpace(c.AccountID) != "" {
		return c.AccountID, nil
	}

	if c.disableCurrentUserLookup {
		return "", nil
	}

	resolvedFromUser, err := c.resolveAccountIDFromCurrentUser()
	if err != nil {
		if shouldIgnoreCurrentUserLookupError(err) {
			// /currentUser is not reliable for non-USER principals and may return
			// backend errors for service-account keys. Keep AccountID empty and
			// continue with account-scoped probes.
			c.disableCurrentUserLookup = true
			return "", nil
		}
		return "", err
	}

	c.AccountID = strings.TrimSpace(resolvedFromUser)
	if strings.TrimSpace(c.AccountID) == "" {
		return "", fmt.Errorf("unable to resolve account identifier from api key")
	}

	return c.AccountID, nil
}

func shouldIgnoreCurrentUserLookupError(err error) bool {
	apiError := &APIError{}
	if !errors.As(err, &apiError) {
		return false
	}

	if apiError.StatusCode >= http.StatusInternalServerError {
		return true
	}

	body := strings.ToLower(strings.TrimSpace(apiError.Body))
	if body == "" {
		return false
	}

	if strings.Contains(body, "current user can be accessed only by 'user' principal type") {
		return true
	}

	return strings.Contains(body, "principal type")
}

func (c *Client) resolveAccountIDFromCurrentUser() (string, error) {
	_, body, err := c.execRequest(http.MethodGet, "/ng/api/user/currentUser", nil, nil, false)
	if err != nil {
		return "", err
	}

	response := map[string]any{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse current user response: %w", err)
	}

	return firstNonEmpty(
		readStringPath(response, "data", "defaultAccountIdentifier"),
		readStringPath(response, "data", "currentAccountIdentifier"),
		readStringPath(response, "defaultAccountIdentifier"),
		readStringPath(response, "accountIdentifier"),
	), nil
}

func parseAccountIDFromToken(token string) string {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return ""
	}

	parts := strings.Split(trimmed, ".")
	if len(parts) < 2 {
		return ""
	}

	if !strings.EqualFold(parts[0], "pat") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

func (c *Client) ensureAccountScope() error {
	if err := c.ensureAccountID(); err != nil {
		return err
	}

	attempts := []struct {
		endpoint string
		query    url.Values
	}{
		{
			endpoint: "/ng/api/organizations",
			query:    c.accountQuery(),
		},
		{
			endpoint: "/v1/orgs",
			query:    c.accountQuery(),
		},
	}

	var lastErr error
	for _, attempt := range attempts {
		_, _, err := c.execRequest(http.MethodGet, attempt.endpoint, attempt.query, nil, false)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("failed to verify account scope: %s", summarizeVerificationError(lastErr))
}

func summarizeVerificationError(err error) string {
	if err == nil {
		return "unknown error"
	}

	apiError := &APIError{}
	if !errors.As(err, &apiError) {
		return strings.TrimSpace(err.Error())
	}

	message := extractHarnessErrorMessage(apiError.Body)
	if message == "" {
		return fmt.Sprintf("request failed with %d", apiError.StatusCode)
	}

	return fmt.Sprintf("request failed with %d: %s", apiError.StatusCode, message)
}

func extractHarnessErrorMessage(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return ""
	}

	parsed := map[string]any{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return truncateString(trimmed, 300)
	}

	message := firstNonEmpty(
		readString(parsed["message"]),
		readString(parsed["error"]),
		readString(parsed["details"]),
	)
	if message != "" {
		return message
	}

	responseMessages, ok := parsed["responseMessages"].([]any)
	if !ok || len(responseMessages) == 0 {
		return truncateString(trimmed, 300)
	}

	firstResponseMessage, ok := responseMessages[0].(map[string]any)
	if !ok {
		return truncateString(trimmed, 300)
	}

	return firstNonEmpty(
		readString(firstResponseMessage["message"]),
		readString(firstResponseMessage["error"]),
		truncateString(trimmed, 300),
	)
}

func truncateString(value string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}
	if len(value) <= maxLength {
		return value
	}
	return value[:maxLength] + "..."
}
