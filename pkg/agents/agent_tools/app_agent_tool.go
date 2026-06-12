package agenttools

import (
	"context"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/agents"
	canvasactions "github.com/superplanehq/superplane/pkg/agents/agent_tools/actions"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
)

const AppAgentToolName = "superplane_app"

func init() {
	Register[canvasactions.Input](AppAgentToolName, func(deps Dependencies) AgentTool[canvasactions.Input] {
		return NewAppAgentTool(AppAgentToolOptions{
			Encryptor:      deps.Encryptor,
			Registry:       deps.ComponentRegistry,
			WebhookBaseURL: deps.WebhookBaseURL,
			AuthService:    deps.AuthService,
			UsageService:   deps.UsageService,
		})
	})
}

var _ AgentTool[canvasactions.Input] = (*AppAgentTool)(nil)

type AppAgentTool struct {
	actions *canvasactions.Registry
}

type AppAgentToolOptions struct {
	Encryptor      crypto.Encryptor
	Registry       *registry.Registry
	WebhookBaseURL string
	AuthService    authorization.Authorization
	UsageService   usage.Service
}

func NewAppAgentTool(opts AppAgentToolOptions) *AppAgentTool {
	return &AppAgentTool{
		actions: canvasactions.NewDefaultRegistry(canvasactions.Dependencies{
			Encryptor:      opts.Encryptor,
			Registry:       opts.Registry,
			WebhookBaseURL: opts.WebhookBaseURL,
			AuthService:    opts.AuthService,
			UsageService:   opts.UsageService,
		}),
	}
}

func (t *AppAgentTool) Name() string {
	return AppAgentToolName
}

func (t *AppAgentTool) Description() string {
	return "Inspect access, read, and update the current SuperPlane app canvas. This is the only way to reach the app; there is no command line or HTTP API to call. Use access to check the current session's interceptor-backed permissions, read for canvas YAML, read_runtime for memory/runs/events/executions/queues, list_integrations for connected integration IDs, and update_draft to save draft graph or Console changes. The tool is bound to the current agent session's canvas and will reject attempts to access any other canvas. It never publishes drafts; update_draft only creates or updates the caller's private draft and returns the draft version ID plus validation issues."
}

func (t *AppAgentTool) InputSchema() agents.CustomToolInputSchema {
	return agents.CustomToolInputSchema{
		Type: "object",
		Properties: map[string]agents.CustomToolInputSchema{
			"action": {
				Type:        "string",
				Enum:        t.actions.Names(),
				Description: "Operation to run. Use access to inspect token-backed API capabilities, read for current YAML, read_runtime for memory/runs/events/executions/queues, update_draft to save canvas_yaml and/or console_yaml to a draft, and list_integrations for connected integration IDs.",
			},
			"canvas_id": {
				Type:        "string",
				Description: "Optional safety check. If provided, it must match the current session canvas_id from the preamble.",
			},
			"use_draft": {
				Type:        "boolean",
				Description: "For read. Defaults to true: return the current user's draft when one exists, otherwise live.",
			},
			"include_console": {
				Type:        "boolean",
				Description: "For read. Include console.yaml in the response.",
			},
			"include_integrations": {
				Type:        "boolean",
				Description: "For read. Include connected integration IDs, names, vendors, and state.",
			},
			"canvas_yaml": {
				Type:        "string",
				Description: "For update_draft. Complete canonical canvas.yaml content to save.",
			},
			"console_yaml": {
				Type:        "string",
				Description: "For update_draft. Complete canonical console.yaml content to save.",
			},
			"auto_layout": {
				Type:        "object",
				Description: "Optional auto-layout settings for canvas_yaml updates. If omitted for a canvas_yaml update, the backend applies horizontal full-canvas auto-layout by default. Omit this for console-only updates.",
				Properties: map[string]agents.CustomToolInputSchema{
					"scope": {
						Type: "string",
						Enum: []string{"full_canvas", "connected_component"},
					},
					"node_ids": {
						Type:  "array",
						Items: &agents.CustomToolInputSchema{Type: "string"},
					},
				},
			},
			"resource": {
				Type:        "string",
				Enum:        []string{"memory", "runs", "canvas_events", "event_executions", "node_executions", "node_queue_items", "node_events", "child_executions"},
				Description: "For read_runtime. Defaults to memory. Selects the canvas-scoped runtime data to read.",
			},
			"namespace": {
				Type:        "string",
				Description: "For read_runtime resource memory. Optional client-side namespace filter.",
			},
			"node_id": {
				Type:        "string",
				Description: "For read_runtime resources node_executions, node_queue_items, and node_events.",
			},
			"event_id": {
				Type:        "string",
				Description: "For read_runtime resource event_executions.",
			},
			"execution_id": {
				Type:        "string",
				Description: "For read_runtime resource child_executions.",
			},
			"limit": {
				Type:        "integer",
				Description: "For read_runtime paginated resources. Backend defaults apply when omitted.",
			},
			"before": {
				Type:        "string",
				Description: "For read_runtime paginated resources. RFC3339 timestamp cursor.",
			},
			"states": {
				Type:        "array",
				Description: "For read_runtime resources runs and node_executions. Use started/finished for runs; pending/started/finished for node executions.",
				Items:       &agents.CustomToolInputSchema{Type: "string"},
			},
			"results": {
				Type:        "array",
				Description: "For read_runtime resources runs and node_executions. Use passed/failed/cancelled for runs; passed/failed for node executions.",
				Items:       &agents.CustomToolInputSchema{Type: "string"},
			},
		},
		Required: []string{"action"},
	}
}

func (t *AppAgentTool) Call(ctx context.Context, session agents.AgentSessionContext, input canvasactions.Input) (Result, error) {
	if err := t.validateSessionBoundInput(session, input.CanvasID); err != nil {
		return Result{}, err
	}

	authedCtx := authentication.SetUserIdInMetadata(ctx, session.UserID)
	payload, err := t.actions.Execute(authedCtx, session, input)
	if err != nil {
		return Result{}, err
	}

	return Result{Payload: payload}, nil
}

func (t *AppAgentTool) validateSessionBoundInput(session agents.AgentSessionContext, requestedCanvasID string) error {
	if strings.TrimSpace(session.CanvasID) == "" || strings.TrimSpace(session.OrganizationID) == "" || strings.TrimSpace(session.UserID) == "" {
		return fmt.Errorf("agent session context is incomplete")
	}
	requestedCanvasID = strings.TrimSpace(requestedCanvasID)
	if requestedCanvasID != "" && requestedCanvasID != session.CanvasID {
		return fmt.Errorf("canvas_id %q is outside this agent session", requestedCanvasID)
	}
	return nil
}
