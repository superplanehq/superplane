package cloudflare

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	originRulePhase       = "http_request_origin"
	originRuleMatchAll    = "all"
	originRuleMatchCustom = "custom"
)

type OriginRuleNodeMetadata struct {
	Zone        string   `json:"zone,omitempty"`
	ZoneName    string   `json:"zoneName,omitempty"`
	Rule        string   `json:"rule,omitempty"`
	Description string   `json:"description,omitempty"`
	MatchMode   string   `json:"matchMode,omitempty"`
	Expression  string   `json:"expression,omitempty"`
	OriginHost  string   `json:"originHost,omitempty"`
	OriginPort  *int     `json:"originPort,omitempty"`
	HostHeader  string   `json:"hostHeader,omitempty"`
	SNI         string   `json:"sni,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"`
	Rewrites    []string `json:"rewrites,omitempty"`
}

type OriginRuleMatchRule struct {
	Field       string `json:"field" mapstructure:"field"`
	Operator    string `json:"operator" mapstructure:"operator"`
	Value       string `json:"value" mapstructure:"value"`
	Conjunction string `json:"conjunction" mapstructure:"conjunction"`
}

func originRuleConfigurationFields(withoutZone bool) []configuration.Field {
	fields := []configuration.Field{}
	if !withoutZone {
		fields = append(fields, configuration.Field{
			Name:        "zone",
			Label:       "Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloudflare zone where the origin rule should run",
			Placeholder: "Select a zone",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "zone",
				},
			},
		})
	}

	return append(fields,
		configuration.Field{
			Name:        "description",
			Label:       "Rule Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "A descriptive name for this origin rule",
		},
		configuration.Field{
			Name:        "matchMode",
			Label:       "Apply To",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     originRuleMatchCustom,
			Description: "Choose whether to apply the rule to all requests or only matching requests",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Custom Filter Expression", Value: originRuleMatchCustom, Description: "Only apply the rule to requests matching the filter"},
						{Label: "All Incoming Requests", Value: originRuleMatchAll, Description: "Apply the rule to all requests"},
					},
				},
			},
		},
		configuration.Field{
			Name:        "matchRules",
			Label:       "When Incoming Requests Match",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Default:     []map[string]any{{"field": "fullUri", "operator": "wildcard", "value": "/*", "conjunction": "and"}},
			Description: "Builds the Cloudflare filter expression from request field predicates",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "matchMode", Values: []string{originRuleMatchCustom}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "matchMode", Values: []string{originRuleMatchCustom}},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Condition",
					ItemDefinition: &configuration.ListItemDefinition{
						Type:   configuration.FieldTypeObject,
						Schema: originRuleMatchRuleSchema(),
					},
				},
			},
		},
		configuration.Field{
			Name:        "originHost",
			Label:       "DNS Record",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Override the DNS record Cloudflare resolves and routes matching requests to",
			Placeholder: "origin.example.com",
		},
		configuration.Field{
			Name:        "hostHeader",
			Label:       "Host Header",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Rewrite the HTTP Host header sent to the origin",
			Placeholder: "app.example.com",
		},
		configuration.Field{
			Name:        "sni",
			Label:       "Server Name Indication (SNI)",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Rewrite the SNI value used for TLS to the origin",
			Placeholder: "tls.example.com",
		},
		configuration.Field{
			Name:        "originPort",
			Label:       "Destination Port",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Rewrite the destination port for matching requests",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 1; return &min }(),
					Max: func() *int { max := 65535; return &max }(),
				},
			},
		},
	)
}

func originRuleMatchRuleSchema() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "field",
			Label:    "Field",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "fullUri",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "URL Full", Value: "fullUri"},
						{Label: "URI Path", Value: "uriPath"},
						{Label: "Hostname", Value: "host"},
						{Label: "URI Query", Value: "query"},
						{Label: "HTTP Method", Value: "method"},
						{Label: "Scheme", Value: "scheme"},
					},
				},
			},
		},
		{
			Name:     "operator",
			Label:    "Operator",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "wildcard",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Wildcard", Value: "wildcard"},
						{Label: "Equals", Value: "equals"},
						{Label: "Not Equals", Value: "notEquals"},
						{Label: "Contains", Value: "contains"},
						{Label: "Starts With", Value: "startsWith"},
						{Label: "Ends With", Value: "endsWith"},
						{Label: "Matches Regex", Value: "matches"},
					},
				},
			},
		},
		{
			Name:        "value",
			Label:       "Value",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Value to compare with the selected request field",
			Placeholder: "/*",
		},
		{
			Name:     "conjunction",
			Label:    "Next Condition",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  "and",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "And", Value: "and"},
						{Label: "Or", Value: "or"},
					},
				},
			},
		},
	}
}

func validateOriginRuleFields(matchMode string, matchRules []OriginRuleMatchRule, expression string, originHost *string, originPort *int, hostHeader, sni *string) (string, error) {
	resolvedExpression, err := buildOriginExpression(matchMode, matchRules, expression)
	if err != nil {
		return "", err
	}

	if originHost == nil && originPort == nil && hostHeader == nil && sni == nil {
		return "", errors.New("at least one origin parameter must be rewritten")
	}

	if originHost != nil && strings.TrimSpace(*originHost) == "" {
		return "", errors.New("originHost must not be empty when enabled")
	}

	if hostHeader != nil && strings.TrimSpace(*hostHeader) == "" {
		return "", errors.New("hostHeader must not be empty when enabled")
	}

	if sni != nil && strings.TrimSpace(*sni) == "" {
		return "", errors.New("sni must not be empty when enabled")
	}

	if originPort != nil && (*originPort < 1 || *originPort > 65535) {
		return "", errors.New("originPort must be between 1 and 65535")
	}

	return resolvedExpression, nil
}

func validateOriginRuleUpdateFields(spec UpdateOriginRuleSpec) (string, error) {
	hasMatchUpdate := strings.TrimSpace(spec.Expression) != "" || spec.MatchMode != "" || len(spec.MatchRules) > 0
	hasOriginUpdate := spec.OriginHost != nil || spec.OriginPort != nil || spec.HostHeader != nil || spec.SNI != nil
	hasUpdate := spec.Description != nil || hasMatchUpdate || hasOriginUpdate || spec.Enabled != nil
	if !hasUpdate {
		return "", errors.New("at least one field must be enabled for update")
	}

	var expression string
	if hasMatchUpdate {
		resolvedExpression, err := buildOriginExpression(spec.MatchMode, spec.MatchRules, spec.Expression)
		if err != nil {
			return "", err
		}
		expression = resolvedExpression
	}

	if spec.OriginHost != nil && strings.TrimSpace(*spec.OriginHost) == "" {
		return "", errors.New("originHost must not be empty when enabled")
	}

	if spec.HostHeader != nil && strings.TrimSpace(*spec.HostHeader) == "" {
		return "", errors.New("hostHeader must not be empty when enabled")
	}

	if spec.SNI != nil && strings.TrimSpace(*spec.SNI) == "" {
		return "", errors.New("sni must not be empty when enabled")
	}

	if spec.OriginPort != nil && (*spec.OriginPort < 1 || *spec.OriginPort > 65535) {
		return "", errors.New("originPort must be between 1 and 65535")
	}

	return expression, nil
}

func buildOriginExpression(matchMode string, matchRules []OriginRuleMatchRule, expression string) (string, error) {
	if strings.TrimSpace(expression) != "" {
		return strings.TrimSpace(expression), nil
	}

	if matchMode == "" {
		matchMode = originRuleMatchCustom
	}

	switch matchMode {
	case originRuleMatchAll:
		return "true", nil

	case originRuleMatchCustom:
		if len(matchRules) == 0 {
			return "", errors.New("matchRules is required for custom match mode")
		}

		parts := make([]string, 0, len(matchRules))
		for i, rule := range matchRules {
			part, err := buildOriginRulePredicate(rule)
			if err != nil {
				return "", fmt.Errorf("matchRules[%d]: %w", i, err)
			}

			parts = append(parts, part)
			if i < len(matchRules)-1 {
				conjunction := strings.ToLower(strings.TrimSpace(rule.Conjunction))
				if conjunction == "" {
					conjunction = "and"
				}
				if conjunction != "and" && conjunction != "or" {
					return "", fmt.Errorf("matchRules[%d]: conjunction must be either 'and' or 'or'", i)
				}

				parts = append(parts, conjunction)
			}
		}

		return fmt.Sprintf("(%s)", strings.Join(parts, " ")), nil

	default:
		return "", errors.New("matchMode must be custom or all")
	}
}

func buildOriginRulePredicate(rule OriginRuleMatchRule) (string, error) {
	field, err := originRuleExpressionField(rule.Field)
	if err != nil {
		return "", err
	}

	operator := strings.TrimSpace(rule.Operator)
	if operator == "" {
		operator = "wildcard"
	}

	value := strings.TrimSpace(rule.Value)
	if value == "" {
		return "", errors.New("value is required")
	}

	quoted := quoteCloudflareString(value)
	rawQuoted := quoteCloudflareRawString(value)
	switch operator {
	case "equals":
		return fmt.Sprintf(`%s eq %s`, field, quoted), nil
	case "notEquals":
		return fmt.Sprintf(`%s ne %s`, field, quoted), nil
	case "contains":
		return fmt.Sprintf(`%s contains %s`, field, quoted), nil
	case "startsWith":
		return fmt.Sprintf(`starts_with(%s, %s)`, field, quoted), nil
	case "endsWith":
		return fmt.Sprintf(`ends_with(%s, %s)`, field, quoted), nil
	case "matches":
		return fmt.Sprintf(`%s matches %s`, field, quoted), nil
	case "wildcard":
		return fmt.Sprintf(`%s wildcard %s`, field, rawQuoted), nil
	default:
		return "", errors.New("operator must be one of wildcard, equals, notEquals, contains, startsWith, endsWith, or matches")
	}
}

func originRuleExpressionField(field string) (string, error) {
	switch field {
	case "fullUri":
		return "http.request.full_uri", nil
	case "uriPath":
		return "http.request.uri.path", nil
	case "host":
		return "http.host", nil
	case "query":
		return "http.request.uri.query", nil
	case "method":
		return "http.request.method", nil
	case "scheme":
		return "http.request.scheme", nil
	default:
		return "", errors.New("field must be one of fullUri, uriPath, host, query, method, or scheme")
	}
}

func quoteCloudflareString(value string) string {
	return strconv.Quote(value)
}

func quoteCloudflareRawString(value string) string {
	return fmt.Sprintf(`r"%s"`, strings.ReplaceAll(value, `"`, `\"`))
}

func buildOriginRule(description, expression string, originHost *string, originPort *int, hostHeader, sni *string, enabled bool) OriginRule {
	actionParam := &OriginActionParameters{}
	if originHost != nil || originPort != nil {
		actionParam.Origin = &RouteOrigin{Port: originPort}
		if originHost != nil {
			actionParam.Origin.Host = strings.TrimSpace(*originHost)
		}
	}

	if hostHeader != nil {
		actionParam.HostHeader = strings.TrimSpace(*hostHeader)
	}

	if sni != nil {
		actionParam.SNI = &RouteSNIValue{Value: strings.TrimSpace(*sni)}
	}

	return OriginRule{
		Action:      "route",
		Expression:  strings.TrimSpace(expression),
		Description: strings.TrimSpace(description),
		Enabled:     enabled,
		ActionParam: actionParam,
	}
}

func buildOriginRuleUpdateRequest(spec UpdateOriginRuleSpec, existing *OriginRule) (OriginRule, error) {
	expression := existing.Expression
	if strings.TrimSpace(spec.Expression) != "" || spec.MatchMode != "" || len(spec.MatchRules) > 0 {
		resolvedExpression, err := buildOriginExpression(spec.MatchMode, spec.MatchRules, spec.Expression)
		if err != nil {
			return OriginRule{}, err
		}
		expression = resolvedExpression
	}

	description := existing.Description
	if spec.Description != nil {
		description = strings.TrimSpace(*spec.Description)
	}

	enabled := existing.Enabled
	if spec.Enabled != nil {
		enabled = *spec.Enabled
	}

	actionParam := cloneOriginActionParameters(existing.ActionParam)
	if spec.OriginHost != nil || spec.OriginPort != nil {
		if actionParam.Origin == nil {
			actionParam.Origin = &RouteOrigin{}
		}
		if spec.OriginHost != nil {
			actionParam.Origin.Host = strings.TrimSpace(*spec.OriginHost)
		}
		if spec.OriginPort != nil {
			actionParam.Origin.Port = spec.OriginPort
		}
	}

	if spec.HostHeader != nil {
		actionParam.HostHeader = strings.TrimSpace(*spec.HostHeader)
	}

	if spec.SNI != nil {
		actionParam.SNI = &RouteSNIValue{Value: strings.TrimSpace(*spec.SNI)}
	}

	return OriginRule{
		Action:      "route",
		Expression:  expression,
		Description: description,
		Enabled:     enabled,
		ActionParam: actionParam,
	}, nil
}

func cloneOriginActionParameters(actionParam *OriginActionParameters) *OriginActionParameters {
	if actionParam == nil {
		return &OriginActionParameters{}
	}

	clone := &OriginActionParameters{
		HostHeader: actionParam.HostHeader,
	}

	if actionParam.Origin != nil {
		clone.Origin = &RouteOrigin{
			Host: actionParam.Origin.Host,
			Port: actionParam.Origin.Port,
		}
	}

	if actionParam.SNI != nil {
		clone.SNI = &RouteSNIValue{Value: actionParam.SNI.Value}
	}

	return clone
}

func originRuleResolvedMatchMode(matchMode string) string {
	if matchMode == "" {
		return originRuleMatchCustom
	}

	return matchMode
}

func resolveOriginRuleZoneName(rule string, integration core.IntegrationContext) string {
	zoneID, _, ok := strings.Cut(strings.TrimSpace(rule), "/")
	if !ok || zoneID == "" {
		return ""
	}

	return resolveZoneName(zoneID, integration)
}

func resolveZoneName(value string, integration core.IntegrationContext) string {
	if value == "" || integration == nil {
		return ""
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return ""
	}

	for _, zone := range metadata.Zones {
		if zone.ID == value || zone.Name == value {
			return zone.Name
		}
	}

	return ""
}

func setOriginRuleNodeMetadata(metadata core.MetadataWriter, nodeMetadata OriginRuleNodeMetadata, originHost *string, originPort *int, hostHeader, sni *string) error {
	if metadata == nil {
		return nil
	}

	if originHost != nil {
		nodeMetadata.OriginHost = strings.TrimSpace(*originHost)
		nodeMetadata.Rewrites = append(nodeMetadata.Rewrites, "DNS Record")
	}

	if hostHeader != nil {
		nodeMetadata.HostHeader = strings.TrimSpace(*hostHeader)
		nodeMetadata.Rewrites = append(nodeMetadata.Rewrites, "Host Header")
	}

	if sni != nil {
		nodeMetadata.SNI = strings.TrimSpace(*sni)
		nodeMetadata.Rewrites = append(nodeMetadata.Rewrites, "SNI")
	}

	if originPort != nil {
		nodeMetadata.OriginPort = originPort
		nodeMetadata.Rewrites = append(nodeMetadata.Rewrites, "Destination Port")
	}

	return metadata.Set(nodeMetadata)
}

func enabledValue(enabled *bool) bool {
	if enabled == nil {
		return true
	}

	return *enabled
}

func boolPtr(value bool) *bool {
	return &value
}

func resolveOriginRuleRef(value string, client *Client, integrationMetadata any) (zoneID, ruleID string, err error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", errors.New("rule is required")
	}

	if zonePart, rulePart, ok := strings.Cut(value, "/"); ok && zonePart != "" && rulePart != "" {
		return zonePart, rulePart, nil
	}

	metadata := Metadata{}
	if decodeErr := mapstructure.Decode(integrationMetadata, &metadata); decodeErr != nil {
		return "", "", fmt.Errorf("failed to decode integration metadata: %w", decodeErr)
	}

	for _, zone := range metadata.Zones {
		rules, listErr := client.ListOriginRules(zone.ID)
		if listErr != nil {
			continue
		}

		for _, rule := range rules {
			if rule.ID == value || rule.Description == value {
				return zone.ID, rule.ID, nil
			}
		}
	}

	return "", "", fmt.Errorf("no origin rule found with ID or description %q in any zone", value)
}

func findOriginRule(rules []OriginRule, ruleID string) *OriginRule {
	for _, rule := range rules {
		if rule.ID == ruleID {
			return &rule
		}
	}

	return nil
}

func emitOriginRule(ctx core.ExecutionContext, payloadType, zoneID string, rule *OriginRule) error {
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, payloadType, []any{
		map[string]any{
			"zoneId": zoneID,
			"rule":   rule,
		},
	})
}
