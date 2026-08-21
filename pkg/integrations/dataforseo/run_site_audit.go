package dataforseo

import (
	"fmt"
	"strings"
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

	// runSiteAuditPagesLimit is the max number of pages fetched per GetPages
	// call. If the number of pages returned equals this limit, there may be
	// more pages on the site that DataForSEO didn't return (no pagination is
	// implemented), so the result is marked as truncated.
	runSiteAuditPagesLimit = 1000
)

// runSiteAuditIssuePagesEmitCap bounds how many issue pages are included in
// the emitted payload, so a large site with many issues doesn't blow past
// the execution engine's per-event payload size limit. It's a var (not a
// const) so tests can override it to exercise the capping logic without
// needing a 100+ item fixture.
var runSiteAuditIssuePagesEmitCap = 100

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
- **Scheduled monitoring**: Run on a schedule and branch into issue-handling automation when problems appear

## Output Channels

- **Issues found**: at least one crawled page failed one of the four checks below
- **Clean**: every crawled page passed all four checks

## Polling

After starting the audit, the component polls DataForSEO every 5 minutes for up to 72 attempts (~6 hours). If the
audit hasn't finished within that window, the execution fails with an "audit_timeout" error.

## What counts as an "issue"

A page is considered to have an issue if any of the following on-page checks fail:

- **Broken links**: the page contains one or more broken outgoing links
- **Duplicate title**: the page's title tag duplicates another page's title
- **Duplicate meta description**: the page's meta description duplicates another page's
- **Broken page response**: the page itself returned a broken (non-2xx) HTTP response

## Coverage limits

Only the first 1000 crawled pages are fetched per audit. If the site has more pages than that, the emitted payload
includes a "truncated" flag so downstream automation can distinguish a genuinely clean site from one where coverage
was only partial. The "pages" list in the payload is also capped to the first 100 issue pages; the true total is
always available in "issuePageCount".`
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
	spec := RunSiteAuditSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.Domain) == "" {
		return fmt.Errorf("domain is required")
	}

	if spec.MaxCrawlPages < 1 {
		return fmt.Errorf("maxCrawlPages must be at least 1")
	}

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
		ctx.Logger.Warnf("failed to get DataForSEO audit summary for task %s: %v", metadata.TaskID, err)
		metadata.PollAttempt++
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}
		return ctx.Requests.ScheduleActionCall(RunSiteAuditPollAction, map[string]any{}, RunSiteAuditPollInterval)
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
	allPages, err := client.GetPages(metadata.TaskID, runSiteAuditPagesLimit)
	if err != nil {
		return fmt.Errorf("failed to fetch audit pages: %w", err)
	}

	issuePages := make([]PageResult, 0, len(allPages))
	for _, page := range allPages {
		if page.Checks.HasIssue() {
			issuePages = append(issuePages, page)
		}
	}

	// The "issues found" vs "clean" decision is based on the true issue
	// count, not the (possibly capped) list of pages emitted below.
	channel := RunSiteAuditCleanChannel
	if len(issuePages) > 0 {
		channel = RunSiteAuditIssuesChannel
	}

	emittedPages := issuePages
	if len(emittedPages) > runSiteAuditIssuePagesEmitCap {
		emittedPages = emittedPages[:runSiteAuditIssuePagesEmitCap]
	}

	// If DataForSEO returned exactly as many pages as we asked for, there may
	// be more pages on the site that we never fetched (no pagination is
	// implemented), so this can't be treated as a fully-covered crawl.
	truncated := len(allPages) >= runSiteAuditPagesLimit

	return ctx.ExecutionState.Emit(channel, "dataforseo.audit.finished", []any{
		map[string]any{
			"taskId":         metadata.TaskID,
			"pages":          emittedPages,
			"truncated":      truncated,
			"issuePageCount": len(issuePages),
		},
	})
}
