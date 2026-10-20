package factories

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	canvasActions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const expectedImplementationTestingGuidance = "Follow the repository's testing instructions and use its existing test tools and conventions. For bug fixes, add a regression test that reproduces the failure when practical. For behavior changes, add or update tests that verify observable behavior. Avoid tests that only mirror the implementation or assert incidental presentation details. Run the relevant tests, lint, and build checks. Fix failures caused by your changes. Report any checks you could not run and explain why."

func TestUpgradeOrganizationImplementationTestingGuidance(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context() // Startup has no user identity; publishing must use a system owner.
	db := database.DB(ctx)
	deps := IntakeDependencies{
		Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	setupCtx := authentication.SetUserIdInMetadata(ctx, r.User.String())
	_, err = CreateFactoryIntake(setupCtx, deps, r.Organization.ID.String(), &pb.CreateFactoryIntakeRequest{
		FactoryId: factory.ID.String(), Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
	})
	require.NoError(t, err)
	implementation := support.CreateFactoryCanvas(t, r, factory.ID, "Implement")
	materialized, err := materializeFactoryTemplate("line-implementation", "", factoryTemplateInput{
		appID: implementation.ID.String(), appName: implementation.Name,
	})
	require.NoError(t, err)
	doc, err := yaml.CanvasFromYAML([]byte(materialized.canvasYAML))
	require.NoError(t, err)
	nodes, edges, err := doc.Parse(r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	require.NoError(t, canvasActions.PublishGeneratedCanvasNodesWithOwner(ctx, db, implementation, nil,
		"Install Implement", nodes, edges, changesets.CanvasPublisherOptions{
			Registry: r.Registry, OrgID: r.Organization.ID, Encryptor: r.Encryptor,
			AuthService: r.AuthService, WebhookBaseURL: deps.WebhookBaseURL,
		}))
	implementation, err = models.FindCanvasInTransaction(db, r.Organization.ID, implementation.ID)
	require.NoError(t, err)
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, implementation)
	require.NoError(t, err)
	for i := range version.Nodes {
		steps, _ := version.Nodes[i].Configuration["steps"].([]any)
		for _, step := range steps {
			item, _ := step.(map[string]any)
			if item["name"] == "Implementation" {
				item["prompt"] = "Customer-authored implementation instructions"
			}
		}
	}
	require.NoError(t, db.Model(version).Update("nodes", datatypes.NewJSONSlice(version.Nodes)).Error)

	canvases, err := factory.ListCanvases(db)
	require.NoError(t, err)
	var implement *models.Canvas
	before := map[string][]models.CanvasVersion{}
	for i := range canvases {
		canvas := &canvases[i]
		stampCloneCommand(t, canvas)
		version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
		require.NoError(t, err)
		if models.FactoryAppTemplateID(version.Nodes) == "line-implementation" {
			implement = canvas
		}
		before[canvas.ID.String()], err = models.ListCanvasVersionsInTransaction(db, canvas.ID)
		require.NoError(t, err)
	}
	require.NotNil(t, implement)
	custom := support.CreateFactoryCanvas(t, r, factory.ID, "Implement")
	stampCloneCommand(t, custom)
	customBefore, err := models.ListCanvasVersionsInTransaction(db, custom.ID)
	require.NoError(t, err)

	result, err := UpgradeOrganizationImplementationTestingGuidance(ctx, r.Organization.ID, deps)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Reset)

	for i := range canvases {
		canvas := &canvases[i]
		versions, err := models.ListCanvasVersionsInTransaction(db, canvas.ID)
		require.NoError(t, err)
		if canvas.ID != implement.ID {
			assert.Equal(t, before[canvas.ID.String()], versions, "other template must stay unchanged")
			continue
		}
		require.Len(t, versions, len(before[canvas.ID.String()])+1)
		assert.Nil(t, versions[0].OwnerID)
		assert.Contains(t, liveCanvasJSON(t, canvas), strings.ToLower(expectedImplementationTestingGuidance))
		assert.NotContains(t, liveCanvasJSON(t, canvas), "customer-authored")
		assert.NotContains(t, liveCanvasJSON(t, canvas), "saved-clone")
	}
	customAfter, err := models.ListCanvasVersionsInTransaction(db, custom.ID)
	require.NoError(t, err)
	assert.Equal(t, customBefore, customAfter)

	// Further customer edits survive subsequent starts once the paragraph exists.
	implement, err = models.FindCanvasInTransaction(db, r.Organization.ID, implement.ID)
	require.NoError(t, err)
	stampCloneCommand(t, implement)
	upgraded, err := models.ListCanvasVersionsInTransaction(db, implement.ID)
	require.NoError(t, err)
	result, err = UpgradeOrganizationImplementationTestingGuidance(ctx, r.Organization.ID, deps)
	require.NoError(t, err)
	assert.Zero(t, result.Reset)
	afterSecondRun, err := models.ListCanvasVersionsInTransaction(db, implement.ID)
	require.NoError(t, err)
	assert.Equal(t, upgraded, afterSecondRun)
}

const overlappingUpgradeEditMarker = "customer-edit-survives-overlap"

func TestUpgradeOrganizationImplementationTestingGuidance_OverlappingStartKeepsLaterEdit(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)
	deps := IntakeDependencies{
		Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	implementation := support.CreateFactoryCanvas(t, r, factory.ID, "Implement")
	publishImplementationTemplate(ctx, t, r, deps, implementation)
	implementation, err = models.FindCanvasInTransaction(db, r.Organization.ID, implementation.ID)
	require.NoError(t, err)
	stripImplementationPrompt(t, db, implementation, "Customer-authored implementation instructions")

	before, err := models.ListCanvasVersionsInTransaction(db, implementation.ID)
	require.NoError(t, err)

	holder := database.Conn().Begin()
	require.NoError(t, holder.Error)
	finished := make(chan struct{})
	done := make(chan error, 1)
	t.Cleanup(func() {
		_ = holder.Rollback()
		select {
		case <-finished:
		case <-time.After(20 * time.Second):
			t.Errorf("upgrade did not finish after releasing the canvas lock")
		}
	})
	locked, err := models.LockCanvasForUpdate(holder, r.Organization.ID, implementation.ID)
	require.NoError(t, err)
	var holderPID int
	require.NoError(t, holder.Raw("SELECT pg_backend_pid()").Scan(&holderPID).Error)

	go func() {
		defer close(finished)
		_, upgradeErr := UpgradeOrganizationImplementationTestingGuidance(ctx, r.Organization.ID, deps)
		done <- upgradeErr
	}()

	require.Eventually(t, func() bool {
		select {
		case upgradeErr := <-done:
			t.Fatalf("upgrade finished before waiting for the canvas lock: %v", upgradeErr)
		default:
		}
		var waiting int
		err := db.Raw(
			"SELECT COUNT(*) FROM pg_stat_activity WHERE ? = ANY(pg_blocking_pids(pid))",
			holderPID,
		).Scan(&waiting).Error
		return err == nil && waiting > 0
	}, 15*time.Second, 20*time.Millisecond)

	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(holder, locked)
	require.NoError(t, err)
	require.True(t, setImplementationPrompt(version, implementationTestingGuidance+" "+overlappingUpgradeEditMarker))
	require.NoError(t, holder.Model(version).Update("nodes", datatypes.NewJSONSlice(version.Nodes)).Error)
	require.NoError(t, holder.Commit().Error)

	require.NoError(t, <-done)
	after, err := models.ListCanvasVersionsInTransaction(db, implementation.ID)
	require.NoError(t, err)
	if len(after) != len(before) {
		t.Fatalf("overlapping upgrade published a replacement; versions %d -> %d", len(before), len(after))
	}
	assert.Equal(t, before[0].ID, after[0].ID)
	live := liveCanvasJSON(t, implementation)
	assert.Contains(t, live, strings.ToLower(implementationTestingGuidance))
	assert.Contains(t, live, overlappingUpgradeEditMarker)
}

func publishImplementationTemplate(
	ctx context.Context,
	t *testing.T,
	r *support.ResourceRegistry,
	deps IntakeDependencies,
	implementation *models.Canvas,
) {
	t.Helper()
	materialized, err := materializeFactoryTemplate("line-implementation", "", factoryTemplateInput{
		appID: implementation.ID.String(), appName: implementation.Name,
	})
	require.NoError(t, err)
	doc, err := yaml.CanvasFromYAML([]byte(materialized.canvasYAML))
	require.NoError(t, err)
	nodes, edges, err := doc.Parse(r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	require.NoError(t, canvasActions.PublishGeneratedCanvasNodesWithOwner(ctx, database.DB(ctx), implementation, nil,
		"Install Implement", nodes, edges, changesets.CanvasPublisherOptions{
			Registry: r.Registry, OrgID: r.Organization.ID, Encryptor: r.Encryptor,
			AuthService: r.AuthService, WebhookBaseURL: deps.WebhookBaseURL,
		}))
}

func stripImplementationPrompt(t *testing.T, db *gorm.DB, canvas *models.Canvas, prompt string) {
	t.Helper()
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)
	require.True(t, setImplementationPrompt(version, prompt))
	require.NoError(t, db.Model(version).Update("nodes", datatypes.NewJSONSlice(version.Nodes)).Error)
}

func setImplementationPrompt(version *models.CanvasVersion, prompt string) bool {
	for i := range version.Nodes {
		steps, _ := version.Nodes[i].Configuration["steps"].([]any)
		for _, step := range steps {
			item, _ := step.(map[string]any)
			if item["name"] != "Implementation" {
				continue
			}
			item["prompt"] = prompt
			return true
		}
	}
	return false
}
