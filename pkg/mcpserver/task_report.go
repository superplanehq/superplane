package mcpserver

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func (rt *Runtime) getTaskReport(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	factory, err := rt.beginWorkspaceCall(ctx, claims, args, "work_orders:read")
	if err != nil {
		return ToolResult{}, err
	}
	task, err := stringArg(args, "task")
	if err != nil {
		return ToolResult{}, err
	}
	limit, err := taskRunPageLimit(args["limit"])
	if err != nil {
		return ToolResult{}, err
	}
	before, err := taskRunCursor(args["before"])
	if err != nil {
		return ToolResult{}, err
	}

	report, err := factories.BuildTaskReport(toolContext(ctx, claims), claims.OrgID.String(), factory.ID.String(), task, models.TaskRunHistoryQuery{
		Limit:  limit,
		Before: before,
	})
	if err != nil {
		return ToolResult{}, actionError(err)
	}
	return TextResult(mustJSON(taskReportJSON(report, newTaskLinks(rt.BaseURL, report.Factory)))), nil
}

func taskRunPageLimit(raw any) (int, error) {
	if raw == nil {
		return models.DefaultTaskRunHistoryLimit, nil
	}
	limit, ok := wholeNumber(raw)
	if !ok || limit < 1 {
		return 0, ToolError("limit must be from 1 to 200")
	}
	if limit > models.MaxTaskRunHistoryLimit {
		return models.MaxTaskRunHistoryLimit, nil
	}
	return limit, nil
}

func taskRunCursor(raw any) (*models.TaskRunCursor, error) {
	if raw == nil {
		return nil, nil
	}
	text, ok := raw.(string)
	if !ok {
		return nil, ToolError("before must be a run cursor")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	stamp, idText, ok := strings.Cut(text, "|")
	if !ok {
		return nil, ToolError("before must be a run cursor")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, stamp)
	if err != nil {
		return nil, ToolError("before must be a run cursor")
	}
	id, err := uuid.Parse(idText)
	if err != nil {
		return nil, ToolError("before must be a run cursor")
	}
	return &models.TaskRunCursor{CreatedAt: createdAt, ID: id}, nil
}

func formatTaskRunCursor(createdAt time.Time, id uuid.UUID) string {
	return createdAt.UTC().Format(time.RFC3339Nano) + "|" + id.String()
}

func wholeNumber(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) || value != math.Trunc(value) {
			return 0, false
		}
		return int(value), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

type taskReportDocument struct {
	Factory            reportFactory        `json:"factory"`
	Task               reportTask           `json:"task"`
	Usage              reportUsage          `json:"usage"`
	UsageByModel       []reportModelUsage   `json:"usage_by_model"`
	UsageByMachineType []reportMachineUsage `json:"usage_by_machine_type"`
	PullRequests       []reportPullRequest  `json:"pull_requests"`
	LineDispatches     []reportLineDispatch `json:"line_dispatches"`
	Runs               []reportRun          `json:"runs"`
	RunCounts          []reportRunCount     `json:"run_counts"`
	HasNextRunPage     bool                 `json:"has_next_run_page"`
	NextRunBefore      string               `json:"next_run_before,omitempty"`
}

type reportFactory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

type reportTask struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Number      int64  `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Result      string `json:"result"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	SourceURL   string `json:"source_url"`
	SourceLabel string `json:"source_label"`
	URL         string `json:"url"`
}

type reportUsage struct {
	Recorded          bool   `json:"recorded"`
	TotalTokens       *int64 `json:"total_tokens,omitempty"`
	TotalCostCents    *int64 `json:"total_cost_cents,omitempty"`
	VMDurationSeconds *int64 `json:"vm_duration_seconds,omitempty"`
}

type reportModelUsage struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	TotalTokens int64  `json:"total_tokens"`
	CostCents   int64  `json:"cost_cents"`
}

type reportMachineUsage struct {
	MachineType       string `json:"machine_type"`
	VMDurationSeconds int64  `json:"vm_duration_seconds"`
	CostCents         int64  `json:"cost_cents"`
}

type reportPullRequest struct {
	URL        string           `json:"url"`
	Repository string           `json:"repository"`
	Title      string           `json:"title"`
	State      string           `json:"state"`
	Activities []reportActivity `json:"activities"`
}

type reportActivity struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	State       string `json:"state,omitempty"`
	RunID       string `json:"run_id,omitempty"`
}

type reportLineDispatch struct {
	ID         string       `json:"id"`
	Line       string       `json:"line"`
	State      string       `json:"state"`
	Result     string       `json:"result"`
	Model      string       `json:"model,omitempty"`
	CreatedAt  string       `json:"created_at"`
	FinishedAt string       `json:"finished_at,omitempty"`
	Steps      []reportStep `json:"steps"`
}

type reportStep struct {
	Name              string   `json:"name"`
	Index             int32    `json:"index"`
	State             string   `json:"state"`
	Result            string   `json:"result"`
	Models            []string `json:"models,omitempty"`
	TotalTokens       int64    `json:"total_tokens,omitempty"`
	CostCents         int64    `json:"cost_cents,omitempty"`
	VMDurationSeconds int64    `json:"vm_duration_seconds,omitempty"`
	RunID             string   `json:"run_id,omitempty"`
}

type reportRun struct {
	ID                  string `json:"id"`
	AutomationID        string `json:"automation_id"`
	AutomationName      string `json:"automation_name"`
	ParentRunID         string `json:"parent_run_id,omitempty"`
	State               string `json:"state"`
	Result              string `json:"result"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at,omitempty"`
	FinishedAt          string `json:"finished_at,omitempty"`
	ActivityTitle       string `json:"activity_title,omitempty"`
	ActivityDescription string `json:"activity_description,omitempty"`
	UsageRecorded       bool   `json:"usage_recorded"`
	TotalTokens         *int64 `json:"total_tokens,omitempty"`
	TotalCostCents      *int64 `json:"total_cost_cents,omitempty"`
	VMDurationSeconds   *int64 `json:"vm_duration_seconds,omitempty"`
}

type reportRunCount struct {
	AutomationID   string `json:"automation_id"`
	AutomationName string `json:"automation_name"`
	Result         string `json:"result"`
	Count          int64  `json:"count"`
}

func taskReportJSON(report *factories.TaskReport, links taskLinks) taskReportDocument {
	order := report.Order
	document := taskReportDocument{
		Factory: reportFactory{
			ID:   report.Factory.ID.String(),
			Name: report.Factory.Name,
			Key:  report.Factory.Key,
		},
		Task:               reportTaskJSON(order, links),
		Usage:              reportUsageJSON(report.UsageRecorded, order),
		UsageByModel:       reportModelUsageJSON(report.UsageRecorded, order.GetUsageByModel()),
		UsageByMachineType: reportMachineUsageJSON(report.UsageRecorded, order.GetUsageByMachineType()),
		PullRequests:       reportPullRequestsJSON(order.GetPullRequests()),
		LineDispatches:     reportDispatchesJSON(order.GetLineDispatches()),
		Runs:               reportRunsJSON(report.Runs.Runs),
		RunCounts:          reportRunCountsJSON(report.Runs.Counts),
		HasNextRunPage:     report.Runs.HasNextPage,
	}
	if report.Runs.HasNextPage && len(report.Runs.Runs) > 0 {
		last := report.Runs.Runs[len(report.Runs.Runs)-1]
		document.NextRunBefore = formatTaskRunCursor(last.CreatedAt, last.ID)
	}
	return document
}

func reportTaskJSON(order *pb.WorkOrder, links taskLinks) reportTask {
	task := reportTask{
		ID:          order.GetId(),
		Key:         order.GetKey(),
		Number:      order.GetNumber(),
		Title:       order.GetTitle(),
		Description: order.GetDescription(),
		State:       protoStateName(order.GetState()),
		Result:      enumToken(order.GetResult().String()),
		CreatedAt:   timestampText(order.GetCreatedAt().AsTime()),
		UpdatedAt:   timestampText(order.GetUpdatedAt().AsTime()),
		URL:         links.url(order.GetNumber()),
	}
	if origin := order.GetOrigin(); origin != nil {
		task.SourceURL = origin.GetUrl()
		task.SourceLabel = origin.GetLabel()
	}
	return task
}

func reportUsageJSON(recorded bool, order *pb.WorkOrder) reportUsage {
	usage := reportUsage{Recorded: recorded}
	if !recorded {
		return usage
	}
	usage.TotalTokens = int64Ptr(order.GetTotalTokens())
	usage.TotalCostCents = int64Ptr(order.GetTotalCostCents())
	usage.VMDurationSeconds = int64Ptr(order.GetTotalDurationSeconds())
	return usage
}

func reportModelUsageJSON(recorded bool, rows []*pb.UsageByModel) []reportModelUsage {
	if !recorded {
		return []reportModelUsage{}
	}
	out := make([]reportModelUsage, 0, len(rows))
	for _, row := range rows {
		out = append(out, reportModelUsage{
			Provider:    row.GetProvider(),
			Model:       row.GetModel(),
			TotalTokens: row.GetTotalTokens(),
			CostCents:   row.GetCostCents(),
		})
	}
	return out
}

func reportMachineUsageJSON(recorded bool, rows []*pb.UsageByMachineType) []reportMachineUsage {
	if !recorded {
		return []reportMachineUsage{}
	}
	out := make([]reportMachineUsage, 0, len(rows))
	for _, row := range rows {
		out = append(out, reportMachineUsage{
			MachineType:       row.GetMachineType(),
			VMDurationSeconds: row.GetDurationSeconds(),
			CostCents:         row.GetCostCents(),
		})
	}
	return out
}

func reportPullRequestsJSON(pullRequests []*pb.FactoryPullRequest) []reportPullRequest {
	out := make([]reportPullRequest, 0, len(pullRequests))
	for _, pullRequest := range pullRequests {
		activities := make([]reportActivity, 0, len(pullRequest.GetActivities()))
		for _, activity := range pullRequest.GetActivities() {
			item := reportActivity{
				Title:       activity.GetTitle(),
				Description: activity.GetDescription(),
				State:       activity.GetState(),
			}
			if run := activity.GetRun(); run != nil {
				item.RunID = run.GetId()
			}
			activities = append(activities, item)
		}
		out = append(out, reportPullRequest{
			URL:        pullRequest.GetUrl(),
			Repository: pullRequest.GetRepository(),
			Title:      pullRequest.GetTitle(),
			State:      enumToken(pullRequest.GetState().String()),
			Activities: activities,
		})
	}
	return out
}

func reportDispatchesJSON(dispatches []*pb.WorkOrderLineDispatch) []reportLineDispatch {
	out := make([]reportLineDispatch, 0, len(dispatches))
	for _, dispatch := range dispatches {
		steps := make([]reportStep, 0, len(dispatch.GetStepExecutions()))
		for _, step := range dispatch.GetStepExecutions() {
			item := reportStep{
				Name:              step.GetStep(),
				Index:             step.GetStepIndex(),
				State:             enumToken(step.GetState().String()),
				Result:            enumToken(step.GetResult().String()),
				Models:            step.GetModels(),
				TotalTokens:       step.GetTotalTokens(),
				CostCents:         step.GetCostCents(),
				VMDurationSeconds: step.GetDurationSeconds(),
			}
			if run := step.GetRun(); run != nil {
				item.RunID = run.GetId()
			}
			steps = append(steps, item)
		}
		item := reportLineDispatch{
			ID:        dispatch.GetId(),
			State:     enumToken(dispatch.GetState().String()),
			Result:    enumToken(dispatch.GetResult().String()),
			Model:     dispatch.GetModel(),
			CreatedAt: timestampText(dispatch.GetCreatedAt().AsTime()),
			Steps:     steps,
		}
		if line := dispatch.GetLine(); line != nil {
			item.Line = line.GetName()
		}
		if finished := dispatch.GetFinishedAt(); finished != nil {
			item.FinishedAt = timestampText(finished.AsTime())
		}
		out = append(out, item)
	}
	return out
}

func reportRunsJSON(runs []models.TaskRun) []reportRun {
	out := make([]reportRun, 0, len(runs))
	for _, run := range runs {
		item := reportRun{
			ID:                  run.ID.String(),
			AutomationID:        run.AutomationID.String(),
			AutomationName:      run.AutomationName,
			State:               run.State,
			Result:              run.Result,
			CreatedAt:           timestampText(run.CreatedAt),
			ActivityTitle:       run.ActivityTitle,
			ActivityDescription: run.ActivityDescription,
			UsageRecorded:       run.UsageRecorded,
		}
		if run.ParentRunID != nil {
			item.ParentRunID = run.ParentRunID.String()
		}
		if !run.UpdatedAt.IsZero() {
			item.UpdatedAt = timestampText(run.UpdatedAt)
		}
		if run.FinishedAt != nil {
			item.FinishedAt = timestampText(*run.FinishedAt)
		}
		if run.UsageRecorded {
			item.TotalTokens = int64Ptr(run.Usage.TotalTokens)
			item.TotalCostCents = int64Ptr(run.Usage.CostCents())
			item.VMDurationSeconds = int64Ptr(run.Usage.DurationSeconds)
		}
		out = append(out, item)
	}
	return out
}

func reportRunCountsJSON(counts []models.TaskRunCount) []reportRunCount {
	out := make([]reportRunCount, 0, len(counts))
	for _, count := range counts {
		out = append(out, reportRunCount{
			AutomationID:   count.AutomationID.String(),
			AutomationName: count.AutomationName,
			Result:         count.Result,
			Count:          count.Count,
		})
	}
	return out
}

func enumToken(value string) string {
	name := strings.ToLower(value)
	if i := strings.LastIndex(name, "_"); i >= 0 {
		name = name[i+1:]
	}
	switch name {
	case "", "unspecified", "unknown":
		return ""
	default:
		return name
	}
}

func timestampText(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func int64Ptr(value int64) *int64 {
	return &value
}
