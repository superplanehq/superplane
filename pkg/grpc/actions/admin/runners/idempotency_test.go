package runners

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	runneractions "github.com/superplanehq/superplane/pkg/grpc/actions/runners"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/runners"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestConcurrentRunnerRetriesRemainFleetLocal(t *testing.T) {
	for _, scope := range []string{models.RunnerFleetScopeInstallation, models.RunnerFleetScopeOrganization} {
		t.Run(scope, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			fleets := []*models.RunnerFleet{createTestFleetWithSlug(t, "first"), createTestFleetWithSlug(t, "second")}
			service := newTestRunnerService()
			if scope == models.RunnerFleetScopeOrganization {
				other, err := models.CreateOrganization("Other runner organization", "")
				require.NoError(t, err)
				for i, id := range []uuid.UUID{r.Organization.ID, other.ID} {
					require.NoError(t, models.EnableExperimentalFeature(id, features.FeatureOrganizationRunners))
					fleets[i] = createTestFleet(t, scope, &id)
				}
			}
			requests := make([]runneractions.CreateRunnerOptions, 2)
			for i := range fleets {
				requests[i] = runneractions.CreateRunnerOptions{Ephemeral: scope == models.RunnerFleetScopeInstallation,
					IdempotencyKey: "arbitrary-shared-key", FleetManagerID: "manager", Provider: "test"}
			}
			responses := make([]*runneractions.CreatedRunner, 8)
			errs := make([]error, len(responses))
			start := make(chan struct{})
			var group sync.WaitGroup
			for i := range responses {
				group.Go(func() {
					<-start
					responses[i], errs[i] = service.manager.CreateRunner(t.Context(), fleets[i%2], requests[i%2])
				})
			}
			close(start)
			group.Wait()
			for i := range responses {
				require.NoError(t, errs[i])
				assert.Equal(t, responses[i%2].Runner.ID, responses[i].Runner.ID)
				assert.Equal(t, responses[i%2].RegistrationToken, responses[i].RegistrationToken)
			}
			assert.NotEqual(t, responses[0].Runner.ID, responses[1].Runner.ID)
			requests[0].Provider = "different"
			_, err := service.manager.CreateRunner(t.Context(), fleets[0], requests[0])
			require.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
			requests[0].Provider = "test"
			runner, err := models.FindRunner(db, responses[0].Runner.ID)
			require.NoError(t, err)
			historicalHash := runner.CreationRequestHash
			require.NoError(t, runner.Terminate(db, models.RunnerTerminationRequested))
			// Include historical terminated records that retained their keys.
			require.NoError(t, db.Model(runner).Updates(map[string]any{
				"creation_idempotency_key": requests[0].IdempotencyKey,
				"creation_request_hash":    historicalHash,
			}).Error)
			replacement, err := service.manager.CreateRunner(t.Context(), fleets[0], requests[0])
			require.NoError(t, err)
			assert.NotEqual(t, responses[0].Runner.ID, replacement.Runner.ID)
			unaffected, err := service.manager.CreateRunner(t.Context(), fleets[1], requests[1])
			require.NoError(t, err)
			assert.Equal(t, responses[1].Runner.ID, unaffected.Runner.ID)
			assert.Equal(t, responses[1].RegistrationToken, unaffected.RegistrationToken)
		})
	}
}

func TestInstallationRunnerRetrySurvivesFleetDisablement(t *testing.T) {
	support.Setup(t)
	fleet := createTestFleetWithSlug(t, "installation")
	service := newTestRunnerService()
	request := &pb.CreateRunnerRequest{FleetId: fleet.Slug, Ephemeral: true, IdempotencyKey: "existing-request"}
	first, err := service.CreateRunner(t.Context(), request)
	require.NoError(t, err)
	require.NoError(t, fleet.SetEnabled(database.DB(t.Context()), false))
	retry, err := service.CreateRunner(t.Context(), request)
	require.NoError(t, err)
	assert.Equal(t, first.Runner.Id, retry.Runner.Id)
	assert.Equal(t, first.RegistrationToken, retry.RegistrationToken)
	request.IdempotencyKey = "new-request"
	_, err = service.CreateRunner(t.Context(), request)
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
}
