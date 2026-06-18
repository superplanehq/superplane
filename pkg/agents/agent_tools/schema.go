package agenttools

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentSchemaAgentToolName = "superplane_component_schema"

const (
	componentSchemaDefaultLimit = 40
	componentSchemaExampleLimit = 1600
)

func init() {
	Register[superPlaneComponentSchemaInput](ComponentSchemaAgentToolName, func(deps Dependencies) AgentTool[superPlaneComponentSchemaInput] {
		return NewComponentSchemaAgentTool(deps.ComponentRegistry)
	})
}

var _ AgentTool[superPlaneComponentSchemaInput] = (*ComponentSchemaAgentTool)(nil)

type ComponentSchemaAgentTool struct {
	registry *registry.Registry
}

func NewComponentSchemaAgentTool(registry *registry.Registry) *ComponentSchemaAgentTool {
	return &ComponentSchemaAgentTool{registry: registry}
}

func (t *ComponentSchemaAgentTool) Name() string {
	return ComponentSchemaAgentToolName
}

func (t *ComponentSchemaAgentTool) Description() string {
	return "Lookup exact SuperPlane component, trigger, and widget schemas from the backend registry without reading mounted reference files. Use this before researcher delegation when you need YAML component keys, configuration fields, output channel names, integration requirements, or compact examples. Prefer this tool for repeated schema lookups; mounted docs are fallback only."
}

func (t *ComponentSchemaAgentTool) InputSchema() agents.CustomToolInputSchema {
	return agents.CustomToolInputSchema{
		Type: "object",
		Properties: map[string]agents.CustomToolInputSchema{
			"component_keys": {
				Type:        "array",
				Description: "Exact component, trigger, or widget keys to look up, for example schedule, http, wait, slack.waitForButtonClick.",
				Items:       &agents.CustomToolInputSchema{Type: "string"},
			},
			"vendors": {
				Type:        "array",
				Description: "Vendor names to list schemas for, for example slack, github, grafana.",
				Items:       &agents.CustomToolInputSchema{Type: "string"},
			},
			"query": {
				Type:        "string",
				Description: "Search term used against component keys, labels, descriptions, kind, and required integration vendor.",
			},
			"include_examples": {
				Type:        "boolean",
				Description: "Include compact example input/output payloads when available. Honored only for exact component_keys lookups; broad vendor and query lookups stay compact.",
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum schemas to return. Defaults to 40 and is capped at 40.",
			},
		},
	}
}

func (t *ComponentSchemaAgentTool) Call(_ context.Context, _ agents.AgentSessionContext, input superPlaneComponentSchemaInput) (Result, error) {
	if t.registry == nil {
		return Result{}, fmt.Errorf("component schema registry is not configured")
	}
	return Result{Payload: t.lookup(input)}, nil
}

func (t *ComponentSchemaAgentTool) lookup(input superPlaneComponentSchemaInput) superPlaneComponentSchemaResult {
	limit := input.Limit
	if limit <= 0 || limit > componentSchemaDefaultLimit {
		limit = componentSchemaDefaultLimit
	}

	seen := map[string]struct{}{}
	components := []superPlaneComponentSchema{}
	missing := []string{}
	omitted := []string{}
	truncated := false

	for _, key := range normalizedList(input.ComponentKeys) {
		component, err := t.lookupComponent(key, input.IncludeExamples)
		if err != nil {
			missing = append(missing, key)
			continue
		}
		var limitHit bool
		components, limitHit = appendUniqueComponent(components, seen, component, limit)
		if limitHit {
			omitted = append(omitted, key)
			truncated = true
		}
	}

	for _, vendor := range normalizedList(input.Vendors) {
		for _, component := range t.vendorComponents(vendor, false) {
			var limitHit bool
			components, limitHit = appendUniqueComponent(components, seen, component, limit)
			if limitHit {
				omitted = append(omitted, component.Key)
				truncated = true
			}
		}
	}

	query := strings.ToLower(strings.TrimSpace(input.Query))
	if query != "" {
		for _, component := range t.allComponents(false) {
			if componentMatchesQuery(component, query) {
				var limitHit bool
				components, limitHit = appendUniqueComponent(components, seen, component, limit)
				if limitHit {
					omitted = append(omitted, component.Key)
					truncated = true
				}
			}
		}
	}

	sort.Slice(components, func(i, j int) bool { return components[i].Key < components[j].Key })
	notes := []string{
		"Use output_channels.name exactly in edge channel values; labels are display-only.",
		"Components with requires_integration need a connected integration instance before running.",
		"Do not read mounted component docs for these components unless validation reports an unfamiliar field or channel.",
		"Examples are included only for exact component_keys lookups to keep broad vendor/query results compact.",
	}
	if truncated {
		notes = append(notes, "Result was truncated by limit; request omitted component_keys explicitly or raise limit up to 40 if you need more.")
	}

	return superPlaneComponentSchemaResult{
		Action:     "lookup",
		Components: components,
		Missing:    missing,
		Omitted:    omitted,
		Truncated:  truncated,
		Notes:      notes,
	}
}

func (t *ComponentSchemaAgentTool) lookupComponent(key string, includeExamples bool) (superPlaneComponentSchema, error) {
	if action, err := t.registry.GetAction(key); err == nil {
		return actionSchema(action, integrationVendor(key), includeExamples), nil
	}
	if trigger, err := t.registry.GetTrigger(key); err == nil {
		return triggerSchema(trigger, integrationVendor(key), includeExamples), nil
	}
	if widget, err := t.registry.GetWidget(key); err == nil {
		return widgetSchema(widget), nil
	}
	return superPlaneComponentSchema{}, fmt.Errorf("component %s not found", key)
}

func (t *ComponentSchemaAgentTool) vendorComponents(vendor string, includeExamples bool) []superPlaneComponentSchema {
	integration, err := t.registry.GetIntegration(vendor)
	if err != nil {
		return nil
	}

	components := []superPlaneComponentSchema{}
	for _, trigger := range integration.Triggers() {
		components = append(components, triggerSchema(trigger, vendor, includeExamples))
	}
	for _, action := range integration.Actions() {
		components = append(components, actionSchema(action, vendor, includeExamples))
	}
	sort.Slice(components, func(i, j int) bool { return components[i].Key < components[j].Key })
	return components
}

func (t *ComponentSchemaAgentTool) allComponents(includeExamples bool) []superPlaneComponentSchema {
	components := []superPlaneComponentSchema{}
	for _, trigger := range t.registry.ListTriggers() {
		components = append(components, triggerSchema(trigger, "", includeExamples))
	}
	for _, action := range t.registry.ListActions() {
		components = append(components, actionSchema(action, "", includeExamples))
	}
	for _, widget := range t.registry.ListWidgets() {
		components = append(components, widgetSchema(widget))
	}
	for _, integration := range t.registry.ListIntegrations() {
		components = append(components, t.vendorComponents(integration.Name(), includeExamples)...)
	}
	sort.Slice(components, func(i, j int) bool { return components[i].Key < components[j].Key })
	return components
}

func actionSchema(action core.Action, vendor string, includeExamples bool) superPlaneComponentSchema {
	outputChannels := outputChannelSchemas(safeActionOutputChannels(action))
	if len(outputChannels) == 0 {
		outputChannels = outputChannelSchemas([]core.OutputChannel{core.DefaultOutputChannel})
	}

	schema := superPlaneComponentSchema{
		Key:                 action.Name(),
		Kind:                "action",
		Label:               action.Label(),
		Description:         action.Description(),
		RequiresIntegration: vendor,
		Configuration:       fieldSchemas(safeActionConfiguration(action)),
		OutputChannels:      outputChannels,
	}
	if includeExamples {
		schema.ExampleOutput = compactJSON(safeActionExampleOutput(action), componentSchemaExampleLimit)
	}
	return schema
}

func triggerSchema(trigger core.Trigger, vendor string, includeExamples bool) superPlaneComponentSchema {
	schema := superPlaneComponentSchema{
		Key:                 trigger.Name(),
		Kind:                "trigger",
		Label:               trigger.Label(),
		Description:         trigger.Description(),
		RequiresIntegration: vendor,
		Configuration:       fieldSchemas(safeTriggerConfiguration(trigger)),
		OutputChannels:      outputChannelSchemas([]core.OutputChannel{core.DefaultOutputChannel}),
	}
	if includeExamples {
		schema.ExampleData = compactJSON(safeTriggerExampleData(trigger), componentSchemaExampleLimit)
	}
	return schema
}

func widgetSchema(widget core.Widget) superPlaneComponentSchema {
	return superPlaneComponentSchema{
		Key:           widget.Name(),
		Kind:          "widget",
		Label:         widget.Label(),
		Description:   widget.Description(),
		Configuration: fieldSchemas(safeWidgetConfiguration(widget)),
	}
}

func fieldSchemas(fields []configuration.Field) []superPlaneComponentField {
	result := make([]superPlaneComponentField, 0, len(fields))
	for _, field := range fields {
		schema := superPlaneComponentField{
			Name:               field.Name,
			Type:               field.Type,
			Label:              field.Label,
			Description:        truncateString(field.Description, 300),
			Required:           field.Required,
			Default:            field.Default,
			RequiredWhen:       requiredConditionSchemas(field.RequiredConditions),
			VisibleWhen:        visibilityConditionSchemas(field.VisibilityConditions),
			Options:            fieldOptions(field.TypeOptions),
			ResourceType:       resourceType(field.TypeOptions),
			ListItemDefinition: listItemDefinition(field.TypeOptions),
		}
		result = append(result, schema)
	}
	return result
}

func outputChannelSchemas(channels []core.OutputChannel) []superPlaneOutputChannel {
	result := make([]superPlaneOutputChannel, 0, len(channels))
	for _, channel := range channels {
		name := channel.Name
		if name == "" {
			name = core.DefaultOutputChannel.Name
		}
		result = append(result, superPlaneOutputChannel{
			Name:        name,
			Label:       channel.Label,
			Description: truncateString(channel.Description, 200),
		})
	}
	return result
}

func requiredConditionSchemas(conditions []configuration.RequiredCondition) []superPlaneFieldCondition {
	result := make([]superPlaneFieldCondition, 0, len(conditions))
	for _, condition := range conditions {
		result = append(result, superPlaneFieldCondition{
			Field:  condition.Field,
			Values: append([]string(nil), condition.Values...),
		})
	}
	return result
}

func visibilityConditionSchemas(conditions []configuration.VisibilityCondition) []superPlaneFieldCondition {
	result := make([]superPlaneFieldCondition, 0, len(conditions))
	for _, condition := range conditions {
		result = append(result, superPlaneFieldCondition{
			Field:  condition.Field,
			Values: append([]string(nil), condition.Values...),
		})
	}
	return result
}

func safeActionOutputChannels(action core.Action) (channels []core.OutputChannel) {
	defer func() {
		if recover() != nil {
			channels = nil
		}
	}()
	return action.OutputChannels(nil)
}

func safeActionConfiguration(action core.Action) (fields []configuration.Field) {
	defer func() {
		if recover() != nil {
			fields = nil
		}
	}()
	return action.Configuration()
}

func safeActionExampleOutput(action core.Action) (output map[string]any) {
	defer func() {
		if recover() != nil {
			output = nil
		}
	}()
	return action.ExampleOutput()
}

func safeTriggerConfiguration(trigger core.Trigger) (fields []configuration.Field) {
	defer func() {
		if recover() != nil {
			fields = nil
		}
	}()
	return trigger.Configuration()
}

func safeTriggerExampleData(trigger core.Trigger) (data map[string]any) {
	defer func() {
		if recover() != nil {
			data = nil
		}
	}()
	return trigger.ExampleData()
}

func safeWidgetConfiguration(widget core.Widget) (fields []configuration.Field) {
	defer func() {
		if recover() != nil {
			fields = nil
		}
	}()
	return widget.Configuration()
}

func fieldOptions(options *configuration.TypeOptions) []superPlaneFieldOption {
	if options == nil {
		return nil
	}
	source := []configuration.FieldOption{}
	switch {
	case options.Select != nil:
		source = options.Select.Options
	case options.MultiSelect != nil:
		source = options.MultiSelect.Options
	case options.AnyPredicateList != nil:
		source = options.AnyPredicateList.Operators
	}

	result := make([]superPlaneFieldOption, 0, len(source))
	for _, option := range source {
		result = append(result, superPlaneFieldOption{
			Label:       option.Label,
			Value:       option.Value,
			Description: truncateString(option.Description, 160),
		})
	}
	return result
}

func resourceType(options *configuration.TypeOptions) string {
	if options == nil || options.Resource == nil {
		return ""
	}
	return options.Resource.Type
}

func listItemDefinition(options *configuration.TypeOptions) []superPlaneComponentField {
	if options == nil || options.List == nil || options.List.ItemDefinition == nil {
		return nil
	}
	return fieldSchemas(options.List.ItemDefinition.Schema)
}

func appendUniqueComponent(
	components []superPlaneComponentSchema,
	seen map[string]struct{},
	component superPlaneComponentSchema,
	limit int,
) ([]superPlaneComponentSchema, bool) {
	if _, ok := seen[component.Key]; ok {
		return components, false
	}
	if len(components) >= limit {
		return components, true
	}
	seen[component.Key] = struct{}{}
	return append(components, component), false
}

func normalizedList(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !slices.Contains(normalized, value) {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

func integrationVendor(componentKey string) string {
	parts := strings.SplitN(componentKey, ".", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}

func componentMatchesQuery(component superPlaneComponentSchema, query string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		component.Key,
		component.Kind,
		component.Label,
		component.Description,
		component.RequiresIntegration,
	}, " "))
	return strings.Contains(haystack, query)
}

func compactJSON(value any, limit int) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return truncateString(string(data), limit)
}

func truncateString(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

type superPlaneComponentSchemaInput struct {
	ComponentKeys   []string `json:"component_keys,omitempty"`
	Vendors         []string `json:"vendors,omitempty"`
	Query           string   `json:"query,omitempty"`
	IncludeExamples bool     `json:"include_examples,omitempty"`
	Limit           int      `json:"limit,omitempty"`
}

type superPlaneComponentSchemaResult struct {
	Action     string                      `json:"action"`
	Components []superPlaneComponentSchema `json:"components"`
	Missing    []string                    `json:"missing,omitempty"`
	Omitted    []string                    `json:"omitted,omitempty"`
	Truncated  bool                        `json:"truncated,omitempty"`
	Notes      []string                    `json:"notes,omitempty"`
}

type superPlaneComponentSchema struct {
	Key                 string                     `json:"key"`
	Kind                string                     `json:"kind"`
	Label               string                     `json:"label,omitempty"`
	Description         string                     `json:"description,omitempty"`
	RequiresIntegration string                     `json:"requires_integration,omitempty"`
	Configuration       []superPlaneComponentField `json:"configuration,omitempty"`
	OutputChannels      []superPlaneOutputChannel  `json:"output_channels,omitempty"`
	ExampleData         string                     `json:"example_data,omitempty"`
	ExampleOutput       string                     `json:"example_output,omitempty"`
}

type superPlaneComponentField struct {
	Name               string                     `json:"name"`
	Type               string                     `json:"type"`
	Label              string                     `json:"label,omitempty"`
	Description        string                     `json:"description,omitempty"`
	Required           bool                       `json:"required"`
	Default            any                        `json:"default,omitempty"`
	ResourceType       string                     `json:"resource_type,omitempty"`
	Options            []superPlaneFieldOption    `json:"options,omitempty"`
	RequiredWhen       []superPlaneFieldCondition `json:"required_when,omitempty"`
	VisibleWhen        []superPlaneFieldCondition `json:"visible_when,omitempty"`
	ListItemDefinition []superPlaneComponentField `json:"list_item_definition,omitempty"`
}

type superPlaneFieldOption struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

type superPlaneFieldCondition struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
}

type superPlaneOutputChannel struct {
	Name        string `json:"name"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}
