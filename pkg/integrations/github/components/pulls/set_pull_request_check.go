package pulls

import (
	"fmt"
	"slices"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// SetPullRequestCheck shares ordered delivery with review checks, without
// creating reviewer requirements or interpreting a named automation's report.
type SetPullRequestCheck struct{ SetPullRequestReviewCheck }

func (*SetPullRequestCheck) Name() string  { return "github.setPullRequestCheck" }
func (*SetPullRequestCheck) Label() string { return "Set Pull Request Check" }
func (*SetPullRequestCheck) Description() string {
	return "Publish a durable status for a pull request commit"
}
func (*SetPullRequestCheck) Documentation() string {
	return `Publish pending, success, failure, or error through GitHub commit statuses. Use any canvas and status context. No reviewer requirements or Factory records are needed.

Pending monitors the persisted attempt. Failure, cancellation, completion without a final status, removal, or a changed or closed pull request resolves pending as error on the assessed commit. Terminal delivery survives source history cleanup and retries without running earlier steps.

The action waits for delivery. Persisted source and attempt records determine order. Older attempts cannot replace newer state, including on the same commit. One attempt can move from pending to one terminal result. Delivery is at least once; an interrupted write can repeat an identical status.

Permission and missing-resource errors stop delivery and fail the action. Correct access and retry the step to resume. Transient failures retry through the existing integration worker. This component does not change branch rules or Checks API permissions.`
}
func (*SetPullRequestCheck) Configuration() []configuration.Field {
	fields := (&SetPullRequestReviewCheck{}).Configuration()
	common := make([]configuration.Field, 0, len(fields))
	for _, field := range fields {
		if slices.Contains([]string{"enabled", "activationNodeId", "phase", "reviewers", "reviewerRequestExecutionId"}, field.Name) {
			continue
		}
		common = append(common, field)
	}
	return append(common, configuration.Field{Name: "state", Label: "State", Type: configuration.FieldTypeSelect, Required: true,
		TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
			{Label: "Pending", Value: "pending"}, {Label: "Success", Value: "success"}, {Label: "Failure", Value: "failure"}, {Label: "Error", Value: "error"},
		}}},
	})
}
func (*SetPullRequestCheck) Setup(ctx core.SetupContext) error {
	return (&ListPullRequestReviewCandidates{}).Setup(ctx)
}

// Node and canvas removal wake owned publications through the generic model
// boundary. Direct status checks do not use reviewer activation or release rules.
func (*SetPullRequestCheck) Cleanup(core.SetupContext) error { return nil }
func (*SetPullRequestCheck) Execute(ctx core.ExecutionContext) error {
	config, err := decodePullRequestCheck(ctx.Configuration)
	if err != nil {
		return err
	}
	if config.Phase != "" || !slices.Contains([]string{"pending", "success", "failure", "error"}, config.State) {
		return fmt.Errorf("state must be pending, success, failure, or error")
	}
	return executePullRequestCheck(ctx, config)
}
func (*SetPullRequestCheck) ExampleOutput() map[string]any {
	return map[string]any{"type": "github.pullRequestCheck", "timestamp": "2026-11-13T09:00:00Z", "data": map[string]any{
		"status": "applied", "state": "success", "sha": "abc", "context": "team/assessment", "generation": 2,
	}}
}
