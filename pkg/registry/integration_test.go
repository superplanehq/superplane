package registry

import (
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// panickingIntegration is an integration that panics in all panicable methods
type panickingIntegration struct{}

func (p *panickingIntegration) Name() string                         { return "panicking-integration" }
func (p *panickingIntegration) Label() string                        { return "Panicking Integration" }
func (p *panickingIntegration) Icon() string                         { return "icon" }
func (p *panickingIntegration) Description() string                  { return "description" }
func (p *panickingIntegration) Instructions() string                 { return "instructions" }
func (p *panickingIntegration) Configuration() []configuration.Field { return nil }
func (p *panickingIntegration) Components() []core.Component         { return nil }
func (p *panickingIntegration) Triggers() []core.Trigger             { return nil }
func (p *panickingIntegration) Sync(ctx core.SyncContext) error      { panic("sync panic") }
func (p *panickingIntegration) Actions() []core.Action               { return nil }

func (p *panickingIntegration) HandleAction(ctx core.IntegrationActionContext) error {
	panic("handle action panic")
}

func (p *panickingIntegration) Cleanup(ctx core.IntegrationCleanupContext) error {
	panic("cleanup panic")
}

func (p *panickingIntegration) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	panic("list resources panic")
}
func (p *panickingIntegration) HandleRequest(ctx core.HTTPRequestContext) {
	panic("handle request panic")
}
func TestPanicableIntegration_Sync_CatchesPanic(t *testing.T) {
	integration := &panickingIntegration{}
	panicable := NewPanicableIntegration(integration)
	ctx := core.SyncContext{}

	err := panicable.Sync(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "integration panicking-integration panicked in Sync()")
	assert.Contains(t, err.Error(), "sync panic")
}

func TestPanicableIntegration_HandleRequest_CatchesPanic(t *testing.T) {
	integration := &panickingIntegration{}
	panicable := NewPanicableIntegration(integration)
	recorder := httptest.NewRecorder()
	ctx := core.HTTPRequestContext{
		Response: recorder,
		Logger:   log.NewEntry(log.StandardLogger()),
	}

	panicable.HandleRequest(ctx)

	assert.Equal(t, 500, recorder.Code)
}
