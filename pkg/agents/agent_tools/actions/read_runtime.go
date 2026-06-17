package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	canvasactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const readRuntimeActionName = "read_runtime"

var runtimeResources = []string{
	"memory",
	"runs",
	"event_executions",
	"node_executions",
	"node_queue_items",
	"node_events",
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
	default:
		return nil, fmt.Errorf("unsupported runtime resource %q", resource)
	}
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
