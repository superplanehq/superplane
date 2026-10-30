package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestPersonalCodexConnectionDoesNotActivateWorkspace(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	ctx := onboardingContext(r)
	before, err := DescribeFactoryCodexConnection(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{})
	require.NoError(t, err)
	assert.True(t, before.Connection.Eligible)
	assert.Equal(t, "disconnected", before.Connection.Status)
	fleets := models.DefaultInstallationRunnerFleets(models.DefaultRunnerVersion)
	require.NoError(t, fleets[0].Create(db))
	t.Setenv("JWT_SECRET", "personal-codex-test")
	t.Setenv("RUNNER_API_BASE_URL", "https://superplane.example")
	started, err := StartFactoryCodexConnection(ctx, r.Encryptor, r.Organization.ID.String(), &pb.StartFactoryCodexConnectionRequest{})
	require.NoError(t, err)
	assert.Equal(t, "pending", started.Connection.Status)
	connection, err := models.FindCodexConnection(db, r.Organization.ID, r.User)
	require.NoError(t, err)
	lease, err := models.FindCodexLoginLease(db, connection.ID)
	require.NoError(t, err)
	require.NotNil(t, lease)
	assert.Nil(t, lease.FactoryID)
	view, err := DescribeFactoryCodexConnection(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{FactoryId: factory.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, started.Connection.Id, view.Connection.Id)
	saved, err := models.FindFactory(db, r.Organization.ID, factory.ID)
	require.NoError(t, err)
	assert.False(t, saved.OnboardingConfigValue().CodexSubscription)
	assert.Empty(t, saved.OnboardingConfigValue().CodexDefaultOwnerID)
}

func TestCancelFactoryCodexConnectionReportsCancellationUntilRunnerDeletion(t *testing.T) {
	for _, test := range []struct {
		status          string
		cancelingStatus string
		finalStatus     string
	}{
		{models.CodexConnectionStatusPending, "canceling", models.CodexConnectionStatusDisconnected},
		{models.CodexConnectionStatusFinishing, models.CodexConnectionStatusFinishing, models.CodexConnectionStatusReady},
		{models.CodexConnectionStatusReady, models.CodexConnectionStatusReady, models.CodexConnectionStatusReady},
	} {
		t.Run(test.status, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			fleet := models.DefaultInstallationRunnerFleets("test")[0]
			require.NoError(t, fleet.Create(db))
			now := time.Now()
			runner := &models.Runner{
				ID: uuid.New(), FleetID: fleet.ID, State: models.RunnerStateIdle,
				RunnerVersion: "test", Ephemeral: true, CreatedAt: now, UpdatedAt: now,
			}
			require.NoError(t, db.Create(runner).Error)
			task := &models.RunnerTask{
				ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID,
				Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
				PayloadCiphertext: []byte("login"), QueuedAt: now, CreatedAt: now, UpdatedAt: now,
			}
			require.NoError(t, db.Create(task).Error)
			connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, now)
			require.NoError(t, err)
			if test.status == models.CodexConnectionStatusReady {
				require.NoError(t, db.Model(connection).Updates(map[string]any{
					"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("saved-bundle"),
				}).Error)
				require.NoError(t, connection.BeginCatalogRefresh(db, task.ID, now))
			} else {
				require.NoError(t, connection.BeginLogin(db, task.ID, now))
				require.NoError(t, connection.RecordDeviceCode(db, task.ID, connection.ReservationGeneration, "https://auth.openai.com/codex/device", "TEST-CODE", now))
			}
			require.NoError(t, task.Reserve(db, runner.ID))
			require.NoError(t, task.Start(db, runner, "test", now))
			if test.status == models.CodexConnectionStatusFinishing {
				require.NoError(t, connection.StoreBundle(db, task.ID, connection.ReservationGeneration, 1, []byte("saved-bundle"), "member@example.com", true, now))
			}

			ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
			before, err := DescribeFactoryCodexConnection(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{FactoryId: factory.ID.String()})
			require.NoError(t, err)
			assert.Equal(t, test.status, before.Connection.Status)
			if test.status == models.CodexConnectionStatusPending {
				assert.Equal(t, "TEST-CODE", before.Connection.VerificationCode)
			}

			canceled, err := CancelFactoryCodexConnection(ctx, crypto.NewNoOpEncryptor(), r.Organization.ID.String(), &pb.CancelFactoryCodexConnectionRequest{FactoryId: factory.ID.String()})
			require.NoError(t, err)
			assert.Equal(t, test.cancelingStatus, canceled.Connection.Status)
			assert.Empty(t, canceled.Connection.VerificationUrl)
			assert.Empty(t, canceled.Connection.VerificationCode)
			assert.True(t, canceled.Connection.ReservationHeld)
			assert.Equal(t, task.ID.String(), canceled.Connection.ActiveTaskId)

			refreshed, err := DescribeFactoryCodexConnection(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{FactoryId: factory.ID.String()})
			require.NoError(t, err)
			assert.Equal(t, test.cancelingStatus, refreshed.Connection.Status)
			assert.Empty(t, refreshed.Connection.VerificationCode)
			stored, err := models.FindCodexConnection(db, r.Organization.ID, r.User)
			require.NoError(t, err)
			assert.Equal(t, test.status, stored.Status)

			completed, err := task.Complete(db, runner, "canceled", datatypes.JSON([]byte(`{}`)), 1, "", true, time.Now())
			require.NoError(t, err)
			require.True(t, completed)
			stopped, err := DescribeFactoryCodexConnection(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{FactoryId: factory.ID.String()})
			require.NoError(t, err)
			assert.Equal(t, test.cancelingStatus, stopped.Connection.Status)
			assert.True(t, stopped.Connection.ReservationHeld)

			require.NoError(t, runner.MarkProviderDeleted(db, time.Now()))
			finished, err := DescribeFactoryCodexConnection(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{FactoryId: factory.ID.String()})
			require.NoError(t, err)
			assert.Equal(t, test.finalStatus, finished.Connection.Status)
			assert.False(t, finished.Connection.ReservationHeld)
			assert.Empty(t, finished.Connection.ActiveTaskId)
		})
	}
}
