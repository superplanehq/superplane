package terraform

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/go-tfe"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type QueueRun struct{}
type QueueRunSpec struct {
	WorkspaceID string `json:"workspaceId"`
	Message     string `json:"message"`
}

func (c *QueueRun) Name() string        { return "terraform.queueRun" }
func (c *QueueRun) Label() string       { return "Queue Run" }
func (c *QueueRun) Description() string { return "Queues a new run for a specific Workspace." }
func (c *QueueRun) Icon() string        { return "play-circle" }
func (c *QueueRun) Color() string       { return "purple" }
func (c *QueueRun) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "workspaceId",
			Label:    "Workspace ID",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "message",
			Label:    "Run Message",
			Type:     configuration.FieldTypeString,
			Required: false,
		},
	}
}
func (c *QueueRun) Setup(ctx core.SetupContext) error { return nil }
func (c *QueueRun) Execute(ctx core.ExecutionContext) error {
	configAPI, err := ctx.Integration.GetConfig("apiToken")
	if err != nil {
		return err
	}
	configAddr, err := ctx.Integration.GetConfig("address")
	if err != nil {
		return err
	}

	client, err := NewClient(map[string]any{
		"apiToken": string(configAPI),
		"address":  string(configAddr),
	})
	if err != nil {
		return err
	}

	spec := QueueRunSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	msg := fmt.Sprintf("⚙ %s", spec.Message)

	opts := tfe.RunCreateOptions{
		Workspace: &tfe.Workspace{ID: spec.WorkspaceID},
		Message:   tfe.String(msg),
	}

	run, err := client.TFE.Runs.Create(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("failed to queue run: %w", err)
	}

	return ctx.ExecutionState.Emit("default", "", []any{
		map[string]any{
			"runId":  run.ID,
			"status": run.Status,
		},
	})
}

func (c *QueueRun) Actions() []core.Action                                    { return nil }
func (c *QueueRun) HandleAction(ctx core.ActionContext) error                 { return nil }
func (c *QueueRun) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return 200, nil }
func (c *QueueRun) Cancel(ctx core.ExecutionContext) error                    { return nil }
func (c *QueueRun) Cleanup(ctx core.SetupContext) error                       { return nil }
func (c *QueueRun) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "default", Label: "Run Queued", Description: "Emits when a new run is successfully queued"},
	}
}
func (c *QueueRun) DefaultOutputChannel() core.OutputChannel { return core.DefaultOutputChannel }
func (c *QueueRun) ExampleOutput() map[string]any {
	return map[string]any{
		"runId":  "run-xxxxxx",
		"status": "pending",
	}
}
func (c *QueueRun) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *QueueRun) Documentation() string { return "" }

type ApplyRun struct{}
type ApplyRunSpec struct {
	RunID   string `json:"runId"`
	Comment string `json:"comment"`
}

func (c *ApplyRun) Name() string  { return "terraform.applyRun" }
func (c *ApplyRun) Label() string { return "Apply Run" }
func (c *ApplyRun) Description() string {
	return "Applies a run that is paused in 'needs attention' or 'planned'."
}
func (c *ApplyRun) Icon() string  { return "check-circle" }
func (c *ApplyRun) Color() string { return "green" }
func (c *ApplyRun) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "runId", Label: "Run ID", Type: configuration.FieldTypeString, Required: true},
		{Name: "comment", Label: "Comment", Type: configuration.FieldTypeString, Required: false},
	}
}
func (c *ApplyRun) Setup(ctx core.SetupContext) error { return nil }
func (c *ApplyRun) Execute(ctx core.ExecutionContext) error {
	configAPI, err := ctx.Integration.GetConfig("apiToken")
	if err != nil {
		return err
	}
	configAddr, err := ctx.Integration.GetConfig("address")
	if err != nil {
		return err
	}

	client, err := NewClient(map[string]any{
		"apiToken": string(configAPI),
		"address":  string(configAddr),
	})
	if err != nil {
		return err
	}

	spec := ApplyRunSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	run, err := client.TFE.Runs.Read(context.Background(), spec.RunID)
	if err != nil {
		return fmt.Errorf("failed to read run: %w", err)
	}

	if run.Status != tfe.RunPlanned {
		return fmt.Errorf("run %s is currently '%s', cannot apply (must be 'planned')", spec.RunID, run.Status)
	}

	opts := tfe.RunApplyOptions{}
	if spec.Comment != "" {
		opts.Comment = tfe.String(spec.Comment)
	}

	err = client.TFE.Runs.Apply(context.Background(), spec.RunID, opts)
	if err != nil {
		return fmt.Errorf("failed to apply run: %w", err)
	}
	return ctx.ExecutionState.Emit("default", "", []any{map[string]any{"runId": spec.RunID, "action": "applied"}})
}

func (c *ApplyRun) Actions() []core.Action                                    { return nil }
func (c *ApplyRun) HandleAction(ctx core.ActionContext) error                 { return nil }
func (c *ApplyRun) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return 200, nil }
func (c *ApplyRun) Cancel(ctx core.ExecutionContext) error                    { return nil }
func (c *ApplyRun) Cleanup(ctx core.SetupContext) error                       { return nil }
func (c *ApplyRun) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "default", Label: "Run Applied", Description: "Emits when the run is successfully applied"},
	}
}
func (c *ApplyRun) DefaultOutputChannel() core.OutputChannel { return core.DefaultOutputChannel }
func (c *ApplyRun) ExampleOutput() map[string]any {
	return map[string]any{
		"runId":  "run-xxxxxx",
		"action": "applied",
	}
}
func (c *ApplyRun) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *ApplyRun) Documentation() string { return "" }

type DiscardRun struct{}
type DiscardRunSpec struct {
	RunID   string `json:"runId"`
	Comment string `json:"comment"`
}

func (c *DiscardRun) Name() string        { return "terraform.discardRun" }
func (c *DiscardRun) Label() string       { return "Discard Run" }
func (c *DiscardRun) Color() string       { return "red" }
func (c *DiscardRun) Icon() string        { return "x-circle" }
func (c *DiscardRun) Description() string { return "Discards a pending or planned run." }
func (c *DiscardRun) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "runId", Label: "Run ID", Type: configuration.FieldTypeString, Required: true},
		{Name: "comment", Label: "Comment", Type: configuration.FieldTypeString, Required: false},
	}
}
func (c *DiscardRun) Execute(ctx core.ExecutionContext) error {
	configAPI, err := ctx.Integration.GetConfig("apiToken")
	if err != nil {
		return err
	}
	configAddr, err := ctx.Integration.GetConfig("address")
	if err != nil {
		return err
	}

	client, err := NewClient(map[string]any{
		"apiToken": string(configAPI),
		"address":  string(configAddr),
	})
	if err != nil {
		return err
	}
	spec := DiscardRunSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	opts := tfe.RunDiscardOptions{}
	if spec.Comment != "" {
		opts.Comment = tfe.String(spec.Comment)
	}

	err = client.TFE.Runs.Discard(context.Background(), spec.RunID, opts)
	if err != nil {
		return fmt.Errorf("failed to discard run: %w", err)
	}
	return ctx.ExecutionState.Emit("default", "", []any{map[string]any{"runId": spec.RunID, "action": "discarded"}})
}

func (c *DiscardRun) Actions() []core.Action                                    { return nil }
func (c *DiscardRun) HandleAction(ctx core.ActionContext) error                 { return nil }
func (c *DiscardRun) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return 200, nil }
func (c *DiscardRun) Cancel(ctx core.ExecutionContext) error                    { return nil }
func (c *DiscardRun) Setup(ctx core.SetupContext) error                         { return nil }
func (c *DiscardRun) Cleanup(ctx core.SetupContext) error                       { return nil }
func (c *DiscardRun) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "default", Label: "Run Discarded", Description: "Emits when the run is successfully discarded"},
	}
}
func (c *DiscardRun) DefaultOutputChannel() core.OutputChannel { return core.DefaultOutputChannel }
func (c *DiscardRun) ExampleOutput() map[string]any {
	return map[string]any{
		"runId":  "run-xxxxxx",
		"action": "discarded",
	}
}
func (c *DiscardRun) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *DiscardRun) Documentation() string { return "" }

type OverridePolicy struct{}
type OverridePolicySpec struct {
	RunID     string `json:"runId"`
	Rationale string `json:"rationale"`
}

func (c *OverridePolicy) Name() string        { return "terraform.overridePolicy" }
func (c *OverridePolicy) Label() string       { return "Override Policy" }
func (c *OverridePolicy) Color() string       { return "orange" }
func (c *OverridePolicy) Icon() string        { return "shield" }
func (c *OverridePolicy) Description() string { return "Overrides a failed Sentinel policy block." }
func (c *OverridePolicy) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "runId", Label: "Run ID", Type: configuration.FieldTypeString, Required: true},
		{Name: "rationale", Label: "Override Rationale", Type: configuration.FieldTypeString, Required: true},
	}
}
func (c *OverridePolicy) Execute(ctx core.ExecutionContext) error {
	configAPI, err := ctx.Integration.GetConfig("apiToken")
	if err != nil {
		return err
	}
	configAddr, err := ctx.Integration.GetConfig("address")
	if err != nil {
		return err
	}

	client, err := NewClient(map[string]any{
		"apiToken": string(configAPI),
		"address":  string(configAddr),
	})
	if err != nil {
		return err
	}
	spec := OverridePolicySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	run, err := client.TFE.Runs.Read(context.Background(), spec.RunID)
	if err != nil {
		return fmt.Errorf("failed to read run: %w", err)
	}

	if run.Status != tfe.RunPolicyOverride {
		return fmt.Errorf("State Altered Externally: Run %s is currently '%s', not pending a policy override", spec.RunID, run.Status)
	}

	policyChecks, err := client.TFE.PolicyChecks.List(context.Background(), spec.RunID, nil)
	if err != nil {
		return fmt.Errorf("failed to list policy checks: %w", err)
	}

	for _, check := range policyChecks.Items {
		if check.Result != nil && check.Result.Result == false {
			_, err = client.TFE.PolicyChecks.Override(context.Background(), check.ID)
			if err != nil {
				return fmt.Errorf("failed to override policy check %s: %w", check.ID, err)
			}
		}
	}

	return ctx.ExecutionState.Emit("default", "", []any{map[string]any{"runId": spec.RunID, "action": "overridden"}})
}

func (c *OverridePolicy) Actions() []core.Action                                    { return nil }
func (c *OverridePolicy) HandleAction(ctx core.ActionContext) error                 { return nil }
func (c *OverridePolicy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return 200, nil }
func (c *OverridePolicy) Cancel(ctx core.ExecutionContext) error                    { return nil }
func (c *OverridePolicy) Setup(ctx core.SetupContext) error                         { return nil }
func (c *OverridePolicy) Cleanup(ctx core.SetupContext) error                       { return nil }
func (c *OverridePolicy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "default", Label: "Policy Overridden", Description: "Emits when the policy check is successfully overridden"},
	}
}
func (c *OverridePolicy) DefaultOutputChannel() core.OutputChannel { return core.DefaultOutputChannel }
func (c *OverridePolicy) ExampleOutput() map[string]any {
	return map[string]any{
		"runId":  "run-xxxxxx",
		"action": "overridden",
	}
}
func (c *OverridePolicy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *OverridePolicy) Documentation() string { return "" }

type ReadRun struct{}
type ReadRunSpec struct {
	RunID string `json:"runId"`
}

func (c *ReadRun) Name() string  { return "terraform.readRun" }
func (c *ReadRun) Label() string { return "Read Run Details" }
func (c *ReadRun) Description() string {
	return "Retrieves comprehensive details and status about a run."
}
func (c *ReadRun) Icon() string  { return "info" }
func (c *ReadRun) Color() string { return "gray" }
func (c *ReadRun) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "runId", Label: "Run ID", Type: configuration.FieldTypeString, Required: true},
	}
}
func (c *ReadRun) Execute(ctx core.ExecutionContext) error {
	configAPI, err := ctx.Integration.GetConfig("apiToken")
	if err != nil {
		return err
	}
	configAddr, err := ctx.Integration.GetConfig("address")
	if err != nil {
		return err
	}

	client, err := NewClient(map[string]any{
		"apiToken": string(configAPI),
		"address":  string(configAddr),
	})
	if err != nil {
		return err
	}
	spec := ReadRunSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	run, err := client.TFE.Runs.Read(context.Background(), spec.RunID)
	if err != nil {
		return fmt.Errorf("failed to read run: %w", err)
	}

	return ctx.ExecutionState.Emit("default", "", []any{
		map[string]any{
			"runId":     run.ID,
			"status":    run.Status,
			"message":   run.Message,
			"createdAt": run.CreatedAt,
		},
	})
}

func (c *ReadRun) Actions() []core.Action                                    { return nil }
func (c *ReadRun) HandleAction(ctx core.ActionContext) error                 { return nil }
func (c *ReadRun) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return 200, nil }
func (c *ReadRun) Cancel(ctx core.ExecutionContext) error                    { return nil }
func (c *ReadRun) Cleanup(ctx core.SetupContext) error                       { return nil }
func (c *ReadRun) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "default", Label: "Run Details", Description: "Emits the run details"},
	}
}
func (c *ReadRun) DefaultOutputChannel() core.OutputChannel { return core.DefaultOutputChannel }
func (c *ReadRun) Setup(ctx core.SetupContext) error        { return nil }
func (c *ReadRun) ExampleOutput() map[string]any {
	return map[string]any{
		"runId":     "run-xxxxxx",
		"status":    "planned",
		"message":   "Queued manually",
		"createdAt": "2024-01-01T12:00:00Z",
	}
}
func (c *ReadRun) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *ReadRun) Documentation() string { return "" }
