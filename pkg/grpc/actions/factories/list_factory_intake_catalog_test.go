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
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__FactoryIntakeCatalog(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	db := database.DB(t.Context())

	catalogByKey := func(t *testing.T) map[string]*pb.FactoryIntakeCatalogEntry {
		t.Helper()
		response, err := ListFactoryIntakeCatalog(ctx, orgID, &pb.ListFactoryIntakeCatalogRequest{})
		require.NoError(t, err)
		byKey := map[string]*pb.FactoryIntakeCatalogEntry{}
		for _, entry := range response.GetEntries() {
			byKey[entry.GetKey()] = entry
		}
		return byKey
	}

	t.Run("lists availability for the organization", func(t *testing.T) {
		catalog := catalogByKey(t)
		assert.True(t, catalog[models.FactoryIntakeSourceGitHubIssues].GetAvailable())
		assert.False(t, catalog[models.FactoryIntakeSourceDatadog].GetAvailable())
		assert.Equal(t, models.IntakeStatusBeta, catalog[models.FactoryIntakeSourceDatadog].GetStatus())
		assert.False(t, catalog["gitlab"].GetAvailable())

		entry, err := models.FindIntakeCatalogEntry(db, models.FactoryIntakeSourceDatadog)
		require.NoError(t, err)
		require.NoError(t, entry.AddOrganization(db, r.Organization.ID))
		t.Cleanup(func() { require.NoError(t, entry.RemoveOrganization(db, r.Organization.ID)) })

		assert.True(t, catalogByKey(t)[models.FactoryIntakeSourceDatadog].GetAvailable())
	})

	t.Run("rejects an intake that the organization cannot use", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		_, err = CreateFactoryIntake(ctx, IntakeDependencies{
			Registry:    r.Registry,
			Encryptor:   r.Encryptor,
			AuthService: r.AuthService,
		}, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_JIRA_ISSUES,
		})
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	})
}
