package dataforseo

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	RunSiteAuditIssuesChannel = "issues found"
	RunSiteAuditCleanChannel  = "clean"

	RunSiteAuditPollInterval    = 5 * time.Minute
	RunSiteAuditPollAction      = "poll"
	RunSiteAuditMaxPollAttempts = 72
	RunSiteAuditKVTaskID        = "task_id"
)

type RunSiteAudit struct{}

type RunSiteAuditSpec struct {
	Domain        string `json:"domain" mapstructure:"domain"`
	MaxCrawlPages int    `json:"maxCrawlPages" mapstructure:"maxCrawlPages"`
}

type RunSiteAuditExecutionMetadata struct {
	TaskID      string `json:"taskId" mapstructure:"taskId"`
	PollAttempt int    `json:"pollAttempt" mapstructure:"pollAttempt"`
}

func (r *RunSiteAudit) Name() string {
	return "dataforseo.runSiteAudit"
}

func (r *RunSiteAudit) Label() string {
	return "Run Site Audit"
}

func (r *RunSiteAudit) Description() string {
	return "Run a DataForSEO OnPage site audit and wait for completion"
}

func (r *RunSiteAudit) Documentation() string {
	return `The Run Site Audit component crawls a domain with DataForSEO's OnPage API and waits for the audit to finish.

## Use Cases

- **SEO guardrail**: Gate a deploy or release step on the site staying free of on-page SEO regressions
- **Scheduled monitoring**: Run on a schedule and branch into issue-handling automation when problems appear`
}

func (r *RunSiteAudit) Icon() string {
	return "search"
}

func (r *RunSiteAudit) Color() string {
	return "blue"
}

func (r *RunSiteAudit) ExampleOutput() map[string]any {
	return map[string]any{
		"taskId": "07131248-1535-0216-1000-17384017ad04",
		"pages": []map[string]any{
			{"url": "https://freehire.me/jobs/example", "checks": map[string]any{"duplicate_title": true}},
		},
	}
}

func (r *RunSiteAudit) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: RunSiteAuditIssuesChannel, Label: "Issues found"},
		{Name: RunSiteAuditCleanChannel, Label: "Clean"},
	}
}

func (r *RunSiteAudit) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "domain",
			Label:    "Domain",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "maxCrawlPages",
			Label:    "Max Crawl Pages",
			Type:     configuration.FieldTypeNumber,
			Required: true,
			Default:  100,
		},
	}
}

func (r *RunSiteAudit) Setup(ctx core.SetupContext) error {
	return nil
}

func (r *RunSiteAudit) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (r *RunSiteAudit) Execute(ctx core.ExecutionContext) error {
	spec := RunSiteAuditSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	taskID, err := client.PostAudit(spec.Domain, spec.MaxCrawlPages)
	if err != nil {
		return fmt.Errorf("failed to start audit: %w", err)
	}

	metadata := RunSiteAuditExecutionMetadata{TaskID: taskID, PollAttempt: 0}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	if err := ctx.ExecutionState.SetKV(RunSiteAuditKVTaskID, taskID); err != nil {
		return err
	}

	ctx.Logger.Infof("Started DataForSEO audit %s for %s", taskID, spec.Domain)
	return ctx.Requests.ScheduleActionCall(RunSiteAuditPollAction, map[string]any{}, RunSiteAuditPollInterval)
}

func (r *RunSiteAudit) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (r *RunSiteAudit) Hooks() []core.Hook {
	return []core.Hook{
		{Name: RunSiteAuditPollAction, Type: core.HookTypeInternal},
	}
}

func (r *RunSiteAudit) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case RunSiteAuditPollAction:
		return r.poll(ctx)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (r *RunSiteAudit) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil // DataForSEO OnPage has no webhook path in v1; polling only.
}

func (r *RunSiteAudit) Cancel(ctx core.ExecutionContext) error {
	ctx.Logger.Info("DataForSEO OnPage tasks cannot be cancelled server-side; nothing to do")
	return nil
}

func (r *RunSiteAudit) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := RunSiteAuditExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.TaskID == "" {
		return fmt.Errorf("task id is missing from execution metadata")
	}

	if metadata.PollAttempt >= RunSiteAuditMaxPollAttempts {
		return ctx.ExecutionState.Fail(
			"audit_timeout",
			fmt.Sprintf("DataForSEO audit did not finish within %d poll attempts", RunSiteAuditMaxPollAttempts),
		)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	crawlProgress, err := client.GetSummary(metadata.TaskID)
	if err != nil {
		return err
	}

	if crawlProgress != "finished" {
		metadata.PollAttempt++
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}
		return ctx.Requests.ScheduleActionCall(RunSiteAuditPollAction, map[string]any{}, RunSiteAuditPollInterval)
	}

	return r.resolve(ctx, client, metadata)
}

func (r *RunSiteAudit) resolve(ctx core.ActionHookContext, client *Client, metadata RunSiteAuditExecutionMetadata) error {
	return ctx.ExecutionState.Pass()
}
