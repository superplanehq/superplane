package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/productanalytics"
)

type analyticsCloseRecorder struct {
	closed bool
}

func (r *analyticsCloseRecorder) Capture(productanalytics.Event) {}

func (r *analyticsCloseRecorder) IdentifyOrganizationGroup(productanalytics.OrganizationGroupIdentity) {
}

func (r *analyticsCloseRecorder) Close() error {
	r.closed = true
	return nil
}

func (r *analyticsCloseRecorder) Enabled() bool {
	return true
}

func TestProcessShutdownFlushesAnalyticsWithoutRunner(t *testing.T) {
	recorder := &analyticsCloseRecorder{}
	previous := productanalytics.SetCurrent(recorder)
	t.Cleanup(func() {
		productanalytics.SetCurrent(previous)
	})

	(&processShutdown{}).stop()

	assert.True(t, recorder.closed)
}

func TestProcessShutdownStopsRunnerBeforeAnalyticsFlush(t *testing.T) {
	recorder := &analyticsCloseRecorder{}
	previous := productanalytics.SetCurrent(recorder)
	t.Cleanup(func() {
		productanalytics.SetCurrent(previous)
	})

	order := []string{}
	notificationsStopped := false
	shutdown := &processShutdown{}
	shutdown.setRunner(func() {
		order = append(order, "runner")
		assert.False(t, recorder.closed)
	}, func() {
		notificationsStopped = true
	})

	shutdown.stop()

	assert.True(t, notificationsStopped)
	assert.Equal(t, []string{"runner"}, order)
	assert.True(t, recorder.closed)
}
