package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	canvasactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const readRuntimeActionName = "read_runtime"

var runtimeResources = []string{
	"memory",
	"runs",
	"event_executions",
	"node_executions",
	"node_queue_items",
	"node_events",
	"runner_logs",
}

type runnerLogTarget struct {
	ExecutionID uuid.UUID
	NodeID      string
	RunID       uuid.UUID
}

type runnerLogsPayload struct {
	Count int                `json:"count"`
	Logs  []runnerLogsResult `json:"logs"`
}

type runnerLogsResult struct {
	ExecutionID  string                       `json:"execution_id"`
	NodeID       string                       `json:"node_id,omitempty"`
	RunID        string                       `json:"run_id,omitempty"`
	BrokerTaskID string                       `json:"broker_task_id,omitempty"`
	Count        int                          `json:"count"`
	Truncated    bool                         `json:"truncated,omitempty"`
	Records      []runneraction.LiveLogRecord `json:"records,omitempty"`
	Error        string                       `json:"error,omitempty"`
}

type readRuntimeAction struct {
	registry *registry.Registry
	auth     organizationPermissionChecker
}

func newReadRuntimeAction(deps Dependencies) readRuntimeAction {
	return readRuntimeAction{
		registry: deps.Registry,
		auth:     deps.AuthService,
	}
}

func (a readRuntimeAction) Name() string {
	return readRuntimeActionName
}

func (a readRuntimeAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	if a.registry == nil {
		return runtimeReadResult{}, fmt.Errorf("component registry is not configured")
	}
	if err := a.checkReadPermission(ctx, session); err != nil {
		return runtimeReadResult{}, err
	}

	resource := strings.TrimSpace(input.Resource)
	if resource == "" {
		resource = "memory"
	}
	if !slices.Contains(runtimeResources, resource) {
		return runtimeReadResult{}, fmt.Errorf("unsupported runtime resource %q", input.Resource)
	}

	payload, err := a.read(ctx, session, input, resource)
	if err != nil {
		return runtimeReadResult{}, err
	}

	return runtimeReadResult{
		Action:   readRuntimeActionName,
		CanvasID: session.CanvasID,
		Resource: resource,
		Payload:  payload,
	}, nil
}

func (a readRuntimeAction) checkReadPermission(ctx context.Context, session agents.AgentSessionContext) error {
	if a.auth == nil {
		return fmt.Errorf("authorization service is unavailable")
	}

	allowed, err := a.auth.CheckOrganizationPermission(ctx, session.UserID, session.OrganizationID, "canvases", "read")
	if err != nil {
		return fmt.Errorf("check canvases:read permission: %w", err)
	}
	if !allowed {
		return fmt.Errorf("user RBAC does not grant canvases:read")
	}
	return nil
}

func (a readRuntimeAction) read(ctx context.Context, session agents.AgentSessionContext, input Input, resource string) (any, error) {
	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid session canvas id: %w", err)
	}

	before, err := parseRuntimeBefore(input.Before)
	if err != nil {
		return nil, err
	}

	switch resource {
	case "memory":
		response, err := canvasactions.ListCanvasMemories(ctx, a.registry, session.OrganizationID, session.CanvasID)
		if err != nil {
			return nil, err
		}
		return filterMemoryResponse(response, input.Namespace)
	case "runs":
		states, err := parseRunStates(input.States)
		if err != nil {
			return nil, err
		}
		results, err := parseRunResults(input.Results)
		if err != nil {
			return nil, err
		}
		return protoPayload(canvasactions.ListRuns(ctx, a.registry, canvasID, input.Limit, before, states, results))
	case "event_executions":
		if strings.TrimSpace(input.EventID) == "" {
			return nil, fmt.Errorf("event_id is required for event_executions")
		}
		return protoPayload(canvasactions.ListEventExecutions(ctx, a.registry, session.CanvasID, input.EventID))
	case "node_executions":
		if strings.TrimSpace(input.NodeID) == "" {
			return nil, fmt.Errorf("node_id is required for node_executions")
		}
		states, err := parseExecutionStates(input.States)
		if err != nil {
			return nil, err
		}
		results, err := parseExecutionResults(input.Results)
		if err != nil {
			return nil, err
		}
		return protoPayload(canvasactions.ListNodeExecutions(ctx, a.registry, session.CanvasID, input.NodeID, states, results, input.Limit, before))
	case "node_queue_items":
		if strings.TrimSpace(input.NodeID) == "" {
			return nil, fmt.Errorf("node_id is required for node_queue_items")
		}
		return protoPayload(canvasactions.ListNodeQueueItems(ctx, a.registry, session.CanvasID, input.NodeID, input.Limit, before))
	case "node_events":
		if strings.TrimSpace(input.NodeID) == "" {
			return nil, fmt.Errorf("node_id is required for node_events")
		}
		return protoPayload(canvasactions.ListNodeEvents(ctx, a.registry, canvasID, input.NodeID, input.Limit, before))
	case "runner_logs":
		return a.readRunnerLogs(ctx, session, canvasID, input)
	default:
		return nil, fmt.Errorf("unsupported runtime resource %q", resource)
	}
}

func (a readRuntimeAction) readRunnerLogs(ctx context.Context, session agents.AgentSessionContext, canvasID uuid.UUID, input Input) (any, error) {
	organizationID, err := uuid.Parse(session.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid session organization id: %w", err)
	}

	targets, err := resolveRunnerLogTargets(database.DB(ctx), canvasID, input)
	if err != nil {
		return nil, err
	}

	logs := make([]runnerLogsResult, 0, len(targets))
	for _, target := range targets {
		log, err := fetchRunnerLogsForTarget(ctx, organizationID, canvasID, target, int(input.Limit))
		if err != nil && len(targets) == 1 {
			return nil, err
		}
		if err != nil {
			log.Error = err.Error()
		}
		logs = append(logs, log)
	}

	return runnerLogsPayload{
		Count: len(logs),
		Logs:  logs,
	}, nil
}

func resolveRunnerLogTargets(tx *gorm.DB, canvasID uuid.UUID, input Input) ([]runnerLogTarget, error) {
	if strings.TrimSpace(input.ExecutionID) != "" {
		return resolveRunnerLogExecutionTarget(tx, canvasID, input.ExecutionID)
	}

	if strings.TrimSpace(input.RunID) != "" {
		return resolveRunnerLogRunTargets(tx, canvasID, input.RunID, input.NodeID)
	}

	if strings.TrimSpace(input.NodeID) != "" {
		return resolveLatestRunnerLogNodeTarget(tx, canvasID, input.NodeID)
	}

	return nil, fmt.Errorf("execution_id, run_id, or node_id is required for runner_logs")
}

func resolveRunnerLogExecutionTarget(tx *gorm.DB, canvasID uuid.UUID, rawExecutionID string) ([]runnerLogTarget, error) {
	executionID, err := uuid.Parse(strings.TrimSpace(rawExecutionID))
	if err != nil {
		return nil, fmt.Errorf("invalid execution_id: %w", err)
	}

	execution, err := findRunnerLogExecutionTarget(tx, canvasID, executionID)
	if err != nil {
		return nil, fmt.Errorf("load execution: %w", err)
	}

	return []runnerLogTarget{{
		ExecutionID: execution.ID,
		NodeID:      execution.NodeID,
		RunID:       execution.RunID,
	}}, nil
}

func resolveRunnerLogRunTargets(tx *gorm.DB, canvasID uuid.UUID, rawRunID, nodeID string) ([]runnerLogTarget, error) {
	runID, err := uuid.Parse(strings.TrimSpace(rawRunID))
	if err != nil {
		return nil, fmt.Errorf("invalid run_id: %w", err)
	}

	if _, err := models.FindCanvasRunInTransaction(tx, canvasID, runID); err != nil {
		return nil, fmt.Errorf("load run: %w", err)
	}

	executions, err := models.ListExecutionsForRunsInTransaction(tx, canvasID, []uuid.UUID{runID})
	if err != nil {
		return nil, fmt.Errorf("list run executions: %w", err)
	}

	targets := make([]runnerLogTarget, 0, len(executions))
	for _, execution := range executions {
		if strings.TrimSpace(nodeID) != "" && execution.NodeID != nodeID {
			continue
		}
		isRunner, err := isRunnerLogExecutionTarget(tx, canvasID, execution.NodeID)
		if err != nil {
			return nil, err
		}
		if !isRunner {
			continue
		}
		targets = append(targets, runnerLogTarget{
			ExecutionID: execution.ID,
			NodeID:      execution.NodeID,
			RunID:       execution.RunID,
		})
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no runner executions found for runner_logs target")
	}
	return targets, nil
}

func resolveLatestRunnerLogNodeTarget(tx *gorm.DB, canvasID uuid.UUID, nodeID string) ([]runnerLogTarget, error) {
	executions, err := listLatestRunnerLogNodeExecutions(tx, canvasID, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list node executions: %w", err)
	}
	if len(executions) == 0 {
		return nil, fmt.Errorf("no executions found for node_id %q", nodeID)
	}

	execution := executions[0]
	return []runnerLogTarget{{
		ExecutionID: execution.ID,
		NodeID:      execution.NodeID,
		RunID:       execution.RunID,
	}}, nil
}

func findRunnerLogExecutionTarget(tx *gorm.DB, canvasID, executionID uuid.UUID) (*models.CanvasNodeExecution, error) {
	var execution models.CanvasNodeExecution
	err := tx.
		Where("workflow_id = ?", canvasID).
		Where("id = ?", executionID).
		First(&execution).
		Error
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

func isRunnerLogExecutionTarget(tx *gorm.DB, canvasID uuid.UUID, nodeID string) (bool, error) {
	node, err := models.FindCanvasNode(tx, canvasID, nodeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("load node %q: %w", nodeID, err)
	}
	ref := node.Ref.Data()
	if ref.Component == nil {
		return false, nil
	}
	return runneraction.IsRunnerComponent(ref.Component.Name), nil
}

func listLatestRunnerLogNodeExecutions(tx *gorm.DB, canvasID uuid.UUID, nodeID string) ([]models.CanvasNodeExecution, error) {
	var executions []models.CanvasNodeExecution
	err := tx.
		Where("workflow_id = ?", canvasID).
		Where("node_id = ?", nodeID).
		Order("created_at DESC").
		Limit(1).
		Find(&executions).
		Error
	if err != nil {
		return nil, err
	}
	return executions, nil
}

func fetchRunnerLogsForTarget(ctx context.Context, organizationID, canvasID uuid.UUID, target runnerLogTarget, limit int) (runnerLogsResult, error) {
	result := runnerLogsResult{
		ExecutionID: target.ExecutionID.String(),
		NodeID:      target.NodeID,
	}
	if target.RunID != uuid.Nil {
		result.RunID = target.RunID.String()
	}

	access, err := runneraction.ResolveLiveLogAccess(organizationID, canvasID, target.ExecutionID)
	if err != nil {
		return result, err
	}

	fetch, err := runneraction.FetchLiveLogRecords(ctx, access.BrokerTaskID, runneraction.LiveLogFetchOptions{Limit: limit})
	if err != nil {
		return result, err
	}

	result.BrokerTaskID = access.BrokerTaskID
	result.Records = fetch.Records
	result.Count = len(fetch.Records)
	result.Truncated = fetch.Truncated
	return result, nil
}

func filterMemoryResponse(response *pb.ListCanvasMemoriesResponse, namespace string) (any, error) {
	if strings.TrimSpace(namespace) == "" {
		return protoPayload(response, nil)
	}

	filtered := &pb.ListCanvasMemoriesResponse{}
	for _, item := range response.GetItems() {
		if item.GetNamespace() == namespace {
			filtered.Items = append(filtered.Items, item)
		}
	}
	return protoPayload(filtered, nil)
}

func protoPayload(response proto.Message, err error) (any, error) {
	if err != nil {
		return nil, err
	}

	bytes, err := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime response: %w", err)
	}

	var payload any
	if err := json.Unmarshal(bytes, &payload); err != nil {
		return nil, fmt.Errorf("decode runtime response: %w", err)
	}
	return payload, nil
}

func parseRuntimeBefore(value string) (*timestamppb.Timestamp, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, fmt.Errorf("before must be an RFC3339 timestamp: %w", err)
	}
	return timestamppb.New(parsed), nil
}

func parseRunStates(values []string) ([]pb.CanvasRun_State, error) {
	states := make([]pb.CanvasRun_State, 0, len(values))
	for _, value := range values {
		switch normalizeRuntimeFilter(value) {
		case "started":
			states = append(states, pb.CanvasRun_STATE_STARTED)
		case "finished":
			states = append(states, pb.CanvasRun_STATE_FINISHED)
		case "":
		default:
			return nil, fmt.Errorf("unsupported run state %q", value)
		}
	}
	return states, nil
}

func parseRunResults(values []string) ([]pb.CanvasRun_Result, error) {
	results := make([]pb.CanvasRun_Result, 0, len(values))
	for _, value := range values {
		switch normalizeRuntimeFilter(value) {
		case "passed":
			results = append(results, pb.CanvasRun_RESULT_PASSED)
		case "failed":
			results = append(results, pb.CanvasRun_RESULT_FAILED)
		case "cancelled", "canceled":
			results = append(results, pb.CanvasRun_RESULT_CANCELLED)
		case "":
		default:
			return nil, fmt.Errorf("unsupported run result %q", value)
		}
	}
	return results, nil
}

func parseExecutionStates(values []string) ([]pb.CanvasNodeExecution_State, error) {
	states := make([]pb.CanvasNodeExecution_State, 0, len(values))
	for _, value := range values {
		switch normalizeRuntimeFilter(value) {
		case "pending":
			states = append(states, pb.CanvasNodeExecution_STATE_PENDING)
		case "started":
			states = append(states, pb.CanvasNodeExecution_STATE_STARTED)
		case "finished":
			states = append(states, pb.CanvasNodeExecution_STATE_FINISHED)
		case "":
		default:
			return nil, fmt.Errorf("unsupported execution state %q", value)
		}
	}
	return states, nil
}

func parseExecutionResults(values []string) ([]pb.CanvasNodeExecution_Result, error) {
	results := make([]pb.CanvasNodeExecution_Result, 0, len(values))
	for _, value := range values {
		switch normalizeRuntimeFilter(value) {
		case "passed":
			results = append(results, pb.CanvasNodeExecution_RESULT_PASSED)
		case "failed":
			results = append(results, pb.CanvasNodeExecution_RESULT_FAILED)
		case "":
		default:
			return nil, fmt.Errorf("unsupported execution result %q", value)
		}
	}
	return results, nil
}

func normalizeRuntimeFilter(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = strings.TrimPrefix(normalized, "state_")
	normalized = strings.TrimPrefix(normalized, "result_")
	return normalized
}
