package factories

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"

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

func TestResetBuiltInAutomationInAllOrganizationsRejectsUnknownType(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	_, err := ResetBuiltInAutomationInAllOrganizations(ctx, "custom", testIntakeDependencies(r))
	require.ErrorIs(t, err, ErrUnknownBuiltInAutomation)
}

func TestResetOneBuiltInAutomationAcrossOrganizations(t *testing.T) {
	r := support.Setup(t)
	other := support.CreateOrganization(t, r, r.User)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	deps := testIntakeDependencies(r)

	firstFactory, firstIntakeID := createGitHubIssuesIntake(t, r, r.Organization.ID, r.User)
	secondUser := support.CreateUser(t, r, other.ID)
	secondFactory, secondIntakeID := createGitHubIssuesIntake(t, r, other.ID, secondUser.ID)
	implementation := installFactoryTemplate(t, r, r.Organization.ID, firstFactory.ID, "line-implementation", models.ProviderGitHub, nil)
	stampFactoryCanvases(t, firstFactory)
	stampFactoryCanvases(t, secondFactory)
	stampCloneCommand(t, implementation)

	result, err := resetBuiltInAutomationForOrganizations(ctx, []models.Organization{*r.Organization, *other}, intakeTemplateID(models.FactoryIntakeSourceGitHubIssues), deps)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Reset)
	assert.Equal(t, 2, result.Organizations)
	assert.Empty(t, result.Failed)

	assert.NotContains(t, liveCanvasJSON(t, canvasByID(t, r.Organization.ID, firstIntakeID)), "git clone")
	assert.NotContains(t, liveCanvasJSON(t, canvasByID(t, other.ID, secondIntakeID)), "git clone")
	assert.Contains(t, liveCanvasJSON(t, implementation), "git clone")
	assertOtherFactoryCanvasesKeepClone(t, firstFactory, firstIntakeID)
	assertKeptFactoryCanvasesKeepClone(t, secondFactory, secondIntakeID)
}

func TestResetOneBuiltInAutomationContinuesAfterOrganizationFailure(t *testing.T) {
	r := support.Setup(t)
	other := support.CreateOrganization(t, r, r.User)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	invalid := installFactoryTemplate(t, r, r.Organization.ID, factoryForOrganization(t, r.Organization.ID).ID, "line-implementation", models.ProviderGitHub, nil)
	version := resetTestLiveVersion(t, invalid)
	findIntakeNode(version.Nodes, implementationAgentNodeID).Configuration["executionTimeoutSeconds"] = -1
	require.NoError(t, database.DB(t.Context()).Model(version).Update("nodes", datatypes.NewJSONSlice(version.Nodes)).Error)
	original := *invalid.LiveVersionID

	valid := installFactoryTemplate(t, r, other.ID, factoryForOrganization(t, other.ID).ID, "line-implementation", models.ProviderGitHub, nil)
	stampCloneCommand(t, valid)

	result, err := resetBuiltInAutomationForOrganizations(ctx, []models.Organization{*r.Organization, *other}, "line-implementation", testIntakeDependencies(r))
	require.NoError(t, err)
	assert.Equal(t, 1, result.Reset)
	assert.Equal(t, 1, result.Organizations)
	require.Len(t, result.Failed, 1)
	assert.Equal(t, r.Organization.ID.String(), result.Failed[0].OrganizationID)
	assert.Equal(t, r.Organization.Name, result.Failed[0].OrganizationName)
	assert.Contains(t, strings.ToLower(result.Failed[0].Error), "timeout")
	assert.Contains(t, result.Failed[0].Error, "This organization was not changed.")

	reloaded := canvasByID(t, r.Organization.ID, invalid.ID)
	assert.Equal(t, original, *reloaded.LiveVersionID)
	assert.NotContains(t, liveCanvasJSON(t, valid), "git clone")
}

func testIntakeDependencies(r *support.ResourceRegistry) IntakeDependencies {
	return IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}
}

func createGitHubIssuesIntake(t *testing.T, r *support.ResourceRegistry, organizationID, userID uuid.UUID) (*models.Factory, uuid.UUID) {
	t.Helper()
	ctx := authentication.SetUserIdInMetadata(context.Background(), userID.String())
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, organizationID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = CreateFactoryIntake(ctx, testIntakeDependencies(r), organizationID.String(), &pb.CreateFactoryIntakeRequest{
		FactoryId: factory.ID.String(),
		Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
	})
	require.NoError(t, err)
	intakes, err := factory.ListIntakes(db)
	require.NoError(t, err)
	require.Len(t, intakes, 1)
	return factory, intakes[0].CanvasID
}

func factoryForOrganization(t *testing.T, organizationID uuid.UUID) *models.Factory {
	t.Helper()
	factory, err := models.CreateFactory(database.DB(t.Context()), organizationID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	return factory
}

func canvasByID(t *testing.T, organizationID, canvasID uuid.UUID) *models.Canvas {
	t.Helper()
	canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), organizationID, canvasID)
	require.NoError(t, err)
	return canvas
}

func stampFactoryCanvases(t *testing.T, factory *models.Factory) {
	t.Helper()
	canvases, err := factory.ListCanvases(database.DB(t.Context()))
	require.NoError(t, err)
	for i := range canvases {
		stampCloneCommand(t, &canvases[i])
	}
}

func assertOtherFactoryCanvasesKeepClone(t *testing.T, factory *models.Factory, intakeCanvasID uuid.UUID) {
	t.Helper()
	assert.Greater(t, assertKeptFactoryCanvasesKeepClone(t, factory, intakeCanvasID), 0)
}

func assertKeptFactoryCanvasesKeepClone(t *testing.T, factory *models.Factory, intakeCanvasID uuid.UUID) int {
	t.Helper()
	canvases, err := factory.ListCanvases(database.DB(t.Context()))
	require.NoError(t, err)
	kept := 0
	for i := range canvases {
		if canvases[i].ID == intakeCanvasID {
			continue
		}
		kept++
		assert.Contains(t, liveCanvasJSON(t, &canvases[i]), "git clone")
	}
	return kept
}

func installFactoryTemplate(
	t *testing.T,
	r *support.ResourceRegistry,
	organizationID uuid.UUID,
	factoryID uuid.UUID,
	templateID string,
	provider string,
	configure func(*yaml.Canvas),
) *models.Canvas {
	t.Helper()
	canvas := newOrganizationFactoryCanvas(t, r, organizationID, factoryID, templateID)
	result, err := materializeFactoryTemplate(templateID, provider, factoryTemplateInput{
		appID:         canvas.ID.String(),
		appName:       canvas.Name,
		installParams: map[string]string{"appRepository": "acme/app", "defaultBranch": "main"},
		agent:         &factoryTemplateAgent{component: models.SuperPlaneRunnerComponent, credentialSource: "hosted"},
	})
	require.NoError(t, err)
	doc, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	if configure != nil {
		configure(doc)
	}
	resetTestPublishCanvas(t, r, canvas, doc)
	return canvas
}

func newOrganizationFactoryCanvas(t *testing.T, r *support.ResourceRegistry, organizationID, factoryID uuid.UUID, name string) *models.Canvas {
	t.Helper()
	now := time.Now()
	liveVersionID := uuid.New()
	canvas := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryID,
		Name:           support.RandomName(name),
		CreatedBy:      &r.User,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.DB(t.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}
		version := models.CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: canvas.ID,
			OwnerID:    &r.User,
			Nodes:      datatypes.NewJSONSlice([]models.Node{}),
			Edges:      datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt:  &now,
			UpdatedAt:  &now,
		}
		return tx.Create(&version).Error
	}))
	return canvas
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
