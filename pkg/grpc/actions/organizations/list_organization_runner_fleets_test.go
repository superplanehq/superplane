package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

func Test__ListOrganizationRunnerFleets(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	t.Run("invalid organization id", func(t *testing.T) {
		_, err := ListOrganizationRunnerFleets(context.Background(), "not-a-uuid")
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("returns legacy machine types when the integrated backend is disabled", func(t *testing.T) {
		response, err := ListOrganizationRunnerFleets(t.Context(), r.Organization.ID.String())
		require.NoError(t, err)
		require.Len(t, response.Fleets, 4)

		assert.Equal(t, models.RunnerFleetE1LargeAMD64, response.Fleets[0].Id)
		assert.Equal(t, models.RunnerFleetE1LargeARM64, response.Fleets[1].Id)
		assert.Equal(t, models.RunnerFleetE1TinyAMD64, response.Fleets[2].Id)
		assert.Equal(t, models.RunnerFleetE1TinyARM64, response.Fleets[3].Id)
	})

	t.Run("returns enabled installation and organization fleets for the integrated backend", func(t *testing.T) {
		require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureNewRunners))

		installationFleet := newOrganizationListTestFleet("aws-large-amd64", models.RunnerFleetScopeInstallation, nil)
		require.NoError(t, installationFleet.Create(db))

		organizationFleet := newOrganizationListTestFleet(
			"private-large-amd64",
			models.RunnerFleetScopeOrganization,
			&r.Organization.ID,
		)
		require.NoError(t, organizationFleet.Create(db))

		otherOrganization, err := models.CreateOrganization("Other Fleet Organization", "")
		require.NoError(t, err)
		foreignFleet := newOrganizationListTestFleet(
			"foreign-large-amd64",
			models.RunnerFleetScopeOrganization,
			&otherOrganization.ID,
		)
		require.NoError(t, foreignFleet.Create(db))

		disabledFleet := newOrganizationListTestFleet(
			"disabled-large-amd64",
			models.RunnerFleetScopeOrganization,
			&r.Organization.ID,
		)
		disabledFleet.Enabled = false
		require.NoError(t, disabledFleet.Create(db))

		response, err := ListOrganizationRunnerFleets(t.Context(), r.Organization.ID.String())
		require.NoError(t, err)
		require.Len(t, response.Fleets, 2)
		assert.Equal(t, installationFleet.Slug, response.Fleets[0].Id)
		assert.Equal(t, models.RunnerFleetScopeInstallation, response.Fleets[0].Scope)
		assert.Equal(t, int32(8000), response.Fleets[0].Spec.CpuMillicores)
		assert.Equal(t, organizationFleet.Slug, response.Fleets[1].Id)
		assert.Equal(t, models.RunnerFleetScopeOrganization, response.Fleets[1].Scope)
	})
}

func newOrganizationListTestFleet(slug, scope string, scopeID *uuid.UUID) *models.RunnerFleet {
	return &models.RunnerFleet{
		ID:        uuid.New(),
		Slug:      slug,
		ScopeType: scope,
		ScopeID:   scopeID,
		Enabled:   true,
		Spec: datatypes.NewJSONType(models.RunnerFleetSpec{
			OperatingSystem: "linux",
			Architecture:    "amd64",
			CPUMillicores:   8000,
			MemoryMB:        32768,
			DiskGB:          30,
			Capabilities:    []string{"docker"},
		}),
		RunnerVersion: "v0.0.1",
	}
}
