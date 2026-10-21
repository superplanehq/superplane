package factories

import (
	"fmt"
	"slices"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/yaml"
)

// BuiltInAutomationSettingsError identifies a configuration failure that an
// administrator can correct. Infrastructure errors must not expose their details.
type BuiltInAutomationSettingsError struct {
	cause error
}

func (e *BuiltInAutomationSettingsError) Error() string { return e.cause.Error() }
func (e *BuiltInAutomationSettingsError) Unwrap() error { return e.cause }

// The canvas publisher can record invalid configuration as a node warning.
// An organization reset must instead fail atomically if settings cannot apply.
func validateBuiltInResetSettings(doc *yaml.Canvas, registry *registry.Registry) error {
	for _, node := range doc.Spec.Nodes {
		if err := validateBuiltInResetNode(node, registry); err != nil {
			return &BuiltInAutomationSettingsError{cause: fmt.Errorf("cannot apply settings for %q: %w", node.Name, err)}
		}
	}
	return nil
}

func validateBuiltInResetNode(node yaml.Node, registry *registry.Registry) error {
	var fields []configuration.Field
	var validator core.NodeConfigurationValidator
	if node.Type == yaml.NodeTypeTrigger {
		trigger, err := registry.GetTrigger(node.Component)
		if err != nil {
			return err
		}
		fields = trigger.Configuration()
	} else if isFactoryAgentHarness(node.Component) || node.Component == "github.createPullRequest" || node.Component == "bitbucket.createPullRequest" || isPrFeedbackWaitComponent(node.Component) {
		action, err := registry.GetAction(node.Component)
		if err != nil {
			return err
		}
		fields = action.Configuration()
		validator, _ = action.(core.NodeConfigurationValidator)
	} else {
		return nil
	}
	selected := []configuration.Field{}
	for _, field := range fields {
		if value, present := node.Configuration[field.Name]; !present || value == nil {
			continue
		}
		if node.Type != yaml.NodeTypeTrigger && !slices.Contains(factoryAgentSettingFields, field.Name) && field.Name != "draft" && !isPrFeedbackWaitComponent(node.Component) {
			continue
		}
		selected = append(selected, field)
	}
	if err := configuration.ValidateConfiguration(selected, node.Configuration); err != nil {
		return err
	}
	if node.Concurrency != nil && node.Concurrency.Max != nil && *node.Concurrency.Max < 1 {
		return fmt.Errorf("maximum simultaneous runs must be at least 1")
	}
	for _, field := range factoryAgentSettingFields {
		if field == "additionalInstructions" {
			continue // User instructions are literal text, not expressions.
		}
		value, _ := node.Configuration[field].(string)
		if configuration.ExpressionPlaceholderRegex.MatchString(value) {
			return nil
		}
	}
	// Missing BYOK credentials can already exist on an unconfigured factory.
	// Keep that connection visible rather than rejecting an otherwise valid reset.
	if isFactoryAgentHarness(node.Component) && validator != nil &&
		(node.Component == models.SuperPlaneRunnerComponent || node.Configuration["credentials"] != nil) {
		return validator.ValidateNodeConfiguration(node.Configuration)
	}
	return nil
}
