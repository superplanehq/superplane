package claude

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	GetDailyUsageDataPayloadType = "claude.getDailyUsageData.result"
	maxUsageReportRangeDays      = 31
)

type GetDailyUsageData struct{}

type GetDailyUsageDataSpec struct {
	StartDate string `json:"startDate" mapstructure:"startDate"`
	EndDate   string `json:"endDate" mapstructure:"endDate"`
}

type Period struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

// ModelUsage is a per-model token rollup from the Messages usage report.
type ModelUsage struct {
	Model               string `json:"model"`
	InputTokens         int64  `json:"inputTokens"`
	OutputTokens        int64  `json:"outputTokens"`
	CacheReadTokens     int64  `json:"cacheReadTokens"`
	CacheCreationTokens int64  `json:"cacheCreationTokens"`
}

// MessagesSummary rolls up raw API/SDK usage. The Messages usage report has no cost or
// request-count field, only token counts.
type MessagesSummary struct {
	InputTokens         int64        `json:"inputTokens"`
	OutputTokens        int64        `json:"outputTokens"`
	CacheReadTokens     int64        `json:"cacheReadTokens"`
	CacheCreationTokens int64        `json:"cacheCreationTokens"`
	WebSearchRequests   int64        `json:"webSearchRequests"`
	ByModel             []ModelUsage `json:"byModel"`
}

// ModelCost is a per-model token and cost rollup from the Claude Code usage report.
type ModelCost struct {
	Model            string  `json:"model"`
	InputTokens      int64   `json:"inputTokens"`
	OutputTokens     int64   `json:"outputTokens"`
	EstimatedCostUsd float64 `json:"estimatedCostUsd"`
}

// ActorSummary is a per-user/per-API-key productivity rollup from the Claude Code usage report.
type ActorSummary struct {
	Actor        string `json:"actor"`
	Type         string `json:"type"`
	Sessions     int64  `json:"sessions"`
	LinesAdded   int64  `json:"linesAdded"`
	LinesRemoved int64  `json:"linesRemoved"`
	Commits      int64  `json:"commits"`
	PullRequests int64  `json:"pullRequests"`
}

// ClaudeCodeSummary rolls up Claude Code productivity metrics and cost.
type ClaudeCodeSummary struct {
	Sessions            int64          `json:"sessions"`
	LinesAdded          int64          `json:"linesAdded"`
	LinesRemoved        int64          `json:"linesRemoved"`
	Commits             int64          `json:"commits"`
	PullRequests        int64          `json:"pullRequests"`
	ToolActionsAccepted int64          `json:"toolActionsAccepted"`
	ToolActionsRejected int64          `json:"toolActionsRejected"`
	EstimatedCostUsd    float64        `json:"estimatedCostUsd"`
	ByModel             []ModelCost    `json:"byModel"`
	ByActor             []ActorSummary `json:"byActor"`
}

// DailyUsage is an org-wide, per-calendar-day rollup combining both reports, for trend charts.
type DailyUsage struct {
	Date                 string  `json:"date"`
	MessagesInputTokens  int64   `json:"messagesInputTokens"`
	MessagesOutputTokens int64   `json:"messagesOutputTokens"`
	CodeSessions         int64   `json:"codeSessions"`
	CodeLinesAdded       int64   `json:"codeLinesAdded"`
	CodeLinesRemoved     int64   `json:"codeLinesRemoved"`
	CodeCommits          int64   `json:"codeCommits"`
	CodePullRequests     int64   `json:"codePullRequests"`
	CodeEstimatedCostUsd float64 `json:"codeEstimatedCostUsd"`
}

type GetDailyUsageDataOutput struct {
	Period     Period            `json:"period"`
	Messages   MessagesSummary   `json:"messages"`
	ClaudeCode ClaudeCodeSummary `json:"claudeCode"`
	Daily      []DailyUsage      `json:"daily"`
}

func (c *GetDailyUsageData) Name() string {
	return "claude.getDailyUsageData"
}

func (c *GetDailyUsageData) Label() string {
	return "Get Daily Usage Data"
}

func (c *GetDailyUsageData) Description() string {
	return "Fetches daily Messages and Claude Code usage metrics from Anthropic's Admin API."
}

func (c *GetDailyUsageData) Documentation() string {
	return `The Get Daily Usage Data component fetches usage metrics from Anthropic's Admin API,
combining raw API/SDK token usage with Claude Code productivity metrics.

## Use Cases

- **Usage reporting**: Track token consumption and Claude Code productivity across the org
- **Cost tracking**: Monitor estimated Claude Code spend by model
- **Analytics dashboards**: Build custom dashboards combining both data sources

## How It Works

1. Fetches token usage for the date range from the Messages usage report
2. Fetches productivity metrics (sessions, lines of code, commits, PRs, tool actions) for the
   date range from the Claude Code usage report
3. Aggregates both into rollup totals, per-model/per-actor breakdowns, and a daily time series

## Configuration

- **Start Date**: Start of the date range (YYYY-MM-DD format, defaults to 7 days ago)
- **End Date**: End of the date range (YYYY-MM-DD format, defaults to today)

## Output

- **messages**: Token usage totals and per-model breakdown (no cost data; Messages usage has
  no dollar figure attached)
- **claudeCode**: Sessions, lines of code, commits, pull requests, tool action accept/reject
  counts, estimated cost, and per-model/per-actor breakdowns
- **daily**: Org-wide per-day rollup combining both reports, for trend charts

## Notes

- Requires an Admin API key configured in the integration, separate from the regular API key
- The date range cannot exceed 31 days
- The Claude Code usage report only accepts one day per call, so the component issues one
  request per day in the range`
}

func (c *GetDailyUsageData) Icon() string {
	return "bar-chart"
}

func (c *GetDailyUsageData) Color() string {
	return "#D97757"
}

func (c *GetDailyUsageData) ExampleOutput() map[string]any {
	return map[string]any{
		"data": GetDailyUsageDataOutput{
			Period: Period{StartDate: "2026-06-26", EndDate: "2026-07-03"},
			Messages: MessagesSummary{
				InputTokens:         1250000,
				OutputTokens:        380000,
				CacheReadTokens:     900000,
				CacheCreationTokens: 60000,
				WebSearchRequests:   42,
				ByModel: []ModelUsage{
					{Model: "claude-sonnet-5", InputTokens: 1250000, OutputTokens: 380000, CacheReadTokens: 900000, CacheCreationTokens: 60000},
				},
			},
			ClaudeCode: ClaudeCodeSummary{
				Sessions:            87,
				LinesAdded:          15400,
				LinesRemoved:        6200,
				Commits:             63,
				PullRequests:        21,
				ToolActionsAccepted: 512,
				ToolActionsRejected: 34,
				EstimatedCostUsd:    42.17,
				ByModel: []ModelCost{
					{Model: "claude-sonnet-5", InputTokens: 45230, OutputTokens: 12450, EstimatedCostUsd: 1.86},
				},
				ByActor: []ActorSummary{
					{Actor: "developer@company.com", Type: "user_actor", Sessions: 12, LinesAdded: 2200, LinesRemoved: 900, Commits: 9, PullRequests: 3},
				},
			},
			Daily: []DailyUsage{
				{Date: "2026-06-26", MessagesInputTokens: 178571, MessagesOutputTokens: 54285, CodeSessions: 12, CodeLinesAdded: 2200, CodeLinesRemoved: 900, CodeCommits: 9, CodePullRequests: 3, CodeEstimatedCostUsd: 6.02},
			},
		},
		"timestamp": "2026-07-03T19:29:35.841265352Z",
		"type":      GetDailyUsageDataPayloadType,
	}
}

func (c *GetDailyUsageData) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetDailyUsageData) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "startDate",
			Label:       "Start Date",
			Type:        configuration.FieldTypeString,
			Description: "YYYY-MM-DD (Defaults to 7 days ago)",
			Required:    false,
		},
		{
			Name:        "endDate",
			Label:       "End Date",
			Type:        configuration.FieldTypeString,
			Description: "YYYY-MM-DD (Defaults to today)",
			Required:    false,
		},
	}
}

func (c *GetDailyUsageData) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *GetDailyUsageData) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetDailyUsageData) Execute(ctx core.ExecutionContext) error {
	spec := GetDailyUsageDataSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	startDate, endDate, err := resolveDateRange(spec)
	if err != nil {
		return err
	}

	if endDate.Sub(startDate) > (maxUsageReportRangeDays-1)*24*time.Hour {
		return fmt.Errorf("date range cannot exceed %d days", maxUsageReportRangeDays)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create claude client: %w", err)
	}

	if client.AdminKey == "" {
		return fmt.Errorf("admin API key is not configured in the integration")
	}

	days := int(endDate.Sub(startDate).Hours()/24) + 1

	ctx.Logger.Infof("Fetching Claude usage data from %s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	messageBuckets, err := client.GetMessagesUsageReport(
		startDate.Format(time.RFC3339),
		endDate.AddDate(0, 0, 1).Format(time.RFC3339),
		days,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch messages usage report: %w", err)
	}

	codeRecords, err := client.GetClaudeCodeUsageReport(startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to fetch claude code usage report: %w", err)
	}

	messagesSummary, messagesDaily := aggregateMessages(messageBuckets)
	codeSummary, codeDaily := aggregateClaudeCode(codeRecords)

	output := GetDailyUsageDataOutput{
		Period: Period{
			StartDate: startDate.Format("2006-01-02"),
			EndDate:   endDate.Format("2006-01-02"),
		},
		Messages:   messagesSummary,
		ClaudeCode: codeSummary,
		Daily:      mergeDaily(messagesDaily, codeDaily),
	}

	ctx.Logger.Infof("Retrieved usage data: %d message buckets, %d claude code records", len(messageBuckets), len(codeRecords))

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetDailyUsageDataPayloadType, []any{output})
}

func (c *GetDailyUsageData) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *GetDailyUsageData) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetDailyUsageData) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetDailyUsageData) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetDailyUsageData) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func resolveDateRange(spec GetDailyUsageDataSpec) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	startDate, err := parseSpecDate(spec.StartDate, startOfToday.AddDate(0, 0, -7))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start date format (expected YYYY-MM-DD): %w", err)
	}

	endDate, err := parseSpecDate(spec.EndDate, startOfToday)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end date format (expected YYYY-MM-DD): %w", err)
	}

	if startDate.After(endDate) {
		return time.Time{}, time.Time{}, fmt.Errorf("start date must be before end date")
	}

	return startDate, endDate, nil
}

func parseSpecDate(value string, fallback time.Time) (time.Time, error) {
	if value == "" {
		return fallback, nil
	}
	return time.Parse("2006-01-02", value)
}

func dateFromReportValue(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return t.Format("2006-01-02")
}

func dailyRow(daily map[string]*DailyUsage, date string) *DailyUsage {
	row, ok := daily[date]
	if !ok {
		row = &DailyUsage{Date: date}
		daily[date] = row
	}
	return row
}

func aggregateMessages(buckets []MessagesUsageBucket) (MessagesSummary, map[string]*DailyUsage) {
	summary := MessagesSummary{}
	byModel := map[string]*ModelUsage{}
	daily := map[string]*DailyUsage{}

	for _, bucket := range buckets {
		day := dailyRow(daily, dateFromReportValue(bucket.StartingAt))
		for _, result := range bucket.Results {
			applyMessagesResult(&summary, byModel, day, result)
		}
	}

	summary.ByModel = modelUsageList(byModel)
	return summary, daily
}

func applyMessagesResult(summary *MessagesSummary, byModel map[string]*ModelUsage, day *DailyUsage, result MessagesUsageResult) {
	cacheCreation := result.CacheCreation.Ephemeral1hInputTokens + result.CacheCreation.Ephemeral5mInputTokens

	summary.InputTokens += result.UncachedInputTokens
	summary.OutputTokens += result.OutputTokens
	summary.CacheReadTokens += result.CacheReadInputTokens
	summary.CacheCreationTokens += cacheCreation
	summary.WebSearchRequests += result.ServerToolUse.WebSearchRequests

	day.MessagesInputTokens += result.UncachedInputTokens
	day.MessagesOutputTokens += result.OutputTokens

	model := result.Model
	if model == "" {
		model = "unknown"
	}
	m, ok := byModel[model]
	if !ok {
		m = &ModelUsage{Model: model}
		byModel[model] = m
	}
	m.InputTokens += result.UncachedInputTokens
	m.OutputTokens += result.OutputTokens
	m.CacheReadTokens += result.CacheReadInputTokens
	m.CacheCreationTokens += cacheCreation
}

func modelUsageList(byModel map[string]*ModelUsage) []ModelUsage {
	list := make([]ModelUsage, 0, len(byModel))
	for _, m := range byModel {
		list = append(list, *m)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Model < list[j].Model })
	return list
}

func aggregateClaudeCode(records []ClaudeCodeUsageRecord) (ClaudeCodeSummary, map[string]*DailyUsage) {
	summary := ClaudeCodeSummary{}
	byModel := map[string]*ModelCost{}
	byActor := map[string]*ActorSummary{}
	daily := map[string]*DailyUsage{}

	for _, record := range records {
		day := dailyRow(daily, dateFromReportValue(record.Date))
		applyCoreMetrics(&summary, day, record.CoreMetrics)
		applyActor(byActor, record.Actor, record.CoreMetrics)
		applyModelBreakdown(&summary, byModel, day, record.ModelBreakdown)
		applyToolActions(&summary, record.ToolActions)
	}

	summary.ByModel = modelCostList(byModel)
	summary.ByActor = actorSummaryList(byActor)
	return summary, daily
}

func applyCoreMetrics(summary *ClaudeCodeSummary, day *DailyUsage, m ClaudeCodeCoreMetrics) {
	summary.Sessions += m.NumSessions
	summary.LinesAdded += m.LinesOfCode.Added
	summary.LinesRemoved += m.LinesOfCode.Removed
	summary.Commits += m.CommitsByClaudeCode
	summary.PullRequests += m.PullRequestsByClaudeCode

	day.CodeSessions += m.NumSessions
	day.CodeLinesAdded += m.LinesOfCode.Added
	day.CodeLinesRemoved += m.LinesOfCode.Removed
	day.CodeCommits += m.CommitsByClaudeCode
	day.CodePullRequests += m.PullRequestsByClaudeCode
}

func applyActor(byActor map[string]*ActorSummary, actor ClaudeCodeActor, m ClaudeCodeCoreMetrics) {
	name := actor.Name()
	if name == "" {
		name = "unknown"
	}
	a, ok := byActor[name]
	if !ok {
		a = &ActorSummary{Actor: name, Type: actor.Type}
		byActor[name] = a
	}
	a.Sessions += m.NumSessions
	a.LinesAdded += m.LinesOfCode.Added
	a.LinesRemoved += m.LinesOfCode.Removed
	a.Commits += m.CommitsByClaudeCode
	a.PullRequests += m.PullRequestsByClaudeCode
}

func applyModelBreakdown(summary *ClaudeCodeSummary, byModel map[string]*ModelCost, day *DailyUsage, breakdown []ClaudeCodeModelBreakdown) {
	for _, mb := range breakdown {
		// amount is in minor currency units (e.g. cents for USD).
		costUsd := mb.EstimatedCost.Amount / 100
		summary.EstimatedCostUsd += costUsd
		day.CodeEstimatedCostUsd += costUsd

		model := mb.Model
		if model == "" {
			model = "unknown"
		}
		m, ok := byModel[model]
		if !ok {
			m = &ModelCost{Model: model}
			byModel[model] = m
		}
		m.InputTokens += mb.Tokens.Input
		m.OutputTokens += mb.Tokens.Output
		m.EstimatedCostUsd += costUsd
	}
}

func applyToolActions(summary *ClaudeCodeSummary, actions map[string]ClaudeCodeToolAction) {
	for _, action := range actions {
		summary.ToolActionsAccepted += action.Accepted
		summary.ToolActionsRejected += action.Rejected
	}
}

func modelCostList(byModel map[string]*ModelCost) []ModelCost {
	list := make([]ModelCost, 0, len(byModel))
	for _, m := range byModel {
		list = append(list, *m)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Model < list[j].Model })
	return list
}

func actorSummaryList(byActor map[string]*ActorSummary) []ActorSummary {
	list := make([]ActorSummary, 0, len(byActor))
	for _, a := range byActor {
		list = append(list, *a)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Actor < list[j].Actor })
	return list
}

func mergeDaily(messagesDaily, codeDaily map[string]*DailyUsage) []DailyUsage {
	merged := map[string]*DailyUsage{}
	for date, row := range messagesDaily {
		rowCopy := *row
		merged[date] = &rowCopy
	}
	for date, row := range codeDaily {
		existing, ok := merged[date]
		if !ok {
			rowCopy := *row
			merged[date] = &rowCopy
			continue
		}
		existing.CodeSessions += row.CodeSessions
		existing.CodeLinesAdded += row.CodeLinesAdded
		existing.CodeLinesRemoved += row.CodeLinesRemoved
		existing.CodeCommits += row.CodeCommits
		existing.CodePullRequests += row.CodePullRequests
		existing.CodeEstimatedCostUsd += row.CodeEstimatedCostUsd
	}

	list := make([]DailyUsage, 0, len(merged))
	for _, row := range merged {
		list = append(list, *row)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Date < list[j].Date })
	return list
}
