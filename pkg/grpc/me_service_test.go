package grpc

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/metadata"
)

func TestPersonalCatalogCancellationPreservesTheExactTarget(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	fleet := models.DefaultInstallationRunnerFleets("test")[0]
	require.NoError(t, fleet.Create(db))
	task := &models.RunnerTask{
		ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID,
		Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
		PayloadCiphertext: []byte("login"), QueuedAt: time.Now(),
	}
	require.NoError(t, db.Create(task).Error)
	connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, time.Now())
	require.NoError(t, err)
	require.NoError(t, connection.BeginLogin(db, task.ID, time.Now()))
	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("x-user-id", r.User.String(), "x-organization-id", r.Organization.ID.String()))
	service := NewMeService(r.AuthService, r.Encryptor)
	response, err := service.CancelCodexConnection(ctx, &pb.CancelCodexConnectionRequest{CatalogRefreshTaskId: uuid.NewString()})
	require.NoError(t, err)
	assert.Equal(t, task.ID.String(), response.Connection.ActiveTaskId)
	assert.Equal(t, models.CodexConnectionStatusPending, response.Connection.Status)
	stored, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateQueued, stored.State, "a stale catalog target must never cancel sign-in")
	assert.Nil(t, stored.CancelRequestedAt)
}
