package factories

import (
	"context"
	"encoding/json"
	"strings"
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

func TestResetOrganizationBuiltInAutomations(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = CreateFactoryIntake(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryIntakeRequest{
		FactoryId: factory.ID.String(),
		Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
	})
	require.NoError(t, err)

	builtIn, err := factory.ListCanvases(db)
	require.NoError(t, err)
	require.NotEmpty(t, builtIn)
	for i := range builtIn {
		stampCloneCommand(t, &builtIn[i])
	}

	custom := support.CreateFactoryCanvas(t, r, factory.ID, "Custom notes")
	stampCloneCommand(t, custom)

	result, err := ResetOrganizationBuiltInAutomations(ctx, r.Organization.ID, deps)
	require.NoError(t, err)
	assert.Equal(t, len(builtIn), result.Reset)

	for i := range builtIn {
		assert.NotContains(t, liveCanvasJSON(t, &builtIn[i]), "git clone")
	}
	assert.Contains(t, liveCanvasJSON(t, custom), "git clone")
}

func stampCloneCommand(t *testing.T, canvas *models.Canvas) {
	t.Helper()
	db := database.DB(t.Context())
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)
	nodes := append(version.Nodes, models.Node{
		ID:   "saved-clone",
		Name: "Clone",
		Type: models.NodeTypeComponent,
		Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}},
		Configuration: map[string]any{
			"steps": []any{map[string]any{
				"name":    "Clone",
				"type":    "bash",
				"command": "git clone https://github.com/acme/app.git repo",
			}},
		},
	})
	require.NoError(t, db.Model(&models.CanvasVersion{}).
		Where("id = ?", version.ID).
		Update("nodes", datatypes.NewJSONSlice(nodes)).Error)
}

func liveCanvasJSON(t *testing.T, canvas *models.Canvas) string {
	t.Helper()
	reloaded, err := models.FindCanvasInTransaction(database.DB(t.Context()), canvas.OrganizationID, canvas.ID)
	require.NoError(t, err)
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), reloaded)
	require.NoError(t, err)
	encoded, err := json.Marshal(version.Nodes)
	require.NoError(t, err)
	return strings.ToLower(string(encoded))
}
