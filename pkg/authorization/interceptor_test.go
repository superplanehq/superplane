package authorization

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAgents "github.com/superplanehq/superplane/pkg/protos/agents"
	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/metadata"
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
		resourceIDs := canvasResourceResolver(&pbAgents.GenerateAgentChatTokenRequest{CanvasId: "canvas-123"})
		require.Equal(t, []string{"canvas-123"}, resourceIDs)
	})

	t.Run("returns nil when request does not expose a canvas id", func(t *testing.T) {
		resourceIDs := canvasResourceResolver(&pbCanvases.ListCanvasesRequest{})
		assert.Nil(t, resourceIDs)
	})
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
			req:         &pbAgents.GenerateAgentChatTokenRequest{CanvasId: "canvas-123"},
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
			req:         &pbAgents.GenerateAgentChatTokenRequest{CanvasId: "canvas-123"},
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
			req:         &pbAgents.GenerateAgentChatTokenRequest{CanvasId: "canvas-123"},
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

func marshalScopes(t *testing.T, scopes []string) string {
	t.Helper()

	payload, err := json.Marshal(scopes)
	require.NoError(t, err)
	return string(payload)
}
