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
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
)

const AppAgentToolName = "superplane_app"

func init() {
	Register[canvasactions.Input](AppAgentToolName, func(deps Dependencies) AgentTool[canvasactions.Input] {
		return NewAppAgentTool(AppAgentToolOptions{
			Encryptor:      deps.Encryptor,
			Registry:       deps.ComponentRegistry,
			GitProvider:    deps.GitProvider,
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
	GitProvider    gitprovider.Provider
	WebhookBaseURL string
	AuthService    authorization.Authorization
	UsageService   usage.Service
}

func NewAppAgentTool(opts AppAgentToolOptions) *AppAgentTool {
	return &AppAgentTool{
		actions: canvasactions.NewDefaultRegistry(canvasactions.Dependencies{
			Encryptor:      opts.Encryptor,
			Registry:       opts.Registry,
			GitProvider:    opts.GitProvider,
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
	return "Inspect access, read the current SuperPlane app, create drafts, update draft canvas/Console state, manage app repository files, list connected integrations, list integration resources, and read runtime data. This is the only way to reach the app; there is no command line or HTTP API to call. The tool is bound to the current agent session's canvas and rejects attempts to access any other canvas. It never publishes drafts. Use patch_draft for graph edits, Console updates, or auto-layout without sending full canvas YAML; patch_draft and file write actions require the exact version_id returned by read, create_draft, or the previous draft update."
}

func (t *AppAgentTool) InputSchema() agents.CustomToolInputSchema {
	return agents.CustomToolInputSchema{
		Type: "object",
		Properties: map[string]agents.CustomToolInputSchema{
			"action": {
				Type:        "string",
				Enum:        t.actions.Names(),
				Description: "Operation to run. Use access to inspect token-backed API capabilities, read for current YAML, read_runtime for memory/runs/events/executions/queues, list_files/read_file for app repository files and AGENTS.md context, create_draft when read returns live/no version_id or when intentionally creating another draft branch, write_file/delete_file to stage normal file changes, commit_files to commit staged draft file changes, patch_draft to apply graph edits, Console updates, or auto-layout without sending full canvas YAML, list_integrations for connected integration IDs, and list_resources for integration-backed resource values.",
			},
			"canvas_id": {
				Type:        "string",
				Description: "Optional safety check. If provided, it must match the current session canvas_id from the preamble.",
			},
			"use_draft": {
				Type:        "boolean",
				Description: "For read. Defaults to true: return the current user's draft when exactly one exists, otherwise live. If multiple owned drafts exist, pass version_id or set use_draft=false.",
			},
			"version_id": {
				Type:        "string",
				Description: "For read, read_file, write_file, delete_file, commit_files, and patch_draft. Draft version ID returned by read, create_draft, or a previous patch_draft. Required for patch_draft and file write/commit actions. If read returns source live with no version_id, call create_draft before updating. For read/read_file, use it to select a specific draft when multiple owned drafts exist. The backend validates that it belongs to the current user and canvas and is still a registered draft branch.",
			},
			"draft_version_id": {
				Type:        "string",
				Description: "Alias for version_id for read, read_file, write_file, delete_file, commit_files, and patch_draft. Use only one of version_id or draft_version_id.",
			},
			"display_name": {
				Type:        "string",
				Description: "For create_draft. Optional user-facing draft display name. If omitted, the backend assigns the next Draft #N name.",
			},
			"include_console": {
				Type:        "boolean",
				Description: "For read. Include console.yaml in the response.",
			},
			"include_canvas_yaml": {
				Type:        "boolean",
				Description: "For read. Defaults to false so read stays compact; set true only when you need the complete canvas.yaml text. Compact reads still return summary, version_id, canvas_yaml_bytes, and canvas_yaml_omitted.",
			},
			"include_integrations": {
				Type:        "boolean",
				Description: "For read. Include connected integration IDs, names, vendors, and state.",
			},
			"integration_id": {
				Type:        "string",
				Description: "For list_resources. Connected integration ID returned by list_integrations or read with include_integrations.",
			},
			"resource_type": {
				Type:        "string",
				Description: "For list_resources. Integration resource type to list, for example repository, model, project, workflow, service, or application. Use the resource type requested by the component schema field.",
			},
			"parameters": {
				Type:        "object",
				Description: "For list_resources. Optional provider-specific string parameters. The backend also receives resource_type as parameter type.",
			},
			"path": {
				Type:        "string",
				Description: "For read_file, write_file, and delete_file. Repository-relative app file path, such as AGENTS.md, README.md, or scripts/runner.py. Paths under .superplane and unsafe paths are rejected. Use patch_draft for canvas.yaml and console.yaml.",
			},
			"paths": {
				Type:        "array",
				Description: "For read_file. Optional repository-relative paths to read in one call.",
				Items:       &agents.CustomToolInputSchema{Type: "string"},
			},
			"content": {
				Type:        "string",
				Description: "For write_file. Complete file content to stage on the selected draft version.",
			},
			"message": {
				Type:        "string",
				Description: "For commit_files. Optional commit message for staged repository file changes; defaults to Update files.",
			},
			"query": {
				Type:        "string",
				Description: "For list_files. Optional case-insensitive path filter, for example AGENTS.md or README.",
			},
			"console_yaml": {
				Type:        "string",
				Description: "For patch_draft. Complete canonical console.yaml content to stage alongside graph patch operations or by itself.",
			},
			"patch_operations": patchOperationsSchema(),
			"auto_layout": {
				Type:        "object",
				Description: "Optional patch_draft auto-layout settings. If omitted for graph patch operations, the backend applies horizontal connected-component layout seeded by affected nodes and edge endpoints. Pass scope full_canvas to re-layout the whole graph, or connected_component with node_ids to choose seeds. Passing auto_layout without patch_operations applies layout only.",
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
				Enum:        []string{"memory", "runs", "event_executions", "node_executions", "node_queue_items", "node_events"},
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
				Description: "Reserved for future runtime resources that target a specific execution.",
			},
			"limit": {
				Type:        "integer",
				Description: "For read_runtime paginated resources and list_resources. Backend defaults apply when omitted; list_resources caps results to keep responses concise.",
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

func patchOperationsSchema() agents.CustomToolInputSchema {
	return agents.CustomToolInputSchema{
		Type:        "array",
		Description: "For patch_draft. Ordered graph edits applied to the selected draft without sending full canvas YAML. Supported op values: add_node, update_node, delete_node, add_edge, delete_edge. Aliases replace_node/remove_node/remove_edge are accepted. update_node can change name, configuration, position, and is_collapsed; use delete_node plus add_node to change component/integration.",
		Items: &agents.CustomToolInputSchema{
			Type: "object",
			Properties: map[string]agents.CustomToolInputSchema{
				"op": {
					Type:        "string",
					Enum:        []string{"add_node", "update_node", "delete_node", "add_edge", "delete_edge", "replace_node", "remove_node", "remove_edge"},
					Description: "Patch operation to apply.",
				},
				"node_id": {
					Type:        "string",
					Description: "For delete_node, or as an ID fallback for update_node.",
				},
				"node": patchNodeSchema(),
				"edge": patchEdgeSchema(),
			},
			Required: []string{"op"},
		},
	}
}

func patchNodeSchema() agents.CustomToolInputSchema {
	return agents.CustomToolInputSchema{
		Type:        "object",
		Description: "Node payload for add_node or update_node.",
		Properties: map[string]agents.CustomToolInputSchema{
			"id": {
				Type:        "string",
				Description: "Stable node ID.",
			},
			"name": {
				Type:        "string",
				Description: "Human-readable node name.",
			},
			"component": {
				Type:        "string",
				Description: "Component, trigger, or widget block name for add_node, for example http, noop, or github.createIssue. update_node ignores component.",
			},
			"configuration": {
				Type:        "object",
				Description: "Node configuration object. For update_node, this replaces the existing configuration when provided.",
			},
			"integration_id": {
				Type:        "string",
				Description: "Connected integration ID required for non-core blocks on add_node. update_node ignores integration_id.",
			},
			"position": {
				Type: "object",
				Properties: map[string]agents.CustomToolInputSchema{
					"x": {Type: "integer"},
					"y": {Type: "integer"},
				},
			},
			"is_collapsed": {
				Type:        "boolean",
				Description: "Whether the node is collapsed in the editor.",
			},
		},
	}
}

func patchEdgeSchema() agents.CustomToolInputSchema {
	return agents.CustomToolInputSchema{
		Type:        "object",
		Description: "Edge payload for add_edge or delete_edge. channel defaults to default when omitted.",
		Properties: map[string]agents.CustomToolInputSchema{
			"source_id": {
				Type:        "string",
				Description: "Source node ID.",
			},
			"target_id": {
				Type:        "string",
				Description: "Target node ID.",
			},
			"channel": {
				Type:        "string",
				Description: "Source output channel. Defaults to default.",
			},
		},
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
