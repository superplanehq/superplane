package pulls

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// SetPullRequestCheckRun uses the ordered status lifecycle with Checks API delivery.
type SetPullRequestCheckRun struct{ SetPullRequestCheck }

type checkRunActivationConfiguration struct {
	Enabled          *bool  `mapstructure:"enabled"`
	ActivationNodeID string `mapstructure:"activationNodeId"`
}

func (*SetPullRequestCheckRun) Name() string  { return "github.setPullRequestCheckRun" }
func (*SetPullRequestCheckRun) Label() string { return "Set Pull Request Check Run" }
func (*SetPullRequestCheckRun) Description() string {
	return "Publish a running check for a pull request assessment"
}
func (*SetPullRequestCheckRun) Configuration() []configuration.Field {
	return append((&SetPullRequestCheck{}).Configuration(),
		configuration.Field{Name: "enabled", Label: "Enabled", Type: configuration.FieldTypeBool, Default: true, Description: "Disabling this node releases its managed checks."},
		configuration.Field{Name: "activationNodeId", Label: "Activation node ID", Type: configuration.FieldTypeString, Description: "Optional node whose enabled state and activation generation govern this check."})
}

func (c *SetPullRequestCheckRun) Setup(ctx core.SetupContext) error {
	var config checkRunActivationConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}
	if config.ActivationNodeID != "" && config.ActivationNodeID != ctx.NodeID {
		return nil
	}
	if config.Enabled != nil && !*config.Enabled {
		return (&SetPullRequestReviewCheck{}).Cleanup(ctx)
	}
	return c.SetPullRequestCheck.Setup(ctx)
}

func (*SetPullRequestCheckRun) Documentation() string {
	return `Publish a GitHub check run for an assessed pull request commit. Use the same configuration as Set Pull Request Check.

Pending shows a running check. Success completes the check successfully. Failure and error complete the check as failed. The output includes the reason and result URL.

The same attempt updates one check run. A stable external ID recovers interrupted creation. New attempts retire running checks, including retries on the same commit. Results stay on the assessed commit.

Cancellation, removal, a closed or changed pull request, and completion without a final result fail the running check. Delivery uses the existing durable retries and ordering.

Enabled defaults to true. Bind Activation Node ID to the starting check node to apply its admitted and live activation generation. Disabling that node releases its outstanding checks. Increase the generation when enabling it again so old callbacks cannot reopen them.

The GitHub App needs Checks write and Pull requests read permissions. Existing installations must accept the permission update. This component does not change branch rules.`
}
func (*SetPullRequestCheckRun) Execute(ctx core.ExecutionContext) error {
	config, err := decodePullRequestCheck(ctx.Configuration)
	if err != nil {
		return err
	}
	if config.Phase != "" || !slices.Contains([]string{"pending", "success", "failure", "error"}, config.State) {
		return fmt.Errorf("state must be pending, success, failure, or error")
	}
	config.CheckRun = true
	if config.Enabled != nil && !*config.Enabled {
		return emitReviewCheck(ctx.ExecutionState, "skipped", nil, nil)
	}
	if config.ActivationNodeID != "" {
		if ctx.Publications == nil {
			return fmt.Errorf("publication context is required")
		}
		order, err := ctx.Publications.ResolveOrder(uuid.Nil, uuid.Nil)
		if err != nil {
			return err
		}
		generation, eligible, err := reviewCheckActivation(ctx.Publications, uuid.MustParse(ctx.WorkflowID), config.ActivationNodeID, order.AttemptRootEventID)
		if err != nil {
			return err
		}
		if !eligible {
			return emitReviewCheck(ctx.ExecutionState, "skipped", nil, nil)
		}
		config.ActivationGeneration = generation
	}
	return executePullRequestCheck(ctx, config)
}
