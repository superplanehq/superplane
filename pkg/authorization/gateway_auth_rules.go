package authorization

import (
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
)

func DefaultAuthorizationRules() map[HTTPRoute]AuthorizationRule {
	return map[HTTPRoute]AuthorizationRule{
		{Method: "DELETE", Pattern: "/api/v1/canvas-folders/{id}"}: {
			Resource:   "canvases",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "DELETE", Pattern: "/api/v1/canvases/{canvas_id}/memory/{memory_id}"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "DELETE", Pattern: "/api/v1/canvases/{canvas_id}/nodes/{node_id}/queue/{item_id}"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "DELETE", Pattern: "/api/v1/canvases/{canvas_id}/staging"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
			LegacyActions:      []string{"update_version"},
		},
		{Method: "DELETE", Pattern: "/api/v1/canvases/{id}"}: {
			Resource:           "canvases",
			Action:             "delete",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{IDPathParam},
		},
		{Method: "DELETE", Pattern: "/api/v1/groups/{group_name}"}: {
			Resource:   "groups",
			Action:     "delete",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "DELETE", Pattern: "/api/v1/organizations/{id}"}: {
			Resource:   "org",
			Action:     "delete",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "DELETE", Pattern: "/api/v1/organizations/{id}/integrations/{integration_id}"}: {
			Resource:   "integrations",
			Action:     "delete",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "DELETE", Pattern: "/api/v1/organizations/{id}/users/{user_id}"}: {
			Resource:   "members",
			Action:     "delete",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "DELETE", Pattern: "/api/v1/roles/{role_name}"}: {
			Resource:   "roles",
			Action:     "delete",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "DELETE", Pattern: "/api/v1/secrets/{id_or_name}"}: {
			Resource:   "secrets",
			Action:     "delete",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "DELETE", Pattern: "/api/v1/secrets/{id_or_name}/keys/{key_name}"}: {
			Resource:   "secrets",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "DELETE", Pattern: "/api/v1/api-keys/{id}"}: {
			Resource:   "api_keys",
			Action:     "delete",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/actions"}: {
			Resource:   "org",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/actions/{name}"}: {
			Resource:   "org",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/agents/canvases/{canvas_id}/chat"}: {
			Resource:                     "agents",
			Action:                       "create",
			DomainType:                   models.DomainTypeOrganization,
			RequiredExperimentalFeatures: []string{features.FeatureClaudeManagedAgents},
		},
		{Method: "GET", Pattern: "/api/v1/agents/chats/{chat_id}/messages"}: {
			Resource:                     "agents",
			Action:                       "read",
			DomainType:                   models.DomainTypeOrganization,
			RequiredExperimentalFeatures: []string{features.FeatureClaudeManagedAgents},
		},
		{Method: "GET", Pattern: "/api/v1/canvas-folders"}: {
			Resource:   "canvases",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/canvases"}: {
			Resource:   "canvases",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/canvases/{canvas_id}/events/{event_id}/executions"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "GET", Pattern: "/api/v1/canvases/{canvas_id}/memory"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "GET", Pattern: "/api/v1/canvases/{canvas_id}/nodes/{node_id}/events"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "GET", Pattern: "/api/v1/canvases/{canvas_id}/nodes/{node_id}/executions"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "GET", Pattern: "/api/v1/canvases/{canvas_id}/nodes/{node_id}/queue"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "GET", Pattern: "/api/v1/canvases/{canvas_id}/repository"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "GET", Pattern: "/api/v1/canvases/{canvas_id}/repository/files"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "GET", Pattern: "/api/v1/canvases/{canvas_id}/runs"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "GET", Pattern: "/api/v1/canvases/{canvas_id}/runs/{run_id}"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "PATCH", Pattern: "/api/v1/canvases/{canvas_id}/runs/{run_id}/cancel"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "GET", Pattern: "/api/v1/canvases/{canvas_id}/versions"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "GET", Pattern: "/api/v1/canvases/{canvas_id}/staging"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "GET", Pattern: "/api/v1/canvases/{canvas_id}/versions/{version_id}"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "GET", Pattern: "/api/v1/canvases/{id}"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{IDPathParam},
		},
		{Method: "GET", Pattern: "/api/v1/groups"}: {
			Resource:   "groups",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/groups/{group_name}"}: {
			Resource:   "groups",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/groups/{group_name}/users"}: {
			Resource:   "groups",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/integrations"}: {
			Resource:   "org",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/organizations/{id}"}: {
			Resource:   "org",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/organizations/{id}/integrations"}: {
			Resource:   "integrations",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/organizations/{id}/integrations/{integration_id}"}: {
			Resource:   "integrations",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/organizations/{id}/integrations/{integration_id}/resources"}: {
			Resource:   "integrations",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/organizations/{id}/integrations/{integration_id}/tools"}: {
			Resource:   "integrations",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "POST", Pattern: "/api/v1/organizations/{id}/integrations/{integration_id}/tools"}: {
			Resource:   "integrations",
			Action:     "update", // TODO: figure out the right permission here
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/organizations/{id}/invite-link"}: {
			Resource:   "members",
			Action:     "create",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/organizations/{id}/usage"}: {
			Resource:   "org",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/roles"}: {
			Resource:   "roles",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/roles/{role_name}"}: {
			Resource:   "roles",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/secrets"}: {
			Resource:   "secrets",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/secrets/{id_or_name}"}: {
			Resource:   "secrets",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/api-keys"}: {
			Resource:   "api_keys",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/api-keys/{id}"}: {
			Resource:   "api_keys",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/triggers"}: {
			Resource:   "org",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/triggers/{name}"}: {
			Resource:   "org",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "GET", Pattern: "/api/v1/users"}: {
			Resource:   "members",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PATCH", Pattern: "/api/v1/canvas-folders/{id}/position"}: {
			Resource:   "canvases",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PATCH", Pattern: "/api/v1/canvases/{canvas_id}/executions/resolve"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "PATCH", Pattern: "/api/v1/canvases/{canvas_id}/executions/{execution_id}/cancel"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "POST", Pattern: "/api/v1/canvases/{canvas_id}/staging/commit"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "PUT", Pattern: "/api/v1/canvases/{canvas_id}/staging"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
			LegacyActions:      []string{"update_version"},
		},
		{Method: "PATCH", Pattern: "/api/v1/groups/{group_name}/users/remove"}: {
			Resource:   "groups",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PATCH", Pattern: "/api/v1/organizations/{id}"}: {
			Resource:   "org",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PATCH", Pattern: "/api/v1/organizations/{id}/integrations/{integration_id}"}: {
			Resource:   "integrations",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PATCH", Pattern: "/api/v1/organizations/{id}/integrations/{integration_id}/next"}: {
			Resource:   "integrations",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PATCH", Pattern: "/api/v1/organizations/{id}/integrations/{integration_id}/previous"}: {
			Resource:   "integrations",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PATCH", Pattern: "/api/v1/organizations/{id}/invite-link"}: {
			Resource:   "members",
			Action:     "create",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PATCH", Pattern: "/api/v1/secrets/{id_or_name}"}: {
			Resource:   "secrets",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PATCH", Pattern: "/api/v1/secrets/{id_or_name}/name"}: {
			Resource:   "secrets",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PATCH", Pattern: "/api/v1/api-keys/{id}"}: {
			Resource:   "api_keys",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "POST", Pattern: "/api/v1/agents/canvases/{canvas_id}/chat/reset"}: {
			Resource:                     "agents",
			Action:                       "create",
			DomainType:                   models.DomainTypeOrganization,
			RequiredExperimentalFeatures: []string{features.FeatureClaudeManagedAgents},
		},
		{Method: "POST", Pattern: "/api/v1/agents/chats/{chat_id}/interrupt"}: {
			Resource:                     "agents",
			Action:                       "create",
			DomainType:                   models.DomainTypeOrganization,
			RequiredExperimentalFeatures: []string{features.FeatureClaudeManagedAgents},
		},
		{Method: "POST", Pattern: "/api/v1/agents/chats/{chat_id}/messages"}: {
			Resource:                     "agents",
			Action:                       "create",
			DomainType:                   models.DomainTypeOrganization,
			RequiredExperimentalFeatures: []string{features.FeatureClaudeManagedAgents},
		},
		{Method: "POST", Pattern: "/api/v1/agents/chats/{chat_id}/outcome"}: {
			Resource:                     "agents",
			Action:                       "create",
			DomainType:                   models.DomainTypeOrganization,
			RequiredExperimentalFeatures: []string{features.FeatureClaudeManagedAgents},
		},
		{Method: "POST", Pattern: "/api/v1/canvas-folders"}: {
			Resource:   "canvases",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "POST", Pattern: "/api/v1/canvases"}: {
			Resource:   "canvases",
			Action:     "create",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "POST", Pattern: "/api/v1/canvases/{canvas_id}/executions/{execution_id}/hooks/{hook_name}"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "POST", Pattern: "/api/v1/canvases/{canvas_id}/memory/namespaces"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "POST", Pattern: "/api/v1/canvases/{canvas_id}/triggers/{node_id}/events/{event_id}/reemit"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "POST", Pattern: "/api/v1/canvases/{canvas_id}/triggers/{node_id}/hooks/{hook_name}"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "POST", Pattern: "/api/v1/groups"}: {
			Resource:   "groups",
			Action:     "create",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "POST", Pattern: "/api/v1/groups/{group_name}/users"}: {
			Resource:   "groups",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "POST", Pattern: "/api/v1/organizations/{id}/integrations"}: {
			Resource:   "integrations",
			Action:     "create",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "POST", Pattern: "/api/v1/organizations/{id}/invite-link/reset"}: {
			Resource:   "members",
			Action:     "create",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "POST", Pattern: "/api/v1/roles"}: {
			Resource:   "roles",
			Action:     "create",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "POST", Pattern: "/api/v1/roles/{role_name}/users"}: {
			Resource:   "members",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "POST", Pattern: "/api/v1/secrets"}: {
			Resource:   "secrets",
			Action:     "create",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "POST", Pattern: "/api/v1/api-keys"}: {
			Resource:   "api_keys",
			Action:     "create",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "POST", Pattern: "/api/v1/api-keys/{id}/token"}: {
			Resource:   "api_keys",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PUT", Pattern: "/api/v1/canvas-folders/{id}"}: {
			Resource:   "canvases",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PUT", Pattern: "/api/v1/canvases/{canvas_id}/memory/namespaces/{namespace}"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "PUT", Pattern: "/api/v1/canvases/{canvas_id}/preference"}: {
			Resource:           "canvases",
			Action:             "read",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{CanvasIDPathParam},
		},
		{Method: "PUT", Pattern: "/api/v1/canvases/{id}"}: {
			Resource:           "canvases",
			Action:             "update",
			DomainType:         models.DomainTypeOrganization,
			ResourcePathParams: []string{IDPathParam},
		},
		{Method: "PUT", Pattern: "/api/v1/groups/{group_name}"}: {
			Resource:   "groups",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PUT", Pattern: "/api/v1/organizations/{id}/integrations/{integration_id}/capabilities"}: {
			Resource:   "integrations",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PUT", Pattern: "/api/v1/organizations/{id}/integrations/{integration_id}/properties"}: {
			Resource:   "integrations",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PUT", Pattern: "/api/v1/organizations/{id}/integrations/{integration_id}/secrets"}: {
			Resource:   "integrations",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PUT", Pattern: "/api/v1/roles/{role_name}"}: {
			Resource:   "roles",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
		{Method: "PUT", Pattern: "/api/v1/secrets/{id_or_name}/keys/{key_name}"}: {
			Resource:   "secrets",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		},
	}
}
