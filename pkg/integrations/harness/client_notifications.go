package harness

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func (c *Client) GetPipelineYAML(pipelineIdentifier string) (string, error) {
	if err := c.ensureProjectScope(); err != nil {
		return "", err
	}

	pipelineIdentifier = strings.TrimSpace(pipelineIdentifier)
	if pipelineIdentifier == "" {
		return "", fmt.Errorf("pipeline identifier is required")
	}

	_, body, err := c.execRequest(
		http.MethodGet,
		fmt.Sprintf("/pipeline/api/pipelines/%s", url.PathEscape(pipelineIdentifier)),
		c.scopeQuery(),
		nil,
		false,
	)
	if err != nil {
		return "", err
	}

	response := map[string]any{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse pipeline response: %w", err)
	}

	pipelineYAML := firstNonEmpty(
		readStringPath(response, "data", "yamlPipeline"),
		readStringPath(response, "data", "pipeline", "yaml"),
	)
	if pipelineYAML == "" {
		return "", fmt.Errorf("pipeline yaml not found in response")
	}

	return pipelineYAML, nil
}

func (c *Client) updatePipelineYAML(pipelineIdentifier, pipelineYAML string) error {
	pipelineIdentifier = strings.TrimSpace(pipelineIdentifier)
	if pipelineIdentifier == "" {
		return fmt.Errorf("pipeline identifier is required")
	}
	if strings.TrimSpace(pipelineYAML) == "" {
		return fmt.Errorf("pipeline yaml is required")
	}

	_, _, err := c.execRawRequest(
		http.MethodPut,
		fmt.Sprintf("/pipeline/api/pipelines/v2/%s", url.PathEscape(pipelineIdentifier)),
		c.scopeQuery(),
		[]byte(pipelineYAML),
		"application/yaml",
		nil,
	)
	return err
}

func (c *Client) UpsertPipelineNotificationRule(request UpsertPipelineNotificationRuleRequest) error {
	if err := c.ensureProjectScope(); err != nil {
		return err
	}

	request.PipelineIdentifier = strings.TrimSpace(request.PipelineIdentifier)
	request.RuleIdentifier = normalizeHarnessIdentifier(request.RuleIdentifier)
	request.RuleName = normalizeHarnessName(request.RuleName, request.RuleIdentifier)
	request.WebhookURL = strings.TrimSpace(request.WebhookURL)
	request.EventTypes = normalizeNotificationRuleEventTypes(request.EventTypes)

	if request.PipelineIdentifier == "" {
		return fmt.Errorf("pipeline identifier is required")
	}
	if request.RuleIdentifier == "" {
		return fmt.Errorf("rule identifier is required")
	}
	if request.RuleName == "" {
		return fmt.Errorf("rule name is required")
	}
	if request.WebhookURL == "" {
		return fmt.Errorf("webhook url is required")
	}

	pipelineYAML, err := c.GetPipelineYAML(request.PipelineIdentifier)
	if err != nil {
		return err
	}

	root, err := decodePipelineYAML(pipelineYAML)
	if err != nil {
		return err
	}

	pipeline, err := ensurePipelineDocument(root)
	if err != nil {
		return err
	}

	rules := notificationRulesFromPipeline(pipeline)
	newRule := buildPipelineNotificationRule(request)
	replaced := false

	for idx, item := range rules {
		rule, ok := item.(map[string]any)
		if !ok {
			continue
		}

		identifier := firstNonEmpty(readString(rule["identifier"]), readString(rule["name"]))
		if identifier != request.RuleIdentifier {
			continue
		}

		rules[idx] = newRule
		replaced = true
		break
	}

	if !replaced {
		rules = append(rules, newRule)
	}

	pipeline["notificationRules"] = rules

	updatedYAML, err := encodePipelineYAML(root)
	if err != nil {
		return err
	}

	return c.updatePipelineYAML(request.PipelineIdentifier, updatedYAML)
}

func (c *Client) DeletePipelineNotificationRule(pipelineIdentifier, ruleIdentifier string) error {
	if err := c.ensureProjectScope(); err != nil {
		return err
	}

	pipelineIdentifier = strings.TrimSpace(pipelineIdentifier)
	ruleIdentifier = normalizeHarnessIdentifier(ruleIdentifier)
	if pipelineIdentifier == "" || ruleIdentifier == "" {
		return nil
	}

	pipelineYAML, err := c.GetPipelineYAML(pipelineIdentifier)
	if err != nil {
		return err
	}

	root, err := decodePipelineYAML(pipelineYAML)
	if err != nil {
		return err
	}

	pipeline, err := ensurePipelineDocument(root)
	if err != nil {
		return err
	}

	rules := notificationRulesFromPipeline(pipeline)
	filtered := make([]any, 0, len(rules))
	removed := false

	for _, item := range rules {
		rule, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}

		identifier := firstNonEmpty(readString(rule["identifier"]), readString(rule["name"]))
		if identifier == ruleIdentifier {
			removed = true
			continue
		}

		filtered = append(filtered, item)
	}

	if !removed {
		return nil
	}

	pipeline["notificationRules"] = filtered

	updatedYAML, err := encodePipelineYAML(root)
	if err != nil {
		return err
	}

	return c.updatePipelineYAML(pipelineIdentifier, updatedYAML)
}

func decodePipelineYAML(raw string) (map[string]any, error) {
	root := map[string]any{}
	if err := yaml.Unmarshal([]byte(raw), &root); err != nil {
		return nil, fmt.Errorf("failed to decode pipeline yaml: %w", err)
	}
	return root, nil
}

func encodePipelineYAML(root map[string]any) (string, error) {
	encoded, err := yaml.Marshal(root)
	if err != nil {
		return "", fmt.Errorf("failed to encode pipeline yaml: %w", err)
	}
	return string(encoded), nil
}

func ensurePipelineDocument(root map[string]any) (map[string]any, error) {
	pipeline, ok := root["pipeline"].(map[string]any)
	if !ok || pipeline == nil {
		return nil, fmt.Errorf("pipeline yaml does not contain root pipeline field")
	}
	return pipeline, nil
}

func notificationRulesFromPipeline(pipeline map[string]any) []any {
	items, ok := pipeline["notificationRules"].([]any)
	if !ok {
		return []any{}
	}
	return items
}

func buildPipelineNotificationRule(request UpsertPipelineNotificationRuleRequest) map[string]any {
	events := make([]map[string]any, 0, len(request.EventTypes))
	for _, eventType := range request.EventTypes {
		events = append(events, map[string]any{"type": eventType})
	}

	spec := map[string]any{"webhookUrl": request.WebhookURL}
	headers := map[string]any{}
	if len(request.Headers) > 0 {
		keys := make([]string, 0, len(request.Headers))
		for key := range request.Headers {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if strings.TrimSpace(key) == "" || strings.TrimSpace(request.Headers[key]) == "" {
				continue
			}
			headers[key] = request.Headers[key]
		}
	}
	if len(headers) > 0 {
		spec["headers"] = headers
	}

	return map[string]any{
		"name":           request.RuleName,
		"identifier":     request.RuleIdentifier,
		"pipelineEvents": events,
		"notificationMethod": map[string]any{
			"type": "Webhook",
			"spec": spec,
		},
		"enabled": true,
	}
}

func normalizeNotificationRuleEventTypes(values []string) []string {
	if len(values) == 0 {
		return []string{"PipelineEnd"}
	}

	normalized := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		switch lower {
		case "pipelineend", "pipeline_end", "pipeline end":
			trimmed = "PipelineEnd"
		case "pipelinestart", "pipeline_start", "pipeline start":
			trimmed = "PipelineStart"
		case "pipelinesuccess", "pipeline_success", "pipeline success":
			trimmed = "PipelineSuccess"
		case "pipelinefailed", "pipeline_failed", "pipeline failed":
			trimmed = "PipelineFailed"
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return []string{"PipelineEnd"}
	}

	return normalized
}

func normalizeHarnessIdentifier(value string) string {
	identifier := normalizeHarnessIdentifierOrEmpty(value)
	if identifier == "" {
		return "superplane-harness"
	}

	return identifier
}

func normalizeHarnessIdentifierOrEmpty(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	builder := strings.Builder{}
	for _, char := range trimmed {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' ||
			char == '_' {
			builder.WriteRune(char)
			continue
		}
		builder.WriteRune('-')
	}

	identifier := strings.Trim(builder.String(), "-_")
	if identifier == "" {
		return ""
	}

	if len(identifier) > 127 {
		identifier = identifier[:127]
		identifier = strings.Trim(identifier, "-_")
	}

	return identifier
}

func normalizeHarnessName(name, fallback string) string {
	if normalizedName := normalizeHarnessIdentifierOrEmpty(name); normalizedName != "" {
		return normalizedName
	}

	return normalizeHarnessIdentifier(fallback)
}
