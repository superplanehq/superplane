package registry

import (
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// panickingApplication is an application that panics in all panicable methods
type panickingApplication struct{}

func (p *panickingApplication) Name() string                         { return "panicking-app" }
func (p *panickingApplication) Label() string                        { return "Panicking App" }
func (p *panickingApplication) Icon() string                         { return "icon" }
func (p *panickingApplication) Description() string                  { return "description" }
func (p *panickingApplication) Configuration() []configuration.Field { return nil }
func (p *panickingApplication) Components() []core.Component         { return nil }
func (p *panickingApplication) Triggers() []core.Trigger             { return nil }
func (p *panickingApplication) Sync(ctx core.SyncContext) error      { panic("sync panic") }
func (p *panickingApplication) HandleRequest(ctx core.HTTPRequestContext) {
	panic("handle request panic")
}
func (p *panickingApplication) CompareWebhookConfig(a, b any) (bool, error) {
	panic("compare webhook config panic")
}
func (p *panickingApplication) SetupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) (any, error) {
	panic("setup webhook panic")
}
func (p *panickingApplication) CleanupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) error {
	panic("cleanup webhook panic")
}

func TestPanicableApplication_Sync_CatchesPanic(t *testing.T) {
	app := &panickingApplication{}
	panicable := NewPanicableApplication(app)
	ctx := core.SyncContext{}

	err := panicable.Sync(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-app panicked in Sync()")
	assert.Contains(t, err.Error(), "sync panic")
}

func TestPanicableApplication_HandleRequest_CatchesPanic(t *testing.T) {
	app := &panickingApplication{}
	panicable := NewPanicableApplication(app)
	recorder := httptest.NewRecorder()
	ctx := core.HTTPRequestContext{
		Response: recorder,
		Logger:   log.NewEntry(log.StandardLogger()),
	}

	panicable.HandleRequest(ctx)

	assert.Equal(t, 500, recorder.Code)
}

func TestPanicableApplication_CompareWebhookConfig_CatchesPanic(t *testing.T) {
	app := &panickingApplication{}
	panicable := NewPanicableApplication(app)

	result, err := panicable.CompareWebhookConfig(nil, nil)

	assert.False(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-app panicked in CompareWebhookConfig()")
	assert.Contains(t, err.Error(), "compare webhook config panic")
}

func TestPanicableApplication_SetupWebhook_CatchesPanic(t *testing.T) {
	app := &panickingApplication{}
	panicable := NewPanicableApplication(app)
	ctx := &contexts.AppInstallationContext{}
	options := core.WebhookOptions{}

	metadata, err := panicable.SetupWebhook(ctx, options)

	require.Error(t, err)
	assert.Nil(t, metadata)
	assert.Contains(t, err.Error(), "panicking-app panicked in SetupWebhook()")
	assert.Contains(t, err.Error(), "setup webhook panic")
}

func TestPanicableApplication_CleanupWebhook_CatchesPanic(t *testing.T) {
	app := &panickingApplication{}
	panicable := NewPanicableApplication(app)
	ctx := &contexts.AppInstallationContext{}
	options := core.WebhookOptions{}

	err := panicable.CleanupWebhook(ctx, options)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-app panicked in CleanupWebhook()")
	assert.Contains(t, err.Error(), "cleanup webhook panic")
}
