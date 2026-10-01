package mcpserver

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/grpc/codes"
)

type Runtime struct {
	Auth authorization.PermissionChecker
}

func Tools() []Tool {
	return []Tool{
		{
			Name:        "list_tasks",
			Description: "List tasks in this workspace. Draft is the backlog. Open is in progress. Closed is finished. Set mine to true to return tasks assigned to you or created by you. Poll after a write to see new state.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"states": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string", "enum": []string{"draft", "open", "closed"}},
						"description": "Task states to include. Omit to list every state.",
					},
					"mine": map[string]any{
						"type":        "boolean",
						"description": "When true, return tasks assigned to you or created by you.",
					},
					"before_id": map[string]any{
						"type":        "string",
						"description": "Return the next page older than this task id.",
					},
				},
			},
		},
		{
			Name:        "get_task",
			Description: "Return one task by id, number, or key.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"task": map[string]any{"type": "string", "description": "Task id, number, or key."}},
				"required":   []string{"task"},
			},
		},
		{
			Name:        "list_task_artifacts",
			Description: "List artifacts on a task. File artifacts include a public download URL. The response does not include file bytes.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"task": map[string]any{"type": "string", "description": "Task id, number, or key."}},
				"required":   []string{"task"},
			},
		},
		{
			Name:        "get_task_agent",
			Description: "Return the refinement agent for a task: session state, recent messages, and an open survey when one is waiting. If the task has no agent session, the tool reports that.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"task": map[string]any{"type": "string", "description": "Task id, number, or key."}},
				"required":   []string{"task"},
			},
		},
		{
			Name:        "send_task_message",
			Description: "Send a message to the refinement agent on a draft task. This stores the message and may restart refinement. It does not wait for the agent. Call get_task_agent to read the reply. Use the same tool to answer a survey.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task":    map[string]any{"type": "string", "description": "Task id, number, or key."},
					"message": map[string]any{"type": "string", "description": "Message text for the agent."},
				},
				"required": []string{"task", "message"},
			},
		},
		{
			Name:        "create_task",
			Description: "Create a draft task in this workspace. The creator is the signed-in user. Refinement starts when the workspace backlog app is on. The result is the new task. It does not wait for an agent session.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":       map[string]any{"type": "string", "description": "Short task title."},
					"description": map[string]any{"type": "string", "description": "Task description in markdown."},
				},
				"required": []string{"title"},
			},
		},
		{
			Name:        "close_task",
			Description: "Close a task after the user asks you to close it. Call this when the user chooses Close the task. Omit result to close a draft as rejected, or an open task as completed.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task": map[string]any{"type": "string", "description": "Task id, number, or key."},
					"result": map[string]any{
						"type":        "string",
						"enum":        []string{"completed", "rejected", "failed"},
						"description": "Close result. Omit to use the default for the task state.",
					},
				},
				"required": []string{"task"},
			},
		},
	}
}

func (rt *Runtime) CallTool(ctx context.Context, claims *AccessClaims, name string, args map[string]any) (ToolResult, error) {
	if rt == nil || rt.Auth == nil || claims == nil {
		return ToolResult{}, ToolError("Not found")
	}
	switch name {
	case "list_tasks":
		return rt.listTasks(ctx, claims, args)
	case "get_task":
		return rt.getTask(ctx, claims, args)
	case "list_task_artifacts":
		return rt.listTaskArtifacts(ctx, claims, args)
	case "get_task_agent":
		return rt.getTaskAgent(ctx, claims, args)
	case "send_task_message":
		return rt.sendTaskMessage(ctx, claims, args)
	case "create_task":
		return rt.createTask(ctx, claims, args)
	case "close_task":
		return rt.closeTask(ctx, claims, args)
	default:
		return ToolResult{}, ToolError("Unknown tool: " + name)
	}
}

func (rt *Runtime) listTasks(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	if err := rt.authorize(ctx, claims, "work_orders:read"); err != nil {
		return ToolResult{}, err
	}
	req := &pb.ListWorkOrdersRequest{FactoryId: claims.FactoryID.String()}
	if mine, _ := args["mine"].(bool); mine {
		userID := claims.UserID.String()
		req.UserId = &userID
	}
	if beforeID, _ := args["before_id"].(string); strings.TrimSpace(beforeID) != "" {
		req.BeforeId = strings.TrimSpace(beforeID)
	}
	states, err := parseTaskStates(args["states"])
	if err != nil {
		return ToolResult{}, err
	}
	req.States = states

	resp, err := factories.ListWorkOrders(toolContext(ctx, claims), claims.OrgID.String(), req)
	if err != nil {
		return ToolResult{}, actionError(err)
	}
	type taskRow struct {
		ID    string `json:"id"`
		Key   string `json:"key"`
		Title string `json:"title"`
		State string `json:"state"`
	}
	rows := make([]taskRow, 0, len(resp.GetOrders()))
	for _, order := range resp.GetOrders() {
		rows = append(rows, taskRow{
			ID:    order.GetId(),
			Key:   order.GetKey(),
			Title: order.GetTitle(),
			State: protoStateName(order.GetState()),
		})
	}
	payload := map[string]any{"tasks": rows, "has_next_page": resp.GetHasNextPage()}
	if resp.GetHasNextPage() && len(rows) > 0 {
		payload["next_before_id"] = rows[len(rows)-1].ID
	}
	return TextResult(mustJSON(payload)), nil
}

func (rt *Runtime) getTask(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	if err := rt.authorize(ctx, claims, "work_orders:read"); err != nil {
		return ToolResult{}, err
	}
	task, err := stringArg(args, "task")
	if err != nil {
		return ToolResult{}, err
	}
	resp, err := factories.DescribeWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.DescribeWorkOrderRequest{
		FactoryId: claims.FactoryID.String(),
		OrderId:   task,
	})
	if err != nil {
		return ToolResult{}, actionError(err)
	}
	order := resp.GetOrder()
	return TextResult(mustJSON(map[string]any{
		"id":          order.GetId(),
		"key":         order.GetKey(),
		"title":       order.GetTitle(),
		"description": order.GetDescription(),
		"state":       protoStateName(order.GetState()),
	})), nil
}

func (rt *Runtime) listTaskArtifacts(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	if err := rt.authorize(ctx, claims, "work_orders:read"); err != nil {
		return ToolResult{}, err
	}
	task, err := stringArg(args, "task")
	if err != nil {
		return ToolResult{}, err
	}
	resp, err := factories.ListWorkOrderArtifacts(toolContext(ctx, claims), claims.OrgID.String(), &pb.ListWorkOrderArtifactsRequest{
		FactoryId: claims.FactoryID.String(),
		OrderId:   task,
	})
	if err != nil {
		return ToolResult{}, actionError(err)
	}
	artifacts := make([]map[string]any, 0, len(resp.GetArtifacts()))
	for _, artifact := range resp.GetArtifacts() {
		row := map[string]any{
			"id":   artifact.GetId(),
			"type": strings.ToLower(strings.TrimPrefix(artifact.GetType().String(), "TYPE_")),
		}
		if data := artifact.GetData(); data != nil {
			fields := data.AsMap()
			if url, _ := fields["url"].(string); url != "" {
				row["url"] = url
			}
			if title, _ := fields["title"].(string); title != "" {
				row["title"] = title
			}
			if filename, _ := fields["filename"].(string); filename != "" {
				row["filename"] = filename
			}
		}
		artifacts = append(artifacts, row)
	}
	return TextResult(mustJSON(map[string]any{"artifacts": artifacts})), nil
}

func (rt *Runtime) getTaskAgent(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	if err := rt.authorize(ctx, claims, "work_orders:read"); err != nil {
		return ToolResult{}, err
	}
	task, err := stringArg(args, "task")
	if err != nil {
		return ToolResult{}, err
	}
	orderID, err := resolveTaskID(ctx, claims, task)
	if err != nil {
		return ToolResult{}, err
	}
	resp, err := factories.FindPlanningSessionByWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
		FactoryId:   claims.FactoryID.String(),
		WorkOrderId: orderID,
	})
	if err != nil {
		if isNotFound(err) {
			return ToolResult{}, ToolError("This task has no agent session.")
		}
		return ToolResult{}, actionError(err)
	}
	return TextResult(mustJSON(sessionSummary(resp.GetSession()))), nil
}

func (rt *Runtime) sendTaskMessage(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	if err := rt.authorize(ctx, claims, "work_orders:update"); err != nil {
		return ToolResult{}, err
	}
	task, err := stringArg(args, "task")
	if err != nil {
		return ToolResult{}, err
	}
	message, err := stringArg(args, "message")
	if err != nil {
		return ToolResult{}, err
	}
	orderID, err := resolveTaskID(ctx, claims, task)
	if err != nil {
		return ToolResult{}, err
	}
	found, err := factories.FindPlanningSessionByWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
		FactoryId:   claims.FactoryID.String(),
		WorkOrderId: orderID,
	})
	if err != nil {
		if isNotFound(err) {
			return ToolResult{}, ToolError("This task has no agent session.")
		}
		return ToolResult{}, actionError(err)
	}
	resp, err := factories.SendPlanningSessionMessage(toolContext(ctx, claims), claims.OrgID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: claims.FactoryID.String(),
		SessionId: found.GetSession().GetId(),
		Text:      message,
	})
	if err != nil {
		return ToolResult{}, actionError(err)
	}
	return TextResult(mustJSON(map[string]any{
		"status":     "stored",
		"session_id": resp.GetSession().GetId(),
		"state":      resp.GetSession().GetState(),
		"note":       "The message is stored. Call get_task_agent to read the agent reply.",
	})), nil
}

func (rt *Runtime) createTask(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	if err := rt.authorize(ctx, claims, "work_orders:create"); err != nil {
		return ToolResult{}, err
	}
	title, err := stringArg(args, "title")
	if err != nil {
		return ToolResult{}, err
	}
	description, _ := args["description"].(string)
	resp, err := factories.CreateWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   claims.FactoryID.String(),
		Title:       title,
		Description: strings.TrimSpace(description),
	})
	if err != nil {
		return ToolResult{}, actionError(err)
	}
	order := resp.GetOrder()
	return TextResult(mustJSON(map[string]any{
		"id":    order.GetId(),
		"key":   order.GetKey(),
		"title": order.GetTitle(),
		"state": protoStateName(order.GetState()),
	})), nil
}

func (rt *Runtime) closeTask(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	if err := rt.authorize(ctx, claims, "work_orders:update"); err != nil {
		return ToolResult{}, err
	}
	task, err := stringArg(args, "task")
	if err != nil {
		return ToolResult{}, err
	}
	orderID, err := resolveTaskID(ctx, claims, task)
	if err != nil {
		return ToolResult{}, err
	}
	result, err := closeTaskResult(ctx, claims, orderID, args["result"])
	if err != nil {
		return ToolResult{}, err
	}
	resp, err := factories.CloseWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.CloseWorkOrderRequest{
		FactoryId: claims.FactoryID.String(),
		OrderId:   orderID,
		Result:    result,
	})
	if err != nil {
		return ToolResult{}, actionError(err)
	}
	order := resp.GetOrder()
	return TextResult(mustJSON(map[string]any{
		"id":     order.GetId(),
		"key":    order.GetKey(),
		"title":  order.GetTitle(),
		"state":  protoStateName(order.GetState()),
		"result": protoResultName(order.GetResult()),
	})), nil
}

func closeTaskResult(ctx context.Context, claims *AccessClaims, orderID string, raw any) (pb.WorkOrder_Result, error) {
	if name, _ := raw.(string); strings.TrimSpace(name) != "" {
		result, ok := closeResultByName(name)
		if !ok {
			return pb.WorkOrder_RESULT_UNSPECIFIED, ToolError("result must be completed, rejected, or failed")
		}
		return result, nil
	}
	described, err := factories.DescribeWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.DescribeWorkOrderRequest{
		FactoryId: claims.FactoryID.String(),
		OrderId:   orderID,
	})
	if err != nil {
		return pb.WorkOrder_RESULT_UNSPECIFIED, actionError(err)
	}
	if described.GetOrder().GetState() == pb.WorkOrder_STATE_DRAFT {
		return pb.WorkOrder_RESULT_REJECTED, nil
	}
	return pb.WorkOrder_RESULT_COMPLETED, nil
}

func closeResultByName(name string) (pb.WorkOrder_Result, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "completed":
		return pb.WorkOrder_RESULT_COMPLETED, true
	case "rejected":
		return pb.WorkOrder_RESULT_REJECTED, true
	case "failed":
		return pb.WorkOrder_RESULT_FAILED, true
	default:
		return pb.WorkOrder_RESULT_UNSPECIFIED, false
	}
}

func protoResultName(result pb.WorkOrder_Result) string {
	switch result {
	case pb.WorkOrder_RESULT_COMPLETED:
		return "completed"
	case pb.WorkOrder_RESULT_REJECTED:
		return "rejected"
	case pb.WorkOrder_RESULT_FAILED:
		return "failed"
	default:
		return ""
	}
}

func (rt *Runtime) authorize(ctx context.Context, claims *AccessClaims, scope string) error {
	if !claims.HasScope(scope) {
		return ToolError("Not found")
	}
	if !OrganizationAllowsPublicMCP(claims.OrgID) || !organizationAllowsFactories(claims.OrgID) {
		return ToolError("Not found")
	}
	resource, action, ok := strings.Cut(scope, ":")
	if !ok {
		return ToolError("Not found")
	}
	allowed, err := rt.Auth.CheckOrganizationPermission(ctx, claims.UserID.String(), claims.OrgID.String(), resource, action)
	if err != nil || !allowed {
		return ToolError("Not found")
	}
	return nil
}

func toolContext(ctx context.Context, claims *AccessClaims) context.Context {
	ctx = authentication.SetUserIdInMetadata(ctx, claims.UserID.String())
	return context.WithValue(ctx, authorization.OrganizationContextKey, claims.OrgID.String())
}

func resolveTaskID(ctx context.Context, claims *AccessClaims, task string) (string, error) {
	resp, err := factories.DescribeWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.DescribeWorkOrderRequest{
		FactoryId: claims.FactoryID.String(),
		OrderId:   task,
	})
	if err != nil {
		return "", actionError(err)
	}
	return resp.GetOrder().GetId(), nil
}

func parseTaskStates(raw any) ([]pb.WorkOrder_State, error) {
	if raw == nil {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, ToolError("states must be an array of draft, open, or closed")
	}
	out := make([]pb.WorkOrder_State, 0, len(items))
	for _, item := range items {
		name, _ := item.(string)
		state := factories.WorkOrderStateToProto(strings.ToLower(strings.TrimSpace(name)))
		if state == pb.WorkOrder_STATE_UNSPECIFIED {
			return nil, ToolError("states must be draft, open, or closed")
		}
		out = append(out, state)
	}
	return out, nil
}

func protoStateName(state pb.WorkOrder_State) string {
	name, ok := factories.WorkOrderStateFromProto(state)
	if !ok {
		return "unspecified"
	}
	return name
}

func sessionSummary(session *pb.PlanningSession) map[string]any {
	if session == nil {
		return map[string]any{}
	}
	messages := make([]map[string]any, 0, len(session.GetMessages()))
	for _, message := range session.GetMessages() {
		messages = append(messages, map[string]any{
			"role": message.GetRole(),
			"text": message.GetText(),
		})
	}
	summary := map[string]any{
		"session_id": session.GetId(),
		"state":      session.GetState(),
		"messages":   messages,
	}
	if survey := session.GetSurvey(); survey != nil && survey.GetId() != "" {
		questions := make([]map[string]any, 0, len(survey.GetQuestions()))
		for _, question := range survey.GetQuestions() {
			questions = append(questions, map[string]any{
				"prompt":  question.GetPrompt(),
				"options": question.GetOptions(),
			})
		}
		summary["survey"] = map[string]any{"id": survey.GetId(), "questions": questions}
	}
	return summary
}

func stringArg(args map[string]any, name string) (string, error) {
	value, _ := args[name].(string)
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ToolError(name + " is required")
	}
	return value, nil
}

func actionError(err error) error {
	if err == nil {
		return nil
	}
	if grpcerrors.Code(err) == codes.NotFound {
		return ToolError("Not found")
	}
	if message := strings.TrimSpace(grpcerrors.StatusMessage(err)); message != "" {
		return ToolError(message)
	}
	return ToolError("The request failed.")
}

func isNotFound(err error) bool {
	return grpcerrors.Code(err) == codes.NotFound
}

func mustJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
