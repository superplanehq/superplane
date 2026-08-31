package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__MaterializeFactoryAppDefaults(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		GitProvider:    r.GitProvider,
		WebhookBaseURL: "http://localhost:8000",
	}

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	t.Run("the Backlog automation resets to its generated graph", func(t *testing.T) {
		factoryModel := newFactory(t)
		_, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factoryModel.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)

		backlog := liveBacklogCanvas(t, factoryModel)

		response, err := MaterializeFactoryAppDefaults(ctx, orgID, &pb.MaterializeFactoryAppDefaultsRequest{
			FactoryId: factoryModel.ID.String(),
			AppId:     backlog.ID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, "backlog", response.GetTemplateId())

		defaults, err := yaml.CanvasFromYAML([]byte(response.GetCanvasYaml()))
		require.NoError(t, err)
		assert.Equal(t, backlog.ID.String(), defaults.Metadata.ID)
		assert.Equal(t, backlog.Name, defaults.Metadata.Name)
	})

	t.Run("a newly created app materializes its install template", func(t *testing.T) {
		factoryModel := newFactory(t)
		canvas := support.CreateFactoryCanvas(t, r, factoryModel.ID, support.RandomName("Plan"))

		response, err := MaterializeFactoryAppTemplate(ctx, orgID, &pb.MaterializeFactoryAppTemplateRequest{
			FactoryId:  factoryModel.ID.String(),
			TemplateId: "line-planning",
			AppId:      canvas.ID.String(),
			InstallParams: map[string]string{
				"appRepository": "acme/app",
				"defaultBranch": "main",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "line-planning", response.GetTemplateId())
		assert.NotEmpty(t, response.GetCanvasYaml())
		assert.NotEmpty(t, response.GetConsoleYaml())
	})

	t.Run("an app from another factory reports not found", func(t *testing.T) {
		factoryModel := newFactory(t)
		other := newFactory(t)
		canvas := support.CreateFactoryCanvas(t, r, other.ID, support.RandomName("Plan"))

		_, err := MaterializeFactoryAppDefaults(ctx, orgID, &pb.MaterializeFactoryAppDefaultsRequest{
			FactoryId: factoryModel.ID.String(),
			AppId:     canvas.ID.String(),
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("an intake resets to its generated graph", func(t *testing.T) {
		factoryModel := newFactory(t)
		intake, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factoryModel.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)

		response, err := MaterializeFactoryAppDefaults(ctx, orgID, &pb.MaterializeFactoryAppDefaultsRequest{
			FactoryId: factoryModel.ID.String(),
			AppId:     intake.GetIntake().GetCanvasId(),
		})
		require.NoError(t, err)
		assert.Equal(t, "intake:"+models.FactoryIntakeSourceGitHubIssues, response.GetTemplateId())

		defaults, err := yaml.CanvasFromYAML([]byte(response.GetCanvasYaml()))
		require.NoError(t, err)
		assert.Equal(t, intake.GetIntake().GetCanvasId(), defaults.Metadata.ID)
	})
}
