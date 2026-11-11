package licensing_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
)

type recordingLifecycle struct {
	service    *licensing.Service
	err        error
	mu         sync.Mutex
	statuses   []licensing.Status
	deadlocked atomic.Bool
}

func (h *recordingLifecycle) HandleStatus(context.Context, licensing.Status) error {
	if h.service != nil {
		done := make(chan struct{})
		go func() {
			_ = h.service.Status()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			h.deadlocked.Store(true)
			return errors.New("Status deadlocked while the lifecycle handler ran")
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.service != nil {
		h.statuses = append(h.statuses, h.service.Status())
	}
	return h.err
}

func (h *recordingLifecycle) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.statuses)
}

func TestServiceNotifiesLifecycleAfterSuccessfulLoads(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	source := &memorySource{}
	handler := &recordingLifecycle{}
	service := licensing.NewService(
		licensing.NewVerifier(licensingtest.KeySet(issuer)),
		source,
		licensing.WithLifecycleHandler(handler),
	)
	handler.service = service

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.Start(ctx)
	require.Equal(t, 1, handler.count())
	assert.False(t, handler.deadlocked.Load())

	require.NoError(t, service.Refresh(ctx))
	require.Equal(t, 2, handler.count())

	_, err := service.Install(ctx, issuer.License(licensing.FeatureGroups), uuid.New())
	require.NoError(t, err)
	require.Equal(t, 3, handler.count())
	assert.Equal(t, licensing.StateActive, handler.statuses[2].State)
	assert.Equal(t, licensing.FeatureGroups, handler.statuses[2].License.Features[0])

	_, err = service.Remove(ctx, uuid.New())
	require.NoError(t, err)
	require.Equal(t, 4, handler.count())
	assert.Equal(t, licensing.StateNone, handler.statuses[3].State)
	assert.False(t, handler.deadlocked.Load())
}

func TestServiceSkipsLifecycleWhenReloadFailsOrLicenseIsDevelopment(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	source := &memorySource{raw: issuer.License(licensing.FeatureGroups)}
	handler := &recordingLifecycle{}
	service := licensing.NewService(
		licensing.NewVerifier(licensingtest.KeySet(issuer)),
		source,
		licensing.WithLifecycleHandler(handler),
	)
	handler.service = service
	require.NoError(t, service.Refresh(context.Background()))
	require.Equal(t, 1, handler.count())

	source.err = errors.New("connection refused")
	require.Error(t, service.Refresh(context.Background()))
	assert.Equal(t, 1, handler.count())

	_, err := service.Install(context.Background(), issuer.License(licensing.FeatureCustomRoles), uuid.New())
	require.Error(t, err)
	assert.Equal(t, 1, handler.count())

	source.raw = nil
	_, err = service.Remove(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 1, handler.count(), "a failed reload does not confirm removal")

	devIssuer := licensingtest.NewIssuer(licensing.DevelopmentKeyID)
	devHandler := &recordingLifecycle{}
	devService := licensing.NewService(
		licensing.NewVerifier(licensingtest.KeySet(devIssuer)),
		&memorySource{raw: devIssuer.License(licensing.FeatureGroups)},
		licensing.WithLifecycleHandler(devHandler),
	)
	devHandler.service = devService
	require.NoError(t, devService.Refresh(context.Background()))
	assert.True(t, devService.Status().IsDevelopment())
	assert.Equal(t, 0, devHandler.count())
	assert.True(t, devService.IsEntitled(licensing.FeatureGroups))
}

func TestServiceLifecycleErrorDoesNotFailTheLoad(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	handler := &recordingLifecycle{err: errors.New("database unavailable")}
	service := licensing.NewService(
		licensing.NewVerifier(licensingtest.KeySet(issuer)),
		&memorySource{raw: issuer.License(licensing.FeatureGroups)},
		licensing.WithLifecycleHandler(handler),
	)
	handler.service = service

	require.NoError(t, service.Refresh(context.Background()))
	assert.True(t, service.IsEntitled(licensing.FeatureGroups))

	_, err := service.Install(context.Background(), issuer.License(licensing.FeatureGroups), uuid.New())
	require.NoError(t, err)

	_, err = service.Remove(context.Background(), uuid.New())
	require.NoError(t, err)
}
