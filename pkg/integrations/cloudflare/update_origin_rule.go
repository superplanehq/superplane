package cloudflare

import (
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/utils"
)

type UpdateOriginRule struct{}

//go:embed example_output_update_origin_rule.json
var exampleOutputUpdateOriginRuleBytes []byte

var exampleOutputUpdateOriginRuleOnce sync.Once
var exampleOutputUpdateOriginRule map[string]any

type UpdateOriginRuleSpec struct {
	Rule        string                `json:"rule" mapstructure:"rule"`
	Description *string               `json:"description" mapstructure:"description"`
	MatchMode   string                `json:"matchMode" mapstructure:"matchMode"`
	MatchRules  []OriginRuleMatchRule `json:"matchRules" mapstructure:"matchRules"`
	Expression  string                `json:"expression" mapstructure:"expression"`
	OriginHost  *string               `json:"originHost" mapstructure:"originHost"`
	OriginPort  *int                  `json:"originPort" mapstructure:"originPort"`
	HostHeader  *string               `json:"hostHeader" mapstructure:"hostHeader"`
	SNI         *string               `json:"sni" mapstructure:"sni"`
	Enabled     *bool                 `json:"enabled" mapstructure:"enabled"`
}

func (c *UpdateOriginRule) Name() string {
	return "cloudflare.updateOriginRule"
}

func (c *UpdateOriginRule) Label() string {
	return "Update Origin Rule"
}

func (c *UpdateOriginRule) Description() string {
	return "Update an existing origin rule"
}

func (c *UpdateOriginRule) Documentation() string {
	return `The Update Origin Rule component updates a Cloudflare origin rule, such as changing the origin host for matching requests.

## Configuration

- **Rule**: Select the origin rule to update
- **Apply To**: Apply to all incoming requests or build a custom filter
- **Match Rules**: Field/operator/value predicates used to build the Cloudflare expression
- **DNS Record**: Optional hostname Cloudflare should resolve and route to
- **Destination Port**: Optional destination port override
- **Host Header**: Optional HTTP Host header override
- **SNI**: Optional SNI override
- **Enabled**: Whether the rule is active

## Output

Emits the updated origin rule on the default channel.`
}

func (c *UpdateOriginRule) Icon() string {
	return "cloud"
}

func (c *UpdateOriginRule) Color() string {
	return "orange"
}

func (c *UpdateOriginRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateOriginRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateOriginRuleOnce, exampleOutputUpdateOriginRuleBytes, &exampleOutputUpdateOriginRule)
}

func (c *UpdateOriginRule) Configuration() []configuration.Field {
	fields := originRuleConfigurationFields(true)
	for i := range fields {
		fields[i].Required = false
		fields[i].Togglable = true
		fields[i].RequiredConditions = nil
	}

	return append([]configuration.Field{
		{
			Name:        "rule",
			Label:       "Rule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The origin rule to update",
			Placeholder: "Select an origin rule",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "origin_rule",
				},
			},
		},
	}, append(fields,
		configuration.Field{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   true,
			Default:     true,
			Description: "Whether the origin rule is enabled",
		},
	)...)
}

func (c *UpdateOriginRule) Setup(ctx core.SetupContext) error {
	spec := UpdateOriginRuleSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Rule) == "" {
		return errors.New("rule is required")
	}

	expression, err := validateOriginRuleUpdateFields(spec)
	if err != nil {
		return err
	}

	description := ""
	if spec.Description != nil {
		description = strings.TrimSpace(*spec.Description)
	}

	return setOriginRuleNodeMetadata(ctx.Metadata, OriginRuleNodeMetadata{
		Rule:        spec.Rule,
		ZoneName:    resolveOriginRuleZoneName(spec.Rule, ctx.Integration),
		MatchMode:   originRuleResolvedMatchMode(spec.MatchMode),
		Expression:  expression,
		Description: description,
		Enabled:     spec.Enabled,
	}, spec.OriginHost, spec.OriginPort, spec.HostHeader, spec.SNI)
}

func (c *UpdateOriginRule) Execute(ctx core.ExecutionContext) error {
	spec := UpdateOriginRuleSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	zoneID, ruleID, err := resolveOriginRuleRef(spec.Rule, client, ctx.Integration.GetMetadata())
	if err != nil {
		return err
	}

	ruleset, err := client.GetOriginRulesetForPhase(zoneID)
	if err != nil {
		return fmt.Errorf("error getting origin ruleset: %w", err)
	}

	existing := findOriginRule(ruleset.Rules, ruleID)
	if existing == nil {
		return fmt.Errorf("origin rule %q not found", ruleID)
	}

	ruleReq, err := buildOriginRuleUpdateRequest(spec, existing)
	if err != nil {
		return err
	}

	rule, err := client.UpdateOriginRule(zoneID, ruleset.ID, ruleID, ruleReq)
	if err != nil {
		return fmt.Errorf("failed to update origin rule: %w", err)
	}

	return emitOriginRule(ctx, c.Name(), zoneID, rule)
}

func (c *UpdateOriginRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateOriginRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateOriginRule) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateOriginRule) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateOriginRule) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateOriginRule) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
