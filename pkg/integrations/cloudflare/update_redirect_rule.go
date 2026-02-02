package cloudflare

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateRedirectRule struct{}

type UpdateRedirectRuleSpec struct {
	Zone             string `json:"zone"`
	RuleID           string `json:"ruleId"`
	Description      string `json:"description"`
	MatchType        string `json:"matchType"`
	SourceURLPattern string `json:"sourceUrlPattern"`
	Expression       string `json:"expression"`
	TargetURL        string `json:"targetUrl"`
	StatusCode       string `json:"statusCode"`
	PreserveQueryStr bool   `json:"preserveQueryString"`
	Enabled          bool   `json:"enabled"`
}

type UpdateRedirectRuleMetadata struct {
	Zone *Zone `json:"zone"`
}

func (c *UpdateRedirectRule) Name() string {
	return "cloudflare.updateRedirectRule"
}

func (c *UpdateRedirectRule) Label() string {
	return "Update Redirect Rule"
}

func (c *UpdateRedirectRule) Description() string {
	return "Update a redirect rule in a Cloudflare zone"
}

func (c *UpdateRedirectRule) Documentation() string {
	return `The Update Redirect Rule component modifies an existing redirect rule in a Cloudflare zone.

## Use Cases

- **URL management**: Update redirect rules dynamically based on workflow events
- **A/B testing**: Switch redirect targets for testing purposes
- **Maintenance**: Temporarily redirect traffic during maintenance
- **Migration**: Update redirects as part of site migration workflows

## Configuration

- **Zone**: Select the Cloudflare zone containing the redirect rule
- **Rule ID**: The ID of the redirect rule to update
- **Description**: Optional description for the rule
- **Match Type**: How to match URLs (exact match or expression-based)
- **Source URL Pattern**: URL pattern to match (for exact match type)
- **Expression**: Cloudflare expression for matching (for expression type)
- **Target URL**: The URL to redirect to (supports expressions)
- **Status Code**: HTTP status code for redirect (301, 302, 307, 308)
- **Preserve Query String**: Whether to preserve query parameters in redirect
- **Enabled**: Whether the rule is active

## Output

Returns the updated redirect rule with all current configuration.`
}

func (c *UpdateRedirectRule) Icon() string {
	return "cloud"
}

func (c *UpdateRedirectRule) Color() string {
	return "orange"
}

func (c *UpdateRedirectRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateRedirectRule) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "zone",
			Label:       "Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloudflare zone containing the redirect rule",
			Placeholder: "Select a zone",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "zone",
				},
			},
		},
		{
			Name:        "ruleId",
			Label:       "Rule ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the redirect rule to update",
		},
		{
			Name:        "description",
			Label:       "Rule Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "A descriptive name for this redirect rule",
		},
		{
			Name:     "matchType",
			Label:    "Match Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "wildcard",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Wildcard Pattern", Value: "wildcard"},
						{Label: "Custom Filter Expression", Value: "expression"},
					},
				},
			},
		},
		{
			Name:        "sourceUrlPattern",
			Label:       "Source URL Pattern",
			Type:        configuration.FieldTypeString,
			Description: "URL pattern with wildcards. Use * to match any path segment. Example: https://example.com/old/*",
			Placeholder: "https://example.com/old/*",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "matchType", Values: []string{"wildcard"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "matchType", Values: []string{"wildcard"}},
			},
		},
		{
			Name:        "expression",
			Label:       "Match Expression",
			Type:        configuration.FieldTypeText,
			Description: "Cloudflare filter expression. Example: (http.host eq \"example.com\" and http.request.uri.path eq \"/old-path\")",
			Placeholder: "(http.host eq \"example.com\")",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "matchType", Values: []string{"expression"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "matchType", Values: []string{"expression"}},
			},
		},
		{
			Name:        "targetUrl",
			Label:       "Target URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The URL to redirect to. For wildcard patterns, use ${1}, ${2}, etc. to reference captured groups.",
			Placeholder: "https://example.com/new/${1}",
		},
		{
			Name:     "statusCode",
			Label:    "Status Code",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "301",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "301 - Permanent Redirect", Value: "301"},
						{Label: "302 - Temporary Redirect", Value: "302"},
						{Label: "307 - Temporary Redirect (preserve method)", Value: "307"},
						{Label: "308 - Permanent Redirect (preserve method)", Value: "308"},
					},
				},
			},
		},
		{
			Name:        "preserveQueryString",
			Label:       "Preserve Query String",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Whether to preserve the query string when redirecting",
		},
		{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Whether the redirect rule is enabled",
		},
	}
}

func (c *UpdateRedirectRule) Setup(ctx core.SetupContext) error {
	spec := UpdateRedirectRuleSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Zone == "" {
		return errors.New("zone is required")
	}

	if spec.RuleID == "" {
		return errors.New("ruleId is required")
	}

	if spec.MatchType == "wildcard" {
		if spec.SourceURLPattern == "" {
			return errors.New("sourceUrlPattern is required for wildcard match type")
		}
	} else if spec.MatchType == "expression" {
		if spec.Expression == "" {
			return errors.New("expression is required for expression match type")
		}
	} else if spec.MatchType == "" {
		return errors.New("matchType is required")
	}

	if spec.TargetURL == "" {
		return errors.New("targetUrl is required")
	}

	return nil
}

func (c *UpdateRedirectRule) Execute(ctx core.ExecutionContext) error {
	spec := UpdateRedirectRuleSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Get the ruleset for the redirect phase
	ruleset, err := client.GetRulesetForPhase(spec.Zone, "http_request_dynamic_redirect")
	if err != nil {
		return fmt.Errorf("error getting ruleset: %v", err)
	}

	// Parse status code
	statusCode, err := strconv.Atoi(spec.StatusCode)
	if err != nil {
		return fmt.Errorf("invalid status code: %v", err)
	}

	// Build the expression and target URL based on match type
	var expression string
	var targetURL *RedirectTargetURL

	if spec.MatchType == "wildcard" {
		// For wildcard patterns, use the 'wildcard' operator with r"" string syntax
		// Format from actual Cloudflare API: (http.request.full_uri wildcard r"pattern*")
		expression = fmt.Sprintf(`(http.request.full_uri wildcard r"%s")`, spec.SourceURLPattern)

		// Check if target URL contains ${1}, ${2}, etc. placeholders
		placeholderRegex := regexp.MustCompile(`\$\{(\d+)\}`)
		if placeholderRegex.MatchString(spec.TargetURL) {
			// Use wildcard_replace() for dynamic redirects
			// Format: wildcard_replace(http.request.full_uri, r"source", r"target_with_${1}")
			targetExpr := fmt.Sprintf(`wildcard_replace(http.request.full_uri, r"%s", r"%s")`, spec.SourceURLPattern, spec.TargetURL)
			targetURL = &RedirectTargetURL{Expression: targetExpr}
		} else {
			// Static URL, use value field
			targetURL = &RedirectTargetURL{Value: spec.TargetURL}
		}
	} else {
		expression = spec.Expression
		targetURL = &RedirectTargetURL{Value: spec.TargetURL}
	}

	// Prepare the update request
	updateReq := UpdateRedirectRuleRequest{
		Action:      "redirect",
		Expression:  expression,
		Description: spec.Description,
		Enabled:     spec.Enabled,
		ActionParam: &RedirectActionData{
			FromValue: &RedirectFromValue{
				StatusCode:       statusCode,
				PreserveQueryStr: spec.PreserveQueryStr,
				TargetURL:        targetURL,
			},
		},
	}

	// Update the redirect rule
	rule, err := client.UpdateRedirectRule(spec.Zone, ruleset.ID, spec.RuleID, updateReq)
	if err != nil {
		return fmt.Errorf("failed to update redirect rule: %v", err)
	}

	result := map[string]any{
		"rule":    rule,
		"zoneId":  spec.Zone,
		"ruleId":  spec.RuleID,
		"enabled": spec.Enabled,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.redirectRule",
		[]any{result},
	)
}

func (c *UpdateRedirectRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateRedirectRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateRedirectRule) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateRedirectRule) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateRedirectRule) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *UpdateRedirectRule) Cleanup(ctx core.SetupContext) error {
	return nil
}
