package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestReviewRequestSetupRequiresFlagAndWorkspaceSource(t *testing.T) {
	r, factory := resetTestFactory(t)
	db := database.DB(t.Context())
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Review request")
	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", "workspace-github", nil)
	require.NoError(t, err)
	request := &pb.MaterializeFactoryAppTemplateRequest{
		FactoryId: factory.ID.String(), AppId: canvas.ID.String(), TemplateId: reviewRequestTemplateID,
		Integrations: []*pb.FactoryAppTemplateIntegration{{Type: "github", Id: integration.ID.String(), Name: integration.InstallationName}},
	}
	_, err = MaterializeFactoryAppTemplate(ctx, r.Organization.ID.String(), request)
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	require.Equal(t, codes.FailedPrecondition, code)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryReviewRequest))
	_, err = MaterializeFactoryAppTemplate(ctx, r.Organization.ID.String(), request)
	require.ErrorContains(t, err, "Add Merge confidence")
	source := resetTestTemplate(t, r, factory, "risk-score", models.ProviderGitHub, func(*yaml.Canvas) {})
	response, err := MaterializeFactoryAppTemplate(ctx, r.Organization.ID.String(), request)
	require.NoError(t, err)
	doc, err := yaml.CanvasFromYAML([]byte(response.CanvasYaml))
	require.NoError(t, err)
	require.Equal(t, source.ID.String(), findYAMLNode(t, doc, reviewRequestTriggerNodeID).Configuration["app"])
	for _, id := range []string{"list-candidates", "request-reviewers", reviewFindingsBodyNodeID, reviewFindingsCommentNode} {
		node := findYAMLNode(t, doc, id)
		require.NotNil(t, node.Integration, "%s must use the workspace GitHub connection", id)
		require.Equal(t, integration.ID.String(), node.Integration.ID)
	}
	for _, id := range []string{reviewFindingsBodyNodeID, reviewFindingsCommentNode} {
		require.Equal(t, false, findYAMLNode(t, doc, id).Configuration["enabled"], "new installations do not publish findings")
	}
	_, _, err = doc.Parse(r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	resetTestPublishCanvas(t, r, canvas, doc)
	require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, features.FeatureFactoryReviewRequest))
	_, err = MaterializeFactoryAppTemplate(ctx, r.Organization.ID.String(), request)
	require.NoError(t, err, "installed automations remain editable after disabling discovery")
	_, err = MaterializeFactoryAutomationDefaults(ctx, r.Organization.ID.String(), &pb.MaterializeFactoryAutomationDefaultsRequest{
		FactoryId: factory.ID.String(), AutomationId: canvas.ID.String(),
	})
	require.NoError(t, err)
	updated, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.ColumnKey)
	require.Equal(t, models.CanvasColumnKeyVerify, *updated.ColumnKey)
}

func TestReviewRequestRejectsSourceFromAnotherWorkspace(t *testing.T) {
	r, factory := resetTestFactory(t)
	db := database.DB(t.Context())
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryReviewRequest))
	other, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("other"), "", "")
	require.NoError(t, err)
	source := resetTestTemplate(t, r, other, "risk-score", models.ProviderGitHub, func(*yaml.Canvas) {})
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Review request")
	_, err = MaterializeFactoryAppTemplate(ctx, r.Organization.ID.String(), &pb.MaterializeFactoryAppTemplateRequest{
		FactoryId: factory.ID.String(), AppId: canvas.ID.String(), TemplateId: reviewRequestTemplateID,
		InstallParams: map[string]string{"mergeConfidenceAppId": source.ID.String()},
	})
	require.ErrorContains(t, err, "Merge confidence app")
}
