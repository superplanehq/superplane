package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__UpdateFactoryOnboarding(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	t.Run("new factory is incomplete in create response", func(t *testing.T) {
		response, err := CreateFactory(context.Background(), r.Organization.ID.String(), &pb.CreateFactoryRequest{
			Name: support.RandomName("factory"),
		})
		require.NoError(t, err)
		require.NotNil(t, response.Factory.Onboarding)
		assert.Nil(t, response.Factory.Onboarding.CompletedAt)
		assert.Equal(t, pb.FactoryOnboarding_ISSUES_SOURCE_UNSPECIFIED, response.Factory.Onboarding.IssuesSource)
		assert.Equal(t, pb.FactoryOnboarding_AGENT_HARNESS_UNSPECIFIED, response.Factory.Onboarding.AgentHarness)
	})

	t.Run("partial update persists and returns merged config", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		vcsID := uuid.New().String()
		appRepo := "acme/api"
		issuesSource := pb.FactoryOnboarding_ISSUES_SOURCE_VCS

		response, err := UpdateFactoryOnboarding(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:               factory.ID.String(),
			VcsIntegrationId: &vcsID,
			AppRepository:    &appRepo,
			IssuesSource:     &issuesSource,
		})
		require.NoError(t, err)
		require.NotNil(t, response.Factory.Onboarding)
		assert.Nil(t, response.Factory.Onboarding.CompletedAt)
		assert.Equal(t, vcsID, response.Factory.Onboarding.VcsIntegrationId)
		assert.Equal(t, "acme/api", response.Factory.Onboarding.AppRepository)
		assert.Equal(t, pb.FactoryOnboarding_ISSUES_SOURCE_VCS, response.Factory.Onboarding.IssuesSource)

		backlogRepo := "acme/backlog"
		agentHarness := pb.FactoryOnboarding_AGENT_HARNESS_CLAUDE_CODE
		response, err = UpdateFactoryOnboarding(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:                factory.ID.String(),
			BacklogRepository: &backlogRepo,
			AgentHarness:      &agentHarness,
		})
		require.NoError(t, err)
		assert.Equal(t, vcsID, response.Factory.Onboarding.VcsIntegrationId)
		assert.Equal(t, "acme/api", response.Factory.Onboarding.AppRepository)
		assert.Equal(t, "acme/backlog", response.Factory.Onboarding.BacklogRepository)
		assert.Equal(t, pb.FactoryOnboarding_AGENT_HARNESS_CLAUDE_CODE, response.Factory.Onboarding.AgentHarness)
	})

	t.Run("complete succeeds when ready and is idempotent", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
		appID, lineID := createOnboardingResources(t, r, factory)

		req := readyOnboardingRequest(factory.ID.String(), vcsID, agentID, appID, lineID)
		complete := true
		req.Complete = &complete

		response, err := UpdateFactoryOnboarding(context.Background(), r.Organization.ID.String(), req)
		require.NoError(t, err)
		require.NotNil(t, response.Factory.Onboarding.CompletedAt)
		firstCompletedAt := response.Factory.Onboarding.CompletedAt.AsTime()

		response, err = UpdateFactoryOnboarding(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:       factory.ID.String(),
			Complete: &complete,
		})
		require.NoError(t, err)
		require.NotNil(t, response.Factory.Onboarding.CompletedAt)
		assert.Equal(t, firstCompletedAt.UnixMicro(), response.Factory.Onboarding.CompletedAt.AsTime().UnixMicro())
	})

	t.Run("complete rejects an integration from another organization", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		otherOrg := support.CreateOrganization(t, r, r.User)
		vcsID := createReadyOnboardingIntegration(t, otherOrg.ID, "github")
		agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
		appID, lineID := createOnboardingResources(t, r, factory)
		complete := true

		request := readyOnboardingRequest(factory.ID.String(), vcsID, agentID, appID, lineID)
		request.Complete = &complete
		_, err = UpdateFactoryOnboarding(context.Background(), r.Organization.ID.String(), request)

		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("complete with missing fields -> invalid argument", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		complete := true
		appRepo := "acme/api"
		_, err = UpdateFactoryOnboarding(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:            factory.ID.String(),
			AppRepository: &appRepo,
			Complete:      &complete,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("invalid uuid -> invalid argument", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		badID := "not-a-uuid"
		_, err = UpdateFactoryOnboarding(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:               factory.ID.String(),
			VcsIntegrationId: &badID,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("wrong organization -> not found", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		otherOrg := support.CreateOrganization(t, r, r.User)
		appRepo := "acme/api"
		_, err = UpdateFactoryOnboarding(context.Background(), otherOrg.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:            factory.ID.String(),
			AppRepository: &appRepo,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("not found -> error", func(t *testing.T) {
		appRepo := "acme/api"
		_, err := UpdateFactoryOnboarding(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:            "00000000-0000-0000-0000-000000000001",
			AppRepository: &appRepo,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})
}

func readyOnboardingRequest(factoryID, vcsID, agentID, appID, lineID string) *pb.UpdateFactoryOnboardingRequest {
	appRepo := "acme/api"
	backlogRepo := "acme/backlog"
	issuesSource := pb.FactoryOnboarding_ISSUES_SOURCE_SKIP
	agentHarness := pb.FactoryOnboarding_AGENT_HARNESS_CLAUDE_CODE

	return &pb.UpdateFactoryOnboardingRequest{
		Id:                 factoryID,
		VcsIntegrationId:   &vcsID,
		AgentIntegrationId: &agentID,
		AppRepository:      &appRepo,
		BacklogRepository:  &backlogRepo,
		IssuesSource:       &issuesSource,
		AgentHarness:       &agentHarness,
		ProvisionedAppId:   &appID,
		ProvisionedLineId:  &lineID,
	}
}

func createOnboardingResources(
	t *testing.T,
	r *support.ResourceRegistry,
	factory *models.Factory,
) (string, string) {
	t.Helper()
	app, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	require.NoError(t, database.DB(t.Context()).Model(app).Update("factory_id", factory.ID).Error)
	line, err := factory.CreateLine(database.DB(t.Context()), support.RandomName("line"), []models.FactoryLineStep{
		{
			Type:       models.FactoryLineStepTypeRunApp,
			AppID:      app.ID,
			Entrypoint: "work-order-dispatch",
		},
	})
	require.NoError(t, err)
	return app.ID.String(), line.ID.String()
}

func createReadyOnboardingIntegration(t *testing.T, organizationID uuid.UUID, appName string) string {
	t.Helper()
	integration, err := models.CreateIntegration(
		uuid.New(),
		organizationID,
		appName,
		support.RandomName(appName),
		map[string]any{},
	)
	require.NoError(t, err)
	require.NoError(t, database.DB(t.Context()).Model(integration).Update("state", models.IntegrationStateReady).Error)
	return integration.ID.String()
}
