package mcpserver

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/grpc/codes"
)

type Runtime struct {
	Auth   authorization.PermissionChecker
	Intake factories.IntakeDependencies
	// BaseURL is the installation's web app base URL. Task and workspace URLs are built from it.
	BaseURL string
}

func Tools() []Tool {
	tools := []Tool{
		{
			Name:        "list_workspaces",
			Description: "List workspaces this connection can use. Each workspace has an id and a name.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "list_tasks",
			Description: "List tasks in this workspace. Draft is the backlog. Open is in progress. Closed is finished. Set mine to true to return tasks assigned to you or created by you. Set source_url to the full issue URL to return tasks created from that issue. Archived tasks stay hidden. Poll after a write to see new state.",
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
					"source_url": map[string]any{
						"type":        "string",
						"description": "Full issue URL. GitHub matches the repository and issue number. Issue 12 does not match issue 123. Archived tasks stay hidden.",
					},
				},
			},
		},
		{
			Name:        "get_task_report",
			Description: "Return recorded usage, pull requests, line steps, and automation runs for one task. The task is an id, a number, or a key. Archived tasks are included. usage.recorded is false when the task has no usage rows. Do not treat missing totals as zero. vm_duration_seconds is runner time, not elapsed time. Task totals are the usage authority. run_counts cover every associated run. The runs list is one page.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task": map[string]any{
						"type":        "string",
						"description": "Task id, number, or key.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Run page size. The default is 50. The maximum is 200.",
					},
					"before": map[string]any{
						"type":        "string",
						"description": "Run cursor from next_run_before. Omit it for the first page.",
					},
				},
				"required": []string{"task"},
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
			Name:        "hand_off_task",
			Description: "Create an open task from this MCP client and move it to Implement or Verify. Store plan as the spec the implement agent reads. Implement starts the implementation stage. Verify tracks an open pull request and does not start an agent. It does not wait for an agent.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":            map[string]any{"type": "string", "description": "Short task title."},
					"description":      map[string]any{"type": "string", "description": "Task description in markdown."},
					"plan":             map[string]any{"type": "string", "description": "Implementation plan stored as the task spec."},
					"column":           map[string]any{"type": "string", "enum": []string{"implement", "verify"}, "description": "Board column to hand the task to."},
					"pull_request_url": map[string]any{"type": "string", "description": "Open pull request URL (GitHub or Bitbucket). Required when column is verify."},
					"line":             map[string]any{"type": "string", "description": "Line name. Required when the workspace has more than one line."},
				},
				"required": []string{"title", "plan", "column"},
			},
		},
		{
			Name:        "assign_task",
			Description: "Set or clear the owner of an existing task. assignee is an email, a GitHub login, or a user id. Set unassign to true to clear every owner. Set one of those values, not both.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task":     map[string]any{"type": "string", "description": "Task id, number, or key."},
					"assignee": map[string]any{"type": "string", "description": "Workspace member email, GitHub login, or user id."},
					"unassign": map[string]any{"type": "boolean", "description": "When true, clear every owner."},
				},
				"required": []string{"task"},
			},
		},
		{
			Name:        "list_members",
			Description: "List human members of this organization. Each member has an id, a name, an email, and a GitHub login. Use this list to choose an assignee.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
	for i := range tools {
		if tools[i].Name == "list_members" || tools[i].Name == "list_workspaces" {
			continue
		}
		tools[i] = withWorkspaceArgument(tools[i])
	}
	return tools
}

func withWorkspaceArgument(tool Tool) Tool {
	if tool.InputSchema == nil {
		tool.InputSchema = map[string]any{"type": "object"}
	}
	properties, _ := tool.InputSchema["properties"].(map[string]any)
	if properties == nil {
		properties = map[string]any{}
		tool.InputSchema["properties"] = properties
	}
	properties["workspace"] = map[string]any{
		"type":        "string",
		"description": "Workspace id from list_workspaces. Required when this connection can use more than one workspace.",
	}
	return tool
}

func (rt *Runtime) CallTool(ctx context.Context, claims *AccessClaims, name string, args map[string]any) (ToolResult, error) {
	if rt == nil || rt.Auth == nil || claims == nil {
		return ToolResult{}, ToolError("Not found")
	}
	switch name {
	case "list_workspaces":
		return rt.listWorkspaces(ctx, claims, args)
	case "list_tasks":
		return rt.listTasks(ctx, claims, args)
	case "get_task":
		return rt.getTask(ctx, claims, args)
	case "get_task_report":
		return rt.getTaskReport(ctx, claims, args)
	case "list_task_artifacts":
		return rt.listTaskArtifacts(ctx, claims, args)
	case "get_task_agent":
		return rt.getTaskAgent(ctx, claims, args)
	case "send_task_message":
		return rt.sendTaskMessage(ctx, claims, args)
	case "create_task":
		return rt.createTask(ctx, claims, args)
	case "hand_off_task":
		return rt.handOffTask(ctx, claims, args)
	case "assign_task":
		return rt.assignTask(ctx, claims, args)
	case "list_members":
		return rt.listMembers(ctx, claims, args)
	default:
		return ToolResult{}, ToolError("Unknown tool: " + name)
	}
}

func (rt *Runtime) listWorkspaces(ctx context.Context, claims *AccessClaims, _ map[string]any) (ToolResult, error) {
	if err := rt.authorize(ctx, claims, "work_orders:read"); err != nil {
		return ToolResult{}, err
	}
	if !OrganizationAllowsPublicMCP(claims.OrgID) || !organizationAllowsFactories(claims.OrgID) {
		return ToolResult{}, ToolError("Not found")
	}
	rows := make([]workspaceRow, 0, len(claims.WorkspaceIDs()))
	for _, factoryID := range claims.WorkspaceIDs() {
		factory, err := models.FindFactory(database.DB(ctx), claims.OrgID, factoryID)
		if err != nil {
			continue
		}
		if !OrganizationAllowsPublicMCP(factory.OrganizationID) || !organizationAllowsFactories(factory.OrganizationID) {
			return ToolResult{}, ToolError("Not found")
		}
		rows = append(rows, workspaceRow{
			ID:   factory.ID.String(),
			Name: factory.Name,
			Slug: factory.RouteSegment(),
			URL:  workspaceURL(rt.BaseURL, factory.OrganizationID, factory.RouteSegment()),
		})
	}
	return TextResult(mustJSON(map[string]any{"workspaces": rows})), nil
}

func (rt *Runtime) listTasks(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	factory, err := rt.beginWorkspaceCall(ctx, claims, args, "work_orders:read")
	if err != nil {
		return ToolResult{}, err
	}
	req := &pb.ListWorkOrdersRequest{FactoryId: factory.ID.String()}
	if mine, _ := args["mine"].(bool); mine {
		userID := claims.UserID.String()
		req.UserId = &userID
	}
	if beforeID, _ := args["before_id"].(string); strings.TrimSpace(beforeID) != "" {
		req.BeforeId = strings.TrimSpace(beforeID)
	}
	if raw, ok := args["source_url"]; ok && raw != nil {
		sourceURL, ok := raw.(string)
		if !ok {
			return ToolResult{}, ToolError("source_url must be an absolute http URL")
		}
		if sourceURL = strings.TrimSpace(sourceURL); sourceURL != "" {
			req.SourceUrl = sourceURL
		}
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
	links := newTaskLinks(rt.BaseURL, factory)
	type taskRow struct {
		ID        string        `json:"id"`
		Key       string        `json:"key"`
		Title     string        `json:"title"`
		State     string        `json:"state"`
		Assignees []assigneeRef `json:"assignees"`
		SourceURL string        `json:"source_url"`
		URL       string        `json:"url"`
	}
	rows := make([]taskRow, 0, len(resp.GetOrders()))
	for _, order := range resp.GetOrders() {
		sourceURL := ""
		if origin := order.GetOrigin(); origin != nil {
			sourceURL = origin.GetUrl()
		}
		rows = append(rows, taskRow{
			ID:        order.GetId(),
			Key:       order.GetKey(),
			Title:     order.GetTitle(),
			State:     protoStateName(order.GetState()),
			Assignees: assigneeRefs(order.GetAssignees()),
			SourceURL: sourceURL,
			URL:       links.url(order.GetNumber()),
		})
	}
	payload := map[string]any{"tasks": rows, "has_next_page": resp.GetHasNextPage()}
	if resp.GetHasNextPage() && len(rows) > 0 {
		payload["next_before_id"] = rows[len(rows)-1].ID
	}
	return TextResult(mustJSON(payload)), nil
}

func (rt *Runtime) getTask(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	factory, err := rt.beginWorkspaceCall(ctx, claims, args, "work_orders:read")
	if err != nil {
		return ToolResult{}, err
	}
	task, err := stringArg(args, "task")
	if err != nil {
		return ToolResult{}, err
	}
	resp, err := factories.DescribeWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.DescribeWorkOrderRequest{
		FactoryId: factory.ID.String(),
		OrderId:   task,
	})
	if err != nil {
		return ToolResult{}, actionError(err)
	}
	links := newTaskLinks(rt.BaseURL, factory)
	order := resp.GetOrder()
	return TextResult(mustJSON(map[string]any{
		"id":          order.GetId(),
		"key":         order.GetKey(),
		"title":       order.GetTitle(),
		"description": order.GetDescription(),
		"state":       protoStateName(order.GetState()),
		"assignees":   assigneeRefs(order.GetAssignees()),
		"url":         links.url(order.GetNumber()),
	})), nil
}

func (rt *Runtime) listTaskArtifacts(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	factory, err := rt.beginWorkspaceCall(ctx, claims, args, "work_orders:read")
	if err != nil {
		return ToolResult{}, err
	}
	task, err := stringArg(args, "task")
	if err != nil {
		return ToolResult{}, err
	}
	resp, err := factories.ListWorkOrderArtifacts(toolContext(ctx, claims), claims.OrgID.String(), &pb.ListWorkOrderArtifactsRequest{
		FactoryId: factory.ID.String(),
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
	factory, err := rt.beginWorkspaceCall(ctx, claims, args, "work_orders:read")
	if err != nil {
		return ToolResult{}, err
	}
	task, err := stringArg(args, "task")
	if err != nil {
		return ToolResult{}, err
	}
	orderID, err := resolveTaskID(ctx, claims, factory.ID, task)
	if err != nil {
		return ToolResult{}, err
	}
	resp, err := factories.FindPlanningSessionByWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
		FactoryId:   factory.ID.String(),
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
	factory, err := rt.beginWorkspaceCall(ctx, claims, args, "work_orders:update")
	if err != nil {
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
	orderID, err := resolveTaskID(ctx, claims, factory.ID, task)
	if err != nil {
		return ToolResult{}, err
	}
	found, err := factories.FindPlanningSessionByWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
		FactoryId:   factory.ID.String(),
		WorkOrderId: orderID,
	})
	if err != nil {
		if isNotFound(err) {
			return ToolResult{}, ToolError("This task has no agent session.")
		}
		return ToolResult{}, actionError(err)
	}
	resp, err := factories.SendPlanningSessionMessage(toolContext(ctx, claims), claims.OrgID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factory.ID.String(),
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
	factory, err := rt.beginWorkspaceCall(ctx, claims, args, "work_orders:create")
	if err != nil {
		return ToolResult{}, err
	}
	title, err := stringArg(args, "title")
	if err != nil {
		return ToolResult{}, err
	}
	description, _ := args["description"].(string)
	resp, err := factories.CreateWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   factory.ID.String(),
		Title:       title,
		Description: strings.TrimSpace(description),
	})
	if err != nil {
		return ToolResult{}, actionError(err)
	}
	links := newTaskLinks(rt.BaseURL, factory)
	order := resp.GetOrder()
	return TextResult(mustJSON(map[string]any{
		"id":    order.GetId(),
		"key":   order.GetKey(),
		"title": order.GetTitle(),
		"state": protoStateName(order.GetState()),
		"url":   links.url(order.GetNumber()),
	})), nil
}

func (rt *Runtime) handOffTask(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	if err := rt.authorize(ctx, claims, "work_orders:create"); err != nil {
		return ToolResult{}, err
	}
	if err := rt.authorize(ctx, claims, "work_orders:update"); err != nil {
		return ToolResult{}, err
	}
	factory, err := rt.resolveGrantedWorkspace(ctx, claims, args)
	if err != nil {
		return ToolResult{}, err
	}
	title, err := stringArg(args, "title")
	if err != nil {
		return ToolResult{}, err
	}
	plan, err := stringArg(args, "plan")
	if err != nil {
		return ToolResult{}, err
	}
	column, err := stringArg(args, "column")
	if err != nil {
		return ToolResult{}, err
	}
	description, _ := args["description"].(string)
	pullRequestURL, _ := args["pull_request_url"].(string)
	line, _ := args["line"].(string)

	clientName := ClientDisplayName(database.DB(ctx), claims.ClientID)
	result, err := factories.HandOffWorkOrder(toolContext(ctx, claims), rt.Intake, claims.OrgID.String(), factories.HandOffWorkOrderRequest{
		FactoryID:      factory.ID.String(),
		Title:          title,
		Description:    strings.TrimSpace(description),
		Plan:           plan,
		Column:         column,
		PullRequestURL: strings.TrimSpace(pullRequestURL),
		LineName:       strings.TrimSpace(line),
		MCPClientID:    claims.ClientID,
		MCPClientName:  clientName,
	})
	if err != nil {
		return ToolResult{}, actionError(err)
	}
	links := newTaskLinks(rt.BaseURL, factory)
	order := result.Order
	payload := map[string]any{
		"id":     order.GetId(),
		"key":    order.GetKey(),
		"state":  protoStateName(order.GetState()),
		"column": result.Column,
		"url":    links.url(order.GetNumber()),
	}
	if pending := order.GetPendingHandoff(); pending != nil {
		payload["pending_handoff"] = map[string]any{"line_id": pending.GetLineId(), "column": pending.GetColumn(), "pull_request_url": pending.GetPullRequestUrl()}
	}
	if result.PullRequestURL != "" {
		payload["pull_request_url"] = result.PullRequestURL
	}
	return TextResult(mustJSON(payload)), nil
}

func (rt *Runtime) assignTask(ctx context.Context, claims *AccessClaims, args map[string]any) (ToolResult, error) {
	factory, err := rt.beginWorkspaceCall(ctx, claims, args, "work_orders:update")
	if err != nil {
		return ToolResult{}, err
	}
	task, err := stringArg(args, "task")
	if err != nil {
		return ToolResult{}, err
	}
	assignee, unassign, err := assignmentArgs(args)
	if err != nil {
		return ToolResult{}, err
	}
	orderID, err := resolveTaskID(ctx, claims, factory.ID, task)
	if err != nil {
		return ToolResult{}, err
	}

	assigneeIDs := []string{}
	if !unassign {
		if err := rt.authorizeMembersRead(ctx, claims); err != nil {
			return ToolResult{}, err
		}
		member, err := rt.findAssignee(ctx, claims, assignee)
		if err != nil {
			return ToolResult{}, err
		}
		assigneeIDs = []string{member.UserID.String()}
	}

	resp, err := factories.UpdateWorkOrderAssignees(toolContext(ctx, claims), claims.OrgID.String(), &pb.UpdateWorkOrderAssigneesRequest{
		FactoryId:   factory.ID.String(),
		OrderId:     orderID,
		AssigneeIds: assigneeIDs,
	})
	if err != nil {
		return ToolResult{}, actionError(err)
	}
	links := newTaskLinks(rt.BaseURL, factory)
	order := resp.GetOrder()
	return TextResult(mustJSON(map[string]any{
		"id":        order.GetId(),
		"key":       order.GetKey(),
		"assignees": assigneeRefs(order.GetAssignees()),
		"url":       links.url(order.GetNumber()),
	})), nil
}

func (rt *Runtime) listMembers(ctx context.Context, claims *AccessClaims, _ map[string]any) (ToolResult, error) {
	if err := rt.authorize(ctx, claims, "work_orders:read"); err != nil {
		return ToolResult{}, err
	}
	if err := rt.authorizeMembersRead(ctx, claims); err != nil {
		return ToolResult{}, err
	}

	members, err := models.ListFactoryVelocityMembers(database.DB(ctx), claims.OrgID)
	if err != nil {
		return ToolResult{}, ToolError("The request failed.")
	}
	slices.SortFunc(members, func(left, right models.FactoryVelocityMember) int {
		if name := cmp.Compare(left.Name, right.Name); name != 0 {
			return name
		}
		return cmp.Compare(left.UserID.String(), right.UserID.String())
	})

	rows := make([]memberRow, 0, len(members))
	for _, member := range members {
		rows = append(rows, memberRow{
			ID:          member.UserID.String(),
			Name:        member.Name,
			Email:       member.Email,
			GitHubLogin: member.GitHubLogin,
		})
	}
	return TextResult(mustJSON(map[string]any{"members": rows})), nil
}

func (rt *Runtime) findAssignee(ctx context.Context, claims *AccessClaims, assignee string) (models.FactoryVelocityMember, error) {
	members, err := models.ListFactoryVelocityMembers(database.DB(ctx), claims.OrgID)
	if err != nil {
		return models.FactoryVelocityMember{}, ToolError("The request failed.")
	}
	return resolveAssigneeMember(members, assignee)
}

type assigneeRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type memberRow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	GitHubLogin string `json:"github_login"`
}

func assigneeRefs(users []*pb.UserRef) []assigneeRef {
	refs := make([]assigneeRef, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		refs = append(refs, assigneeRef{ID: user.GetId(), Name: user.GetName()})
	}
	return refs
}

func assignmentArgs(args map[string]any) (string, bool, error) {
	assignee, err := optionalStringArg(args, "assignee")
	if err != nil {
		return "", false, err
	}
	unassign, err := optionalBoolArg(args, "unassign")
	if err != nil {
		return "", false, err
	}
	if assignee != "" && unassign {
		return "", false, ToolError("Do not set assignee and unassign.")
	}
	if assignee == "" && !unassign {
		return "", false, ToolError("Set assignee or set unassign to true.")
	}
	return assignee, unassign, nil
}

func optionalStringArg(args map[string]any, name string) (string, error) {
	raw, ok := args[name]
	if !ok || raw == nil {
		return "", nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", ToolError(name + " must be text.")
	}
	return strings.TrimSpace(value), nil
}

func optionalBoolArg(args map[string]any, name string) (bool, error) {
	raw, ok := args[name]
	if !ok || raw == nil {
		return false, nil
	}
	value, ok := raw.(bool)
	if !ok {
		return false, ToolError(name + " must be true or false.")
	}
	return value, nil
}

func resolveAssigneeMember(members []models.FactoryVelocityMember, assignee string) (models.FactoryVelocityMember, error) {
	query := strings.TrimSpace(assignee)
	if id, err := uuid.Parse(query); err == nil {
		for _, member := range members {
			if member.UserID == id {
				return member, nil
			}
		}
	}

	emailMatches := membersMatching(members, query, func(member models.FactoryVelocityMember) string {
		return member.Email
	})
	if len(emailMatches) > 1 {
		return models.FactoryVelocityMember{}, ambiguousMemberError()
	}
	if len(emailMatches) == 1 {
		return emailMatches[0], nil
	}

	loginMatches := membersMatching(members, query, func(member models.FactoryVelocityMember) string {
		return member.GitHubLogin
	})
	if len(loginMatches) > 1 {
		return models.FactoryVelocityMember{}, ambiguousMemberError()
	}
	if len(loginMatches) == 1 {
		return loginMatches[0], nil
	}
	return models.FactoryVelocityMember{}, ToolError("The member was not found.")
}

func membersMatching(members []models.FactoryVelocityMember, query string, field func(models.FactoryVelocityMember) string) []models.FactoryVelocityMember {
	folded := strings.ToLower(query)
	matches := make([]models.FactoryVelocityMember, 0)
	for _, member := range members {
		value := strings.ToLower(strings.TrimSpace(field(member)))
		if value == "" || value != folded {
			continue
		}
		matches = append(matches, member)
	}
	return matches
}

func ambiguousMemberError() error {
	return ToolError("More than one member matches. Use a user id.")
}

func (rt *Runtime) authorizeMembersRead(ctx context.Context, claims *AccessClaims) error {
	allowed, err := rt.Auth.CheckOrganizationPermission(ctx, claims.UserID.String(), claims.OrgID.String(), "members", "read")
	if err != nil || !allowed {
		return ToolError("Not found")
	}
	return nil
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
	ctx = productanalytics.WithSource(ctx, productanalytics.SourceMCP)
	return context.WithValue(ctx, authorization.OrganizationContextKey, claims.OrgID.String())
}

func resolveTaskID(ctx context.Context, claims *AccessClaims, factoryID uuid.UUID, task string) (string, error) {
	resp, err := factories.DescribeWorkOrder(toolContext(ctx, claims), claims.OrgID.String(), &pb.DescribeWorkOrderRequest{
		FactoryId: factoryID.String(),
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

type workspaceRow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	URL  string `json:"url"`
}

func (rt *Runtime) beginWorkspaceCall(ctx context.Context, claims *AccessClaims, args map[string]any, scope string) (*models.Factory, error) {
	if err := rt.authorize(ctx, claims, scope); err != nil {
		return nil, err
	}
	return rt.resolveGrantedWorkspace(ctx, claims, args)
}

func (rt *Runtime) resolveGrantedWorkspace(ctx context.Context, claims *AccessClaims, args map[string]any) (*models.Factory, error) {
	requested, err := optionalStringArg(args, "workspace")
	if err != nil {
		return nil, err
	}
	granted := claims.WorkspaceIDs()
	var factoryID uuid.UUID
	switch {
	case requested == "" && len(granted) == 1:
		factoryID = granted[0]
	case requested == "" && len(granted) > 1:
		return nil, ToolError("This connection can use more than one workspace. Pass workspace.")
	default:
		parsed, parseErr := uuid.Parse(requested)
		if parseErr != nil || !slices.Contains(granted, parsed) {
			return nil, ToolError("Not found")
		}
		factoryID = parsed
	}
	if !OrganizationAllowsPublicMCP(claims.OrgID) || !organizationAllowsFactories(claims.OrgID) {
		return nil, ToolError("Not found")
	}
	factory, err := models.FindFactory(database.DB(ctx), claims.OrgID, factoryID)
	if err != nil || factory.OrganizationID != claims.OrgID {
		return nil, ToolError("Not found")
	}
	if !OrganizationAllowsPublicMCP(factory.OrganizationID) || !organizationAllowsFactories(factory.OrganizationID) {
		return nil, ToolError("Not found")
	}
	return factory, nil
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
