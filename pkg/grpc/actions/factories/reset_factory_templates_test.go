package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__OnboardingFactoryTemplates(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}

	t.Run("lists onboarding templates and resets one across organizations", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factoryModel.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)
		backlog := liveBacklogCanvas(t, factoryModel)
		setRefineTaskPrompt(t, backlog, "CUSTOM OLD BACKLOG PROMPT")

		implement := support.CreateFactoryCanvas(t, r, factoryModel.ID, "Implement")
		publishOnboardingTemplate(t, ctx, deps, factoryModel, implement, onboardingTemplateImplement)

		otherOrg := support.CreateOrganization(t, r, r.User)
		otherUser := support.CreateUser(t, r, otherOrg.ID)
		otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())
		otherFactory, err := models.CreateFactory(database.DB(t.Context()), otherOrg.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		otherImplement := createFactoryCanvasInOrg(t, otherOrg.ID, otherFactory.ID, otherUser.ID, "Implement")
		publishOnboardingTemplate(t, otherCtx, deps, otherFactory, otherImplement, onboardingTemplateImplement)

		listed, err := ListOnboardingFactoryTemplates(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{
			onboardingTemplateBacklog,
			onboardingTemplateImplement,
			onboardingTemplatePRClosure,
			onboardingTemplateIntake,
		}, templateIDs(listed))
		assert.Equal(t, 1, templateCount(listed, onboardingTemplateBacklog))
		assert.Equal(t, 2, templateCount(listed, onboardingTemplateImplement))
		assert.Equal(t, 1, templateCount(listed, onboardingTemplateIntake))
		assert.Equal(t, 0, templateCount(listed, onboardingTemplatePRClosure))

		result, err := ResetOnboardingFactoryTemplate(ctx, deps, onboardingTemplateBacklog)
		require.NoError(t, err)
		require.Empty(t, result.Failures)
		require.Equal(t, 1, result.Reset)
		assert.NotContains(t, liveRefineTaskPrompt(t, backlog), "CUSTOM OLD BACKLOG PROMPT")
		assert.Contains(t, liveRefineTaskPrompt(t, backlog), "Task:\n{{ root().data.workOrder }}")

		implementVersion := liveVersionID(t, implement)
		otherImplementVersion := liveVersionID(t, otherImplement)
		result, err = ResetOnboardingFactoryTemplate(ctx, deps, onboardingTemplateImplement)
		require.NoError(t, err)
		require.Empty(t, result.Failures)
		require.Equal(t, 2, result.Reset)
		assert.NotEqual(t, implementVersion, liveVersionID(t, implement))
		assert.NotEqual(t, otherImplementVersion, liveVersionID(t, otherImplement))
	})

	t.Run("rejects an unknown template", func(t *testing.T) {
		_, err := ResetOnboardingFactoryTemplate(ctx, deps, "risk-score")
		require.ErrorIs(t, err, ErrUnknownOnboardingFactoryTemplate)
	})
}

func createFactoryCanvasInOrg(t *testing.T, orgID, factoryID, userID uuid.UUID, name string) *models.Canvas {
	t.Helper()
	return support.CreateFactoryCanvas(t, &support.ResourceRegistry{
		Organization: &models.Organization{ID: orgID},
		User:         userID,
	}, factoryID, name)
}

func publishOnboardingTemplate(
	t *testing.T,
	ctx context.Context,
	deps IntakeDependencies,
	factory *models.Factory,
	canvas *models.Canvas,
	templateID string,
) {
	t.Helper()
	response, err := MaterializeFactoryAppTemplate(ctx, factory.OrganizationID.String(), &pb.MaterializeFactoryAppTemplateRequest{
		FactoryId:  factory.ID.String(),
		TemplateId: templateID,
		AppId:      canvas.ID.String(),
		InstallParams: map[string]string{
			"appRepository": "acme/app",
			"defaultBranch": "main",
		},
	})
	require.NoError(t, err)
	doc, err := yaml.CanvasFromYAML([]byte(response.GetCanvasYaml()))
	require.NoError(t, err)
	nodes, edges, err := doc.Parse(deps.Registry, factory.OrganizationID.String())
	require.NoError(t, err)
	require.NoError(t, canvases.PublishGeneratedCanvasNodesWithOwner(
		ctx,
		database.DB(t.Context()),
		canvas,
		nil,
		"Install onboarding template",
		nodes,
		edges,
		intakePublisherOptions(deps, factory.OrganizationID),
	))
	template, ok := lookupFactoryAppTemplate(templateID, "")
	require.True(t, ok)
	reloaded, err := models.FindCanvasInTransaction(database.DB(t.Context()), factory.OrganizationID, canvas.ID)
	require.NoError(t, err)
	require.NoError(t, reloaded.StampFactoryAppTemplate(
		database.DB(t.Context()),
		template.entrypointNodeID,
		templateID,
		factoryTemplateVersion,
	))
}

func templateIDs(items []OnboardingFactoryTemplate) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func templateCount(items []OnboardingFactoryTemplate, id string) int {
	for _, item := range items {
		if item.ID == id {
			return item.Count
		}
	}
	return 0
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
