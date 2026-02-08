package registry

import (
	"fmt"
	"runtime/debug"

	"github.com/superplanehq/superplane/pkg/core"
)

type PanicableWebhookHandler struct {
	underlying core.WebhookHandler
}

func NewPanicableWebhookHandler(underlying core.WebhookHandler) *PanicableWebhookHandler {
	return &PanicableWebhookHandler{underlying: underlying}
}

func (h *PanicableWebhookHandler) CompareConfig(a, b any) (bool, error) {
	return h.underlying.CompareConfig(a, b)
}

func (h *PanicableWebhookHandler) Setup(ctx core.WebhookHandlerContext) (metadata any, err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Webhook handler panicked in Setup(): %v\nStack: %s",
				r, debug.Stack())
			metadata = nil
			err = fmt.Errorf("webhook handler panicked in Setup(): %v", r)
		}
	}()
	return h.underlying.Setup(ctx)
}

func (h *PanicableWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Webhook handler panicked in Cleanup(): %v\nStack: %s",
				r, debug.Stack())
			err = fmt.Errorf("webhook handler panicked in Cleanup(): %v", r)
		}
	}()
	return h.underlying.Cleanup(ctx)
}
