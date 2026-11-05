package sentry

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// IssueAPIURL builds the issue endpoint without making an API request.
func IssueAPIURL(integration core.IntegrationContext, issueID string) string {
	if integration == nil {
		return ""
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil || metadata.Organization == nil {
		return ""
	}

	baseURL := DefaultBaseURL
	if !metadata.HostedApp {
		configured, err := integration.GetConfig("baseUrl")
		if err != nil {
			return ""
		}
		baseURL = normalizeBaseURL(string(configured))
	}

	return issueAPIURL(baseURL, metadata.Organization.Slug, issueID)
}

func (c *Client) IssueAPIURL(issueID string) string {
	if c == nil {
		return ""
	}
	return issueAPIURL(c.baseURL, c.orgSlug, issueID)
}

func issueAPIURL(baseURL, organizationSlug, issueID string) string {
	if strings.TrimSpace(organizationSlug) == "" || strings.TrimSpace(issueID) == "" {
		return ""
	}
	return fmt.Sprintf("%s/api/0/organizations/%s/issues/%s/",
		strings.TrimRight(baseURL, "/"), url.PathEscape(organizationSlug), url.PathEscape(issueID))
}
