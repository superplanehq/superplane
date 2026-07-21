package tools

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type Hello struct{}

func (h *Hello) Name() string {
	return "semaphore.hello"
}

func (h *Hello) Label() string {
	return "Hello"
}

func (h *Hello) Description() string {
	return "Say hello to the world"
}

func (h *Hello) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (h *Hello) Call(ctx core.IntegrationToolContext) (any, error) {
	return map[string]any{
		"message": "Hello, from semaphore custom tool",
	}, nil
}
