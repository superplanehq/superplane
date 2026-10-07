package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__ResetOrganizationBacklogDefaults(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}

	t.Run("replaces a customized Backlog prompt and leaves other canvases", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factoryModel.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)

		backlog := liveBacklogCanvas(t, factoryModel)
		setRefineTaskPrompt(t, backlog, "CUSTOM OLD BACKLOG PROMPT")
		require.Contains(t, liveRefineTaskPrompt(t, backlog), "CUSTOM OLD BACKLOG PROMPT")

		otherOrg := support.CreateOrganization(t, r, r.User)
		otherUser := support.CreateUser(t, r, otherOrg.ID)
		otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())
		otherFactory, err := models.CreateFactory(database.DB(t.Context()), otherOrg.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = CreateFactoryIntake(otherCtx, deps, otherOrg.ID.String(), &pb.CreateFactoryIntakeRequest{
			FactoryId: otherFactory.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)
		otherBacklog := liveBacklogCanvas(t, otherFactory)
		setRefineTaskPrompt(t, otherBacklog, "OTHER ORG CUSTOM PROMPT")
		otherVersionID := liveVersionID(t, otherBacklog)

		plainCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		plainVersionID := liveVersionID(t, plainCanvas)

		result, err := ResetOrganizationBacklogDefaults(ctx, deps, orgID)
		require.NoError(t, err)
		require.Empty(t, result.Failures)
		require.Equal(t, 1, result.Reset)

		prompt := liveRefineTaskPrompt(t, backlog)
		assert.NotContains(t, prompt, "CUSTOM OLD BACKLOG PROMPT")
		assert.Contains(t, prompt, "Task:\n{{ root().data.workOrder }}")
		assert.NotEqual(t, otherVersionID, liveVersionID(t, backlog))
		assert.Equal(t, otherVersionID, liveVersionID(t, otherBacklog))
		assert.Contains(t, liveRefineTaskPrompt(t, otherBacklog), "OTHER ORG CUSTOM PROMPT")
		assert.Equal(t, plainVersionID, liveVersionID(t, plainCanvas))
	})

	t.Run("returns zero when the organization has no Backlog automation", func(t *testing.T) {
		emptyOrg := support.CreateOrganization(t, r, r.User)
		result, err := ResetOrganizationBacklogDefaults(ctx, deps, emptyOrg.ID.String())
		require.NoError(t, err)
		assert.Equal(t, 0, result.Reset)
		assert.Empty(t, result.Failures)
	})
}

func setRefineTaskPrompt(t *testing.T, canvas *models.Canvas, prompt string) {
	t.Helper()
	db := database.DB(t.Context())
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)

	nodes := append([]models.Node(nil), version.Nodes...)
	found := false
	for i := range nodes {
		if nodes[i].ID != backlogRefinementNodeID {
			continue
		}
		steps, ok := nodes[i].Configuration["steps"].([]any)
		require.True(t, ok)
		for j := range steps {
			step, ok := steps[j].(map[string]any)
			if !ok || step["name"] != "Refine Task" {
				continue
			}
			step["prompt"] = prompt
			found = true
		}
		nodes[i].Configuration["steps"] = steps
	}
	require.True(t, found, "refine-task prompt step not found")
	require.NoError(t, db.Model(version).Update("nodes", datatypes.NewJSONSlice(nodes)).Error)
}

func liveRefineTaskPrompt(t *testing.T, canvas *models.Canvas) string {
	t.Helper()
	reloaded, err := models.FindCanvasInTransaction(database.DB(t.Context()), canvas.OrganizationID, canvas.ID)
	require.NoError(t, err)
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), reloaded)
	require.NoError(t, err)
	for _, node := range version.Nodes {
		if node.ID != backlogRefinementNodeID {
			continue
		}
		steps, ok := node.Configuration["steps"].([]any)
		require.True(t, ok)
		for _, raw := range steps {
			step, ok := raw.(map[string]any)
			if !ok || step["name"] != "Refine Task" {
				continue
			}
			text, ok := step["prompt"].(string)
			require.True(t, ok)
			return text
		}
	}
	require.Fail(t, "refine-task prompt not found")
	return ""
}

func liveVersionID(t *testing.T, canvas *models.Canvas) string {
	t.Helper()
	reloaded, err := models.FindCanvasInTransaction(database.DB(t.Context()), canvas.OrganizationID, canvas.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.LiveVersionID)
	return reloaded.LiveVersionID.String()
}
