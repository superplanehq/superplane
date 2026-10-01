package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
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
			Description: "Create a task in this workspace. The creator is the signed-in user. By default, the task is a draft in backlog, and refinement starts when the workspace backlog app is on. Optional arguments allow you to hand off the task to a later line column: use line to name the target line (required when a factory has multiple lines), start_step to name or index the starting column (resolved against the line's step names or numeric position), and pull_request to attach an existing pull request so automations have a target. When you provide start_step, the task is dispatched from that step and earlier steps are skipped. The result is the new task. It does not wait for an agent session.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":       map[string]any{"type": "string", "description": "Short task title."},
					"description": map[string]any{"type": "string", "description": "Task description in markdown."},
					"line": map[string]any{
						"type":        "string",
						"description": "Optional line name for handoff. When provided, the task is dispatched to this line at start_step. Required when the factory has more than one line and start_step is given.",
					},
					"start_step": map[string]any{
						"type":        "string",
						"description": "Optional starting step for handoff. Accepts a step name (e.g. 'Verify', case-insensitive) or zero-based numeric index as a string (e.g. '2'). When provided, the task is dispatched from this step and earlier steps are skipped.",
					},
					"pull_request": map[string]any{
						"type":        "object",
						"description": "Optional existing pull request to attach. Used to record the PR so automations have a target.",
						"properties": map[string]any{
							"repository": map[string]any{"type": "string", "description": "Repository owner and name (e.g., 'octocat/Hello-World')."},
							"number":     map[string]any{"type": "integer", "description": "Pull request number."},
							"url":        map[string]any{"type": "string", "description": "Pull request URL."},
							"title":      map[string]any{"type": "string", "description": "Pull request title."},
							"state":      map[string]any{"type": "string", "description": "Pull request state (e.g., 'open', 'closed', 'merged')."},
							"provider":   map[string]any{"type": "string", "description": "Provider name (defaults to 'github')."},
						},
						"required": []string{"repository", "number", "url", "title", "state"},
					},
				},
				"required": []string{"title"},
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

// pullRequestInput is the validated form of the create_task tool's optional
// pull_request argument. Resolving it does not write anything, so a bad
// pull request never leaves a task behind.
type pullRequestInput struct {
	repository string
	number     int64
	url        string
	title      string
	state      pb.FactoryPullRequest_State
	provider   pb.FactoryPullRequest_Provider
}

// dispatchTarget is the validated form of the create_task tool's optional
// line and start_step arguments. Resolving it does not write anything, so
// an unknown line or step never leaves a task behind.
type dispatchTarget struct {
	lineName  string
	stepIndex int
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

	// Validate the optional handoff arguments before creating the task. A
	// task should not be left behind in the backlog when the requested
	// handoff cannot happen.
	pullRequest, err := rt.resolvePullRequestInput(ctx, claims, args)
	if err != nil {
		return ToolResult{}, err
	}
	dispatch, err := rt.resolveDispatchTarget(ctx, claims, args)
	if err != nil {
		return ToolResult{}, err
	}

	resp, err := factories.CreateWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   claims.FactoryID.String(),
		Title:       title,
		Description: strings.TrimSpace(description),
	})
	if err != nil {
		return ToolResult{}, actionError(err)
	}
	order := resp.GetOrder()
	orderID := order.GetId()

	if pullRequest != nil {
		if err := rt.attachPullRequest(ctx, claims, orderID, pullRequest); err != nil {
			return ToolResult{}, err
		}
	}

	if dispatch != nil {
		dispatched, err := rt.dispatchWorkOrder(ctx, claims, orderID, dispatch)
		if err != nil {
			return ToolResult{}, err
		}
		// Dispatch transitions the order out of draft; use the response
		// so the tool reports the order's actual state.
		order = dispatched
	}

	return TextResult(mustJSON(map[string]any{
		"id":    order.GetId(),
		"key":   order.GetKey(),
		"title": order.GetTitle(),
		"state": protoStateName(order.GetState()),
	})), nil
}

// resolvePullRequestInput validates the optional pull_request argument. It
// returns nil when the argument is absent. It performs no writes.
func (rt *Runtime) resolvePullRequestInput(ctx context.Context, claims *AccessClaims, args map[string]any) (*pullRequestInput, error) {
	prRaw, ok := args["pull_request"]
	if !ok || prRaw == nil {
		return nil, nil
	}

	if err := rt.authorize(ctx, claims, "work_orders:update"); err != nil {
		return nil, err
	}

	prMap, ok := prRaw.(map[string]any)
	if !ok {
		return nil, ToolError("pull_request must be an object")
	}

	repository, _ := prMap["repository"].(string)
	url, _ := prMap["url"].(string)
	prTitle, _ := prMap["title"].(string)
	state, _ := prMap["state"].(string)
	provider, _ := prMap["provider"].(string)

	if strings.TrimSpace(repository) == "" {
		return nil, ToolError("pull_request.repository is required")
	}
	if strings.TrimSpace(url) == "" {
		return nil, ToolError("pull_request.url is required")
	}
	if strings.TrimSpace(prTitle) == "" {
		return nil, ToolError("pull_request.title is required")
	}
	if strings.TrimSpace(state) == "" {
		return nil, ToolError("pull_request.state is required")
	}

	number, err := parsePullRequestNumber(prMap["number"])
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(provider) == "" {
		provider = "github"
	}

	return &pullRequestInput{
		repository: strings.TrimSpace(repository),
		number:     number,
		url:        strings.TrimSpace(url),
		title:      strings.TrimSpace(prTitle),
		state:      pullRequestStateToProto(strings.ToLower(strings.TrimSpace(state))),
		provider:   pullRequestProviderToProto(strings.ToLower(strings.TrimSpace(provider))),
	}, nil
}

func parsePullRequestNumber(raw any) (int64, error) {
	number, ok := raw.(float64)
	if !ok {
		return 0, ToolError("pull_request.number is required and must be a positive integer")
	}
	if number != math.Trunc(number) {
		return 0, ToolError("pull_request.number must be a whole number")
	}
	// maxSafeInteger is the largest integer a float64 (and so JSON) can
	// represent exactly. No real pull request number comes close to it;
	// the cap only guards against precision loss.
	const maxSafeInteger = 1 << 53
	if number <= 0 || number > maxSafeInteger {
		return 0, ToolError("pull_request.number is required and must be positive")
	}
	return int64(number), nil
}

func (rt *Runtime) attachPullRequest(ctx context.Context, claims *AccessClaims, orderID string, pr *pullRequestInput) error {
	_, err := factories.CreateFactoryPullRequest(toolContext(ctx, claims), factories.IntakeDependencies{}, claims.OrgID.String(), &pb.CreateFactoryPullRequestRequest{
		FactoryId:   claims.FactoryID.String(),
		WorkOrderId: orderID,
		Provider:    pr.provider,
		Repository:  pr.repository,
		Number:      pr.number,
		Url:         pr.url,
		Title:       pr.title,
		State:       pr.state,
	})
	if err != nil {
		return actionError(err)
	}
	return nil
}

// resolveDispatchTarget validates the optional line and start_step
// arguments. It returns nil when start_step is absent. It performs no
// writes.
func (rt *Runtime) resolveDispatchTarget(ctx context.Context, claims *AccessClaims, args map[string]any) (*dispatchTarget, error) {
	startStepRaw, present := args["start_step"]
	if !present {
		return nil, nil
	}
	startStep, ok := startStepRaw.(string)
	if !ok || strings.TrimSpace(startStep) == "" {
		return nil, ToolError("start_step must be a non-empty string")
	}
	startStep = strings.TrimSpace(startStep)

	if err := rt.authorize(ctx, claims, "work_orders:update"); err != nil {
		return nil, err
	}

	db := database.DB(ctx)

	factory, err := findFactoryForDispatch(db, claims.OrgID.String(), claims.FactoryID.String())
	if err != nil {
		return nil, actionError(err)
	}

	lines, err := factory.ListLines(db)
	if err != nil {
		return nil, actionError(err)
	}

	lineName, _ := args["line"].(string)
	lineName = strings.TrimSpace(lineName)

	// If no line is specified, require exactly one line
	if lineName == "" {
		if len(lines) != 1 {
			lineNames := make([]string, 0, len(lines))
			for _, line := range lines {
				lineNames = append(lineNames, line.Name)
			}
			return nil, ToolError("multiple lines exist; specify 'line' argument with one of: " + strings.Join(lineNames, ", "))
		}
		lineName = lines[0].Name
	}

	// Find the target line
	var targetLine *models.FactoryLine
	for i := range lines {
		if lines[i].Name == lineName {
			targetLine = &lines[i]
			break
		}
	}
	if targetLine == nil {
		lineNames := make([]string, 0, len(lines))
		for _, line := range lines {
			lineNames = append(lineNames, line.Name)
		}
		return nil, ToolError("line '" + lineName + "' not found; available lines: " + strings.Join(lineNames, ", "))
	}

	startStepIndex, err := resolveStartStepIndex(db, startStep, targetLine, claims.OrgID)
	if err != nil {
		return nil, err
	}

	return &dispatchTarget{lineName: lineName, stepIndex: startStepIndex}, nil
}

func (rt *Runtime) dispatchWorkOrder(ctx context.Context, claims *AccessClaims, orderID string, target *dispatchTarget) (*pb.WorkOrder, error) {
	resp, err := factories.DispatchWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId:      claims.FactoryID.String(),
		OrderId:        orderID,
		LineName:       target.lineName,
		StartStepIndex: int32(target.stepIndex),
		ReplaceActive:  false,
	})
	if err != nil {
		return nil, actionError(err)
	}
	return resp.GetOrder(), nil
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

func findFactoryForDispatch(tx *gorm.DB, organizationID, factoryID string) (*models.Factory, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, err
	}
	factory, err := models.FindFactory(tx, orgID, parseUUID(factoryID))
	if err != nil {
		return nil, err
	}
	return factory, nil
}

func parseOrganizationID(organizationID string) (uuid.UUID, error) {
	return uuid.Parse(organizationID)
}

func parseUUID(id string) uuid.UUID {
	parsed, _ := uuid.Parse(id)
	return parsed
}

func resolveStartStepIndex(tx *gorm.DB, startStep string, line *models.FactoryLine, orgID uuid.UUID) (int, error) {
	// Try parsing as integer first
	if index, err := parseInt(startStep); err == nil && index >= 0 && index < len(line.Steps) {
		return index, nil
	}

	// Try matching against step names (canvas names)
	startStepLower := strings.ToLower(startStep)
	for i, step := range line.Steps {
		// Load canvas name for this step
		canvasName, err := getCanvasName(tx, orgID, step.AppID)
		if err == nil && strings.ToLower(canvasName) == startStepLower {
			return i, nil
		}
	}

	// Build error message with available steps
	stepNames := make([]string, 0, len(line.Steps))
	for i, step := range line.Steps {
		canvasName, err := getCanvasName(tx, orgID, step.AppID)
		if err == nil && canvasName != "" {
			stepNames = append(stepNames, canvasName)
		} else {
			stepNames = append(stepNames, fmt.Sprintf("step_%d", i))
		}
	}

	return 0, ToolError("start_step '" + startStep + "' not found; available steps: " + strings.Join(stepNames, ", "))
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(s))
}

func getCanvasName(tx *gorm.DB, orgID, appID uuid.UUID) (string, error) {
	// Load the canvas from the database
	canvas, err := models.FindCanvasInTransaction(tx, orgID, appID)
	if err != nil {
		return "", err
	}
	return canvas.Name, nil
}

func pullRequestProviderToProto(provider string) pb.FactoryPullRequest_Provider {
	switch strings.ToLower(provider) {
	case "github":
		return pb.FactoryPullRequest_PROVIDER_GITHUB
	case "bitbucket":
		return pb.FactoryPullRequest_PROVIDER_BITBUCKET
	default:
		return pb.FactoryPullRequest_PROVIDER_GITHUB
	}
}

func pullRequestStateToProto(state string) pb.FactoryPullRequest_State {
	switch strings.ToLower(state) {
	case "open":
		return pb.FactoryPullRequest_STATE_OPEN
	case "draft":
		return pb.FactoryPullRequest_STATE_DRAFT
	case "closed":
		return pb.FactoryPullRequest_STATE_CLOSED
	case "merged":
		return pb.FactoryPullRequest_STATE_MERGED
	default:
		return pb.FactoryPullRequest_STATE_UNSPECIFIED
	}
}
