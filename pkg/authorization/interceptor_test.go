package authorization

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pbAgents "github.com/superplanehq/superplane/pkg/protos/agents"
	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func TestDefaultResourceResolver(t *testing.T) {
	t.Run("returns request id when available", func(t *testing.T) {
		resourceIDs := defaultResourceResolver(&pbCanvases.DescribeCanvasRequest{Id: "canvas-123"})
		require.Equal(t, []string{"canvas-123"}, resourceIDs)
	})

	t.Run("returns nil when request does not expose an id", func(t *testing.T) {
		resourceIDs := defaultResourceResolver(&pbCanvases.ListCanvasesRequest{})
		assert.Nil(t, resourceIDs)
	})
}

func TestCanvasResourceResolver(t *testing.T) {
	t.Run("returns canvas id when available", func(t *testing.T) {
		resourceIDs := canvasResourceResolver(&pbCanvases.ListCanvasEventsRequest{CanvasId: "canvas-123"})
		require.Equal(t, []string{"canvas-123"}, resourceIDs)
	})

	t.Run("returns canvas id for list runs", func(t *testing.T) {
		resourceIDs := canvasResourceResolver(&pbCanvases.ListRunsRequest{CanvasId: "canvas-123"})
		require.Equal(t, []string{"canvas-123"}, resourceIDs)
	})

	t.Run("returns nil when request does not expose a canvas id", func(t *testing.T) {
		resourceIDs := canvasResourceResolver(&pbCanvases.ListCanvasesRequest{})
		assert.Nil(t, resourceIDs)
	})
}

func TestCanvasAuthorizationRulesSeparateDraftAndLiveActions(t *testing.T) {
	interceptor := NewAuthorizationInterceptor(nil)

	tests := []struct {
		method string
		action string
	}{
		{pbCanvases.Canvases_CreateCanvasVersion_FullMethodName, "update_version"},
		{pbCanvases.Canvases_UpdateCanvasVersion_FullMethodName, "update_version"},
		{pbCanvases.Canvases_ApplyCanvasVersionChangeset_FullMethodName, "update_version"},
		{pbCanvases.Canvases_DeleteCanvasVersion_FullMethodName, "update_version"},
		{pbCanvases.Canvases_PublishCanvasVersion_FullMethodName, "publish"},
		{pbCanvases.Canvases_ActOnCanvasChangeRequest_FullMethodName, "publish"},
		{pbCanvases.Canvases_UpdateCanvas_FullMethodName, "update"},
		{pbCanvases.Canvases_DeleteCanvas_FullMethodName, "delete"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			rule, ok := interceptor.rules[tt.method]
			require.True(t, ok)
			assert.Equal(t, "canvases", rule.Resource)
			assert.Equal(t, tt.action, rule.Action)
		})
	}
}

func TestHasRequiredScopedTokenPermission(t *testing.T) {
	ruleWithDefaultResolver := AuthorizationRule{
		Resource:         "canvases",
		Action:           "read",
		DomainType:       models.DomainTypeOrganization,
		ResourceResolver: defaultResourceResolver,
	}
	ruleWithCanvasResolver := AuthorizationRule{
		Resource:         "canvases",
		Action:           "read",
		DomainType:       models.DomainTypeOrganization,
		ResourceResolver: canvasResourceResolver,
	}

	tests := []struct {
		name        string
		ctx         context.Context
		req         any
		rule        AuthorizationRule
		expectAllow bool
	}{
		{
			name:        "allows request without metadata",
			ctx:         context.Background(),
			req:         &pbCanvases.ListCanvasesRequest{},
			rule:        ruleWithDefaultResolver,
			expectAllow: true,
		},
		{
			name:        "allows request without scoped token scopes metadata",
			ctx:         metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "user-123")),
			req:         &pbCanvases.ListCanvasesRequest{},
			rule:        ruleWithDefaultResolver,
			expectAllow: true,
		},
		{
			name: "rejects malformed scoped token scopes metadata",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs("x-token-scopes", "not-json"),
			),
			req:         &pbCanvases.ListCanvasesRequest{},
			rule:        ruleWithDefaultResolver,
			expectAllow: false,
		},
		{
			name: "allows matching permission without resource scoping",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(
					"x-token-scopes",
					marshalScopes(t, []string{"canvases:read"}),
				),
			),
			req:         &pbCanvases.ListCanvasesRequest{},
			rule:        ruleWithDefaultResolver,
			expectAllow: true,
		},
		{
			name: "allows matching permission with default id resolver",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(
					"x-token-scopes",
					marshalScopes(t, []string{"canvases:read:canvas-123"}),
				),
			),
			req:         &pbCanvases.DescribeCanvasRequest{Id: "canvas-123"},
			rule:        ruleWithDefaultResolver,
			expectAllow: true,
		},
		{
			name: "rejects resource scoped permission when request has no resolvable resource id",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(
					"x-token-scopes",
					marshalScopes(t, []string{"canvases:read:canvas-123"}),
				),
			),
			req:         &pbCanvases.ListCanvasesRequest{},
			rule:        ruleWithDefaultResolver,
			expectAllow: false,
		},
		{
			name: "allows matching permission with canvas resolver",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(
					"x-token-scopes",
					marshalScopes(t, []string{"canvases:read:canvas-123"}),
				),
			),
			req:         &pbCanvases.ListCanvasEventsRequest{CanvasId: "canvas-123"},
			rule:        ruleWithCanvasResolver,
			expectAllow: true,
		},
		{
			name: "rejects non matching permission with canvas resolver",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(
					"x-token-scopes",
					marshalScopes(t, []string{"canvases:read:canvas-456"}),
				),
			),
			req:         &pbCanvases.ListCanvasEventsRequest{CanvasId: "canvas-123"},
			rule:        ruleWithCanvasResolver,
			expectAllow: false,
		},
		{
			name: "rejects permission with wrong action",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(
					"x-token-scopes",
					marshalScopes(t, []string{"canvases:update"}),
				),
			),
			req:         &pbCanvases.ListCanvasEventsRequest{CanvasId: "canvas-123"},
			rule:        ruleWithCanvasResolver,
			expectAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := hasRequiredScopedTokenPermission(tt.ctx, tt.req, tt.rule)
			assert.Equal(t, tt.expectAllow, allowed)
		})
	}
}

func TestMetadataScopedTokenPermissions(t *testing.T) {
	t.Run("returns nil when metadata does not include scoped token scopes", func(t *testing.T) {
		permissions, err := metadataScopedTokenPermissions(metadata.Pairs("x-user-id", "user-123"))
		require.NoError(t, err)
		assert.Nil(t, permissions)
	})

	t.Run("parses scoped token scopes from metadata", func(t *testing.T) {
		permissions, err := metadataScopedTokenPermissions(
			metadata.Pairs(
				"x-token-scopes",
				marshalScopes(t, []string{"canvases:read:canvas-123"}),
			),
		)

		require.NoError(t, err)
		require.Len(t, permissions, 1)
		assert.Equal(t, "canvases", permissions[0].ResourceType)
		assert.Equal(t, "read", permissions[0].Action)
		assert.Equal(t, []string{"canvas-123"}, permissions[0].Resources)
	})
}

func TestAgentRoutesRequireManagedAgentsFeature(t *testing.T) {
	interceptor := NewAuthorizationInterceptor(nil)
	routes := []string{
		pbAgents.Agents_GetCanvasAgentChat_FullMethodName,
		pbAgents.Agents_SendAgentChatMessage_FullMethodName,
		pbAgents.Agents_ListAgentChatMessages_FullMethodName,
	}

	for _, route := range routes {
		rule, ok := interceptor.rules[route]
		require.True(t, ok)
		assert.Equal(t, []string{features.FeatureClaudeManagedAgents}, rule.RequiredExperimentalFeatures)
	}
}

func TestCheckRequiredExperimentalFeatures(t *testing.T) {
	rule := AuthorizationRule{
		RequiredExperimentalFeatures: []string{features.FeatureClaudeManagedAgents},
	}

	err := checkRequiredExperimentalFeatures(&models.Organization{}, rule)
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	err = checkRequiredExperimentalFeatures(&models.Organization{
		EnabledExperimentalFeatures: datatypes.JSONSlice[string]{features.FeatureClaudeManagedAgents},
	}, rule)
	require.NoError(t, err)
}

func marshalScopes(t *testing.T, scopes []string) string {
	t.Helper()

	payload, err := json.Marshal(scopes)
	require.NoError(t, err)
	return string(payload)
}
