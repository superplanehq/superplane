package factories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestHumanReviewRecoversRejectedReopenEvents(t *testing.T) {
	for _, scenario := range []struct {
		name        string
		state       string
		draft       bool
		readFailure bool
		newRequest  bool
		staleUpdate bool
	}{
		{name: "open", state: "open"},
		{name: "draft", state: "open", draft: true},
		{name: "stale reopen", state: "closed"},
		{name: "provider failure", state: "open", readFailure: true},
		{name: "new reopen during reconciliation", state: "closed", newRequest: true},
		{name: "lifecycle update during reconciliation", state: "closed", newRequest: true, staleUpdate: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newHumanReviewReconciliationFixture(t)
			order, err := f.factory.FindWorkOrder(f.db, f.pr.WorkOrderID)
			require.NoError(t, err)
			require.NoError(t, f.db.Model(order).Update("state", models.FactoryWorkOrderStateOpen).Error)
			_, err = applyGitHubPullRequestClosed(f.db, f.factory, f.pr, false, nil, nil, nil)
			require.NoError(t, err)
			integration, err := models.CreateIntegration(uuid.New(), f.factory.OrganizationID, "github", support.RandomName("github"), map[string]any{})
			require.NoError(t, err)
			client := &humanReviewRecoveryFake{humanReviewSetupFake: f.client, factoryGitHubAPI: &fakeFactoryGitHub{}}
			stubFactoryGitHub(t, client)
			original := factoryHumanReviewScheduler
			// An occupied slot fills the queue without starting provider work.
			factoryHumanReviewScheduler = newHumanReviewScheduler(0, 1, reconcileFactoryHumanReview)
			require.True(t, factoryHumanReviewScheduler.schedule(t.Context(), IntakeDependencies{}, models.FactoryPullRequest{ID: uuid.New()}))
			t.Cleanup(func() { factoryHumanReviewScheduler = original })

			RefreshFactoryPullRequestMergeabilityFromGitHubEvent(t.Context(), IntakeDependencies{}, &models.Webhook{AppInstallationID: &integration.ID}, "pull_request",
				[]byte(`{"action":"reopened","repository":{"full_name":"acme/app"},"pull_request":{"number":42,"head":{"sha":"current-head"}}}`))

			pending, err := models.ListFactoryHumanReviewPullRequestsToReconcile(f.db, uuid.Nil, 100)
			require.NoError(t, err)
			require.Len(t, pending, 1, "polling must recover a closed PR whose reopen event was rejected")
			require.Equal(t, f.pr.ID, pending[0].ID)
			require.Equal(t, models.FactoryPullRequestStateClosed, pending[0].State, "queue admission must not reopen a task without live confirmation")
			client.current.State = github.Ptr(scenario.state)
			client.current.Draft = github.Ptr(scenario.draft)
			if scenario.newRequest {
				client.onRead = func() error {
					stale := *f.pr
					if err := f.pr.RequestHumanReviewReconciliation(f.db); err != nil {
						return err
					}
					if scenario.staleUpdate {
						return stale.Update(f.db, models.FactoryPullRequestPatch{Title: github.Ptr("Updated title")})
					}
					return nil
				}
			}
			if scenario.readFailure {
				client.readFailure = errors.New("GitHub unavailable")
				require.ErrorIs(t, reconcileFactoryHumanReview(t.Context(), IntakeDependencies{}, pending[0]), client.readFailure)
				pending, err = models.ListFactoryHumanReviewPullRequestsToReconcile(f.db, uuid.Nil, 100)
				require.NoError(t, err)
				require.Len(t, pending, 1, "a provider failure must leave the request pending")
				client.readFailure = nil
			}
			// A fresh scheduler models recovery after the original process exits.
			completed := make(chan error, 1)
			factoryHumanReviewScheduler = newHumanReviewScheduler(1, 1, func(ctx context.Context, deps IntakeDependencies, pr models.FactoryPullRequest) error {
				err := reconcileFactoryHumanReview(ctx, deps, pr)
				completed <- err
				return err
			})
			require.True(t, ScheduleFactoryHumanReview(t.Context(), IntakeDependencies{}, pending[0]))
			select {
			case err := <-completed:
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("reconciliation did not finish")
			}
			stored, err := f.factory.FindPullRequest(f.db, models.FactoryPullRequestLookup{ID: f.pr.ID})
			require.NoError(t, err)
			order, err = f.factory.FindWorkOrder(f.db, order.ID)
			require.NoError(t, err)
			if scenario.state == "closed" {
				require.Equal(t, models.FactoryPullRequestStateClosed, stored.State)
				require.False(t, order.IsOpen())
				pending, err = models.ListFactoryHumanReviewPullRequestsToReconcile(f.db, uuid.Nil, 100)
				require.NoError(t, err)
				if scenario.newRequest {
					require.Len(t, pending, 1, "completing an older request must preserve a newer reopen event")
					client.onRead = nil
					require.NoError(t, reconcileFactoryHumanReview(t.Context(), IntakeDependencies{}, pending[0]))
					pending, err = models.ListFactoryHumanReviewPullRequestsToReconcile(f.db, uuid.Nil, 100)
					require.NoError(t, err)
				}
				require.Empty(t, pending, "a stale reopen must retire after live confirmation")
				return
			}
			state := models.FactoryPullRequestStateOpen
			if scenario.draft {
				state = models.FactoryPullRequestStateDraft
			}
			require.Equal(t, state, stored.State)
			require.Nil(t, stored.ClosedAt)
			require.Nil(t, stored.HumanReviewRequestID)
			require.True(t, order.IsOpen())
		})
	}
}

type humanReviewRecoveryFake struct {
	*humanReviewSetupFake
	factoryGitHubAPI
	readFailure error
	onRead      func() error
}

func (f *humanReviewRecoveryFake) GetPullRequest(ctx context.Context, repository string, number int) (*github.PullRequest, *github.Response, error) {
	if f.readFailure != nil {
		return nil, nil, f.readFailure
	}
	if f.onRead != nil {
		if err := f.onRead(); err != nil {
			return nil, nil, err
		}
	}
	return f.humanReviewSetupFake.GetPullRequest(ctx, repository, number)
}

func (f *humanReviewRecoveryFake) ListCheckRunsForRef(ctx context.Context, repository, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
	return f.humanReviewSetupFake.ListCheckRunsForRef(ctx, repository, ref, opts)
}
