package runners

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/runners"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestCreateRunnerAtOrganizationCapacityRollsBackAndCanRetry(t *testing.T) {
	resource := support.Setup(t)
	t.Setenv(billingmodels.BillingEnabledEnv, "true")
	db := database.DB(t.Context())
	plan := &billingmodels.BillingPlan{
		Key: "runner_" + uuid.NewString()[:8], Name: "Runner capacity", Category: "paid",
		Visibility: "private", CreditFrequency: "none", CreditExpiration: "never", ParallelTaskLimit: new(1),
	}
	require.NoError(t, billingmodels.SaveBillingPlan(db, plan))
	_, err := billingmodels.SetAdminOrganizationPlan(db, resource.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	fleet := createTestFleet(t, models.RunnerFleetScopeInstallation, nil)
	otherFleet := createTestFleetWithSlug(t, "other-capacity-fleet")
	firstTask := createTestTask(t, resource.Organization.ID, fleet.ID, []byte("encrypted"))
	secondTask := createTestTask(t, resource.Organization.ID, otherFleet.ID, []byte("encrypted"))
	service := newTestRunnerService()
	firstRequest := &pb.CreateRunnerRequest{
		FleetId: fleet.Slug, TaskId: stringPointer(firstTask.ID.String()),
		IdempotencyKey: "first-slot", Ephemeral: true,
	}
	first, err := service.CreateRunner(t.Context(), firstRequest)
	require.NoError(t, err)
	secondRequest := &pb.CreateRunnerRequest{
		FleetId: otherFleet.Slug, TaskId: stringPointer(secondTask.ID.String()),
		IdempotencyKey: "denied-slot", Ephemeral: true,
	}
	_, err = service.CreateRunner(t.Context(), secondRequest)
	require.Equal(t, codes.ResourceExhausted, grpcerrors.Code(err))

	var runners, registrations int64
	require.NoError(t, db.Model(&models.Runner{}).Count(&runners).Error)
	require.NoError(t, db.Model(&models.RunnerRegistration{}).Count(&registrations).Error)
	assert.Equal(t, int64(1), runners)
	assert.Equal(t, int64(1), registrations)
	queued, err := models.FindRunnerTask(db, secondTask.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateQueued, queued.State)
	assert.Nil(t, queued.RunnerID)

	// Retries reuse an existing slot, and generic provisioning uses no slot.
	retried, err := service.CreateRunner(t.Context(), firstRequest)
	require.NoError(t, err)
	assert.Equal(t, first.Runner.Id, retried.Runner.Id)
	_, err = service.CreateRunner(t.Context(), &pb.CreateRunnerRequest{FleetId: fleet.Slug, Ephemeral: true})
	require.NoError(t, err)

	runner, err := models.FindRunner(db, uuid.MustParse(first.Runner.Id))
	require.NoError(t, err)
	require.NoError(t, runner.Terminate(db, models.RunnerTerminationProvisioningFailed))
	admitted, err := service.CreateRunner(t.Context(), secondRequest)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerStatePending, admitted.Runner.State)
}
