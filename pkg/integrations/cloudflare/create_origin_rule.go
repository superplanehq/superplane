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

type CreateOriginRule struct{}

//go:embed example_output_create_origin_rule.json
var exampleOutputCreateOriginRuleBytes []byte

var exampleOutputCreateOriginRuleOnce sync.Once
var exampleOutputCreateOriginRule map[string]any

type CreateOriginRuleSpec struct {
	Zone        string                `json:"zone" mapstructure:"zone"`
	Description string                `json:"description" mapstructure:"description"`
	MatchMode   string                `json:"matchMode" mapstructure:"matchMode"`
	MatchRules  []OriginRuleMatchRule `json:"matchRules" mapstructure:"matchRules"`
	Expression  string                `json:"expression" mapstructure:"expression"`
	OriginHost  *string               `json:"originHost" mapstructure:"originHost"`
	OriginPort  *int                  `json:"originPort" mapstructure:"originPort"`
	HostHeader  *string               `json:"hostHeader" mapstructure:"hostHeader"`
	SNI         *string               `json:"sni" mapstructure:"sni"`
	Enabled     *bool                 `json:"enabled" mapstructure:"enabled"`
}

func (c *CreateOriginRule) Name() string {
	return "cloudflare.createOriginRule"
}

func (c *CreateOriginRule) Label() string {
	return "Create Origin Rule"
}

func (c *CreateOriginRule) Description() string {
	return "Create a rule to override the origin server for matching requests"
}

func (c *CreateOriginRule) Documentation() string {
	return `The Create Origin Rule component creates a Cloudflare origin rule that routes matching requests to a different origin server.

## Configuration

- **Zone**: Select the Cloudflare zone where the rule should run
- **Apply To**: Apply to all incoming requests or build a custom filter
- **Match Rules**: Field/operator/value predicates used to build the Cloudflare expression
- **DNS Record**: Optional hostname Cloudflare should resolve and route to
- **Destination Port**: Optional destination port override
- **Host Header**: Optional HTTP Host header override
- **SNI**: Optional SNI override
- **Enabled**: Whether the rule is active

## Output

Emits the created origin rule on the default channel.`
}

func (c *CreateOriginRule) Icon() string {
	return "cloud"
}

func (c *CreateOriginRule) Color() string {
	return "orange"
}

func (c *CreateOriginRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateOriginRule) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateOriginRuleOnce, exampleOutputCreateOriginRuleBytes, &exampleOutputCreateOriginRule)
}

func (c *CreateOriginRule) Configuration() []configuration.Field {
	return append(originRuleConfigurationFields(false),
		configuration.Field{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Whether the origin rule is enabled",
		},
	)
}

func (c *CreateOriginRule) Setup(ctx core.SetupContext) error {
	spec := CreateOriginRuleSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Zone) == "" {
		return errors.New("zone is required")
	}

	expression, err := validateOriginRuleFields(spec.MatchMode, spec.MatchRules, spec.Expression, spec.OriginHost, spec.OriginPort, spec.HostHeader, spec.SNI)
	if err != nil {
		return err
	}

	return setOriginRuleNodeMetadata(ctx.Metadata, OriginRuleNodeMetadata{
		Zone:       spec.Zone,
		ZoneName:   resolveZoneName(spec.Zone, ctx.Integration),
		MatchMode:  originRuleResolvedMatchMode(spec.MatchMode),
		Expression: expression,
		Enabled:    boolPtr(enabledValue(spec.Enabled)),
	}, spec.OriginHost, spec.OriginPort, spec.HostHeader, spec.SNI)
}

func (c *CreateOriginRule) Execute(ctx core.ExecutionContext) error {
	spec := CreateOriginRuleSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	zoneID := resolveZoneID(spec.Zone, ctx.Integration)
	expression, err := buildOriginExpression(spec.MatchMode, spec.MatchRules, spec.Expression)
	if err != nil {
		return err
	}

	ruleReq := buildOriginRule(spec.Description, expression, spec.OriginHost, spec.OriginPort, spec.HostHeader, spec.SNI, enabledValue(spec.Enabled))

	ruleset, err := client.GetOriginRulesetForPhase(zoneID)
	if err != nil {
		if !isCloudflareNotFound(err) {
			return fmt.Errorf("error getting origin ruleset: %w", err)
		}

		ruleset, err = client.CreateOriginRuleset(zoneID, CreateOriginRulesetRequest{
			Name:        "Origin Rules ruleset",
			Description: "Zone-level ruleset that will execute origin rules.",
			Kind:        "zone",
			Phase:       originRulePhase,
			Rules:       []OriginRule{ruleReq},
		})
		if err != nil {
			return fmt.Errorf("failed to create origin ruleset: %w", err)
		}

		if len(ruleset.Rules) == 0 {
			return errors.New("created origin rule not found in response")
		}

		return emitOriginRule(ctx, c.Name(), zoneID, &ruleset.Rules[len(ruleset.Rules)-1])
	}

	rule, err := client.CreateOriginRule(zoneID, ruleset.ID, ruleReq)
	if err != nil {
		return fmt.Errorf("failed to create origin rule: %w", err)
	}

	return emitOriginRule(ctx, c.Name(), zoneID, rule)
}

func (c *CreateOriginRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateOriginRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateOriginRule) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateOriginRule) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateOriginRule) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateOriginRule) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
