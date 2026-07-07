package shared

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	canvasactions "github.com/superplanehq/superplane/pkg/agents/agent_tools/actions"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"

	// Register components used by patch_staging.
	_ "github.com/superplanehq/superplane/pkg/components/noop"
)

var agentPatchStagingRegistry struct {
	once sync.Once
	reg  *canvasactions.Registry
}

func agentPatchStagingActionRegistry(t *testing.T) *canvasactions.Registry {
	t.Helper()

	agentPatchStagingRegistry.once.Do(func() {
		encryptor := crypto.NewNoOpEncryptor()
		componentRegistry, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
		require.NoError(t, err)

		authService, err := authorization.NewAuthService()
		require.NoError(t, err)

		agentPatchStagingRegistry.reg = canvasactions.NewDefaultRegistry(canvasactions.Dependencies{
			Encryptor:      encryptor,
			Registry:       componentRegistry,
			AuthService:    authService,
			WebhookBaseURL: "https://hooks.example.test",
		})
	})

	return agentPatchStagingRegistry.reg
}

// StageNoopNodeViaAgentPatch stages a noop node the same way the managed agent does
// (patch_staging -> PutCanvasStaging -> staging_updated websocket).
func StageNoopNodeViaAgentPatch(
	t *testing.T,
	organizationID uuid.UUID,
	userID uuid.UUID,
	canvasID uuid.UUID,
	nodeName string,
	pos models.Position,
) {
	t.Helper()

	nodeID := "agent-staged-" + uuid.NewString()
	ctx := authentication.SetUserIdInMetadata(context.Background(), userID.String())

	_, err := agentPatchStagingActionRegistry(t).Execute(ctx, agents.AgentSessionContext{
		SessionID:      "e2e-agent-staging",
		OrganizationID: organizationID.String(),
		UserID:         userID.String(),
		CanvasID:       canvasID.String(),
	}, canvasactions.Input{
		Action: "patch_staging",
		PatchOperations: []canvasactions.PatchOperation{
			{
				Op: "add_node",
				Node: &canvasactions.PatchNode{
					ID:        nodeID,
					Name:      nodeName,
					Component: "noop",
					Position:  &canvasactions.PatchPosition{X: pos.X, Y: pos.Y},
				},
			},
		},
	})
	require.NoError(t, err)
}

// StageNoopNodeViaAgentPatch is a CanvasSteps wrapper around StageNoopNodeViaAgentPatch.
func (s *CanvasSteps) StageNoopNodeViaAgentPatch(nodeName string, pos models.Position) {
	StageNoopNodeViaAgentPatch(s.t, s.session.OrgID, s.sessionUserID(), s.WorkflowID, nodeName, pos)
}
