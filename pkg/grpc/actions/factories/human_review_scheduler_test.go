package factories

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestHumanReviewSchedulerBoundsAndCoalescesRequests(t *testing.T) {
	started := make(chan models.FactoryPullRequest, 20)
	release := make(chan struct{})
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	scheduler := newHumanReviewScheduler(2, 4, func(_ context.Context, _ IntakeDependencies, pr models.FactoryPullRequest) error {
		started <- pr
		<-release
		return nil
	})
	requests := make([]models.FactoryPullRequest, 5)
	for i := range requests {
		requests[i] = models.FactoryPullRequest{ID: uuid.New(), Repository: "original"}
	}
	next := func() models.FactoryPullRequest {
		t.Helper()
		select {
		case pr := <-started:
			return pr
		case <-time.After(5 * time.Second):
			t.Fatal("reconciliation did not start")
			return models.FactoryPullRequest{}
		}
	}
	for _, pr := range requests[:2] {
		require.True(t, scheduler.schedule(t.Context(), IntakeDependencies{}, pr))
	}
	first, second := next(), next()
	require.ElementsMatch(t, []uuid.UUID{requests[0].ID, requests[1].ID}, []uuid.UUID{first.ID, second.ID})
	for _, pr := range requests[2:4] {
		require.True(t, scheduler.schedule(t.Context(), IntakeDependencies{}, pr))
	}
	for range 100 {
		requests[0].Repository = "latest running request"
		requests[2].Repository = "latest queued request"
		require.True(t, scheduler.schedule(t.Context(), IntakeDependencies{}, requests[0]))
		require.True(t, scheduler.schedule(t.Context(), IntakeDependencies{}, requests[2]))
	}
	require.False(t, scheduler.schedule(t.Context(), IntakeDependencies{}, requests[4]), "a full queue must apply backpressure")
	select {
	case pr := <-started:
		t.Fatalf("concurrency limit exceeded by %s", pr.ID)
	default:
	}
	once.Do(func() { close(release) })
	remaining := []models.FactoryPullRequest{next(), next(), next()}
	require.ElementsMatch(t, []models.FactoryPullRequest{requests[0], requests[2], requests[3]}, remaining)
}
