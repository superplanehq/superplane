package registry

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

// panickingWebhookHandler is a webhook handler that panics in all panicable methods
type panickingWebhookHandler struct{}

func (p *panickingWebhookHandler) CompareConfig(a, b any) (bool, error) {
	panic("compare config panic")
}

func (p *panickingWebhookHandler) Merge(current, requested any) (any, bool, error) {
	panic("merge panic")
}

func (p *panickingWebhookHandler) Setup(ctx core.WebhookHandlerContext) (metadata any, err error) {
	panic("setup panic")
}

func (p *panickingWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	panic("cleanup panic")
}

func Test_PanicableWebhookHandler_Setup_CatchesPanic(t *testing.T) {
	handler := &panickingWebhookHandler{}
	panicable := NewPanicableWebhookHandler(handler)
	ctx := core.WebhookHandlerContext{
		Logger: log.NewEntry(log.StandardLogger()),
	}

	_, err := panicable.Setup(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "setup panic")
}

func Test_PanicableWebhookHandler_Cleanup_CatchesPanic(t *testing.T) {
	handler := &panickingWebhookHandler{}
	panicable := NewPanicableWebhookHandler(handler)
	ctx := core.WebhookHandlerContext{
		Logger: log.NewEntry(log.StandardLogger()),
	}

	err := panicable.Cleanup(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cleanup panic")
}

func Test_PanicableWebhookHandler_Merge_CatchesPanic(t *testing.T) {
	handler := &panickingWebhookHandler{}
	panicable := NewPanicableWebhookHandler(handler)

	merged, changed, err := panicable.Merge(map[string]any{"a": "b"}, nil)
	require.Error(t, err)
	assert.False(t, changed)
	assert.Equal(t, map[string]any{"a": "b"}, merged)
	assert.Contains(t, err.Error(), "merge panic")
}
