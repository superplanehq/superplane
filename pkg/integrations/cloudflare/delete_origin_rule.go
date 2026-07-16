package cloudflare

import (
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/utils"
)

type DeleteOriginRule struct{}

//go:embed example_output_delete_origin_rule.json
var exampleOutputDeleteOriginRuleBytes []byte
var exampleOutputDeleteOriginRule = utils.NewEmbeddedJSON(exampleOutputDeleteOriginRuleBytes)

type DeleteOriginRuleSpec struct {
	Rule string `json:"rule"`
}

func (c *DeleteOriginRule) Name() string {
	return "cloudflare.deleteOriginRule"
}

func (c *DeleteOriginRule) Label() string {
	return "Delete Origin Rule"
}

func (c *DeleteOriginRule) Description() string {
	return "Remove an origin rule"
}

func (c *DeleteOriginRule) Documentation() string {
	return `The Delete Origin Rule component removes a Cloudflare origin rule from a zone.

## Configuration

- **Rule**: Select the origin rule to delete

## Output

Emits the deleted origin rule on the default channel.`
}

func (c *DeleteOriginRule) Icon() string {
	return "cloud"
}

func (c *DeleteOriginRule) Color() string {
	return "orange"
}

func (c *DeleteOriginRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteOriginRule) ExampleOutput() map[string]any {
	return exampleOutputDeleteOriginRule.Value()
}

func (c *DeleteOriginRule) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "rule",
			Label:       "Rule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The origin rule to delete",
			Placeholder: "Select origin rule",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "origin_rule",
				},
			},
		},
	}
}

func (c *DeleteOriginRule) Setup(ctx core.SetupContext) error {
	spec := DeleteOriginRuleSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Rule) == "" {
		return errors.New("rule is required")
	}

	if ctx.Metadata == nil {
		return nil
	}

	return ctx.Metadata.Set(OriginRuleNodeMetadata{
		Rule:     spec.Rule,
		ZoneName: resolveOriginRuleZoneName(spec.Rule, ctx.Integration),
	})
}

func (c *DeleteOriginRule) Execute(ctx core.ExecutionContext) error {
	spec := DeleteOriginRuleSpec{}
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

	deleted := findOriginRule(ruleset.Rules, ruleID)
	if deleted == nil {
		return fmt.Errorf("origin rule %q not found", ruleID)
	}

	if _, err := client.DeleteOriginRule(zoneID, ruleset.ID, ruleID); err != nil {
		return fmt.Errorf("failed to delete origin rule: %w", err)
	}

	return emitOriginRule(ctx, c.Name(), zoneID, deleted)
}

func (c *DeleteOriginRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteOriginRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteOriginRule) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteOriginRule) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteOriginRule) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteOriginRule) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
