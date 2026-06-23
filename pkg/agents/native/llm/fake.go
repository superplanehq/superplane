package llm

import (
	"context"
	"fmt"
	"sync"
)

type ScriptedClient struct {
	mu      sync.Mutex
	calls   []StreamRequest
	scripts [][]StreamEvent
	errs    []error
}

func NewScriptedClient(scripts ...[]StreamEvent) *ScriptedClient {
	return &ScriptedClient{scripts: scripts}
}

func (c *ScriptedClient) SetErrors(errs ...error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errs = append([]error(nil), errs...)
}

func (c *ScriptedClient) Stream(ctx context.Context, req StreamRequest, onEvent func(StreamEvent) error) error {
	script, err := c.next(req)
	if err != nil {
		return err
	}
	for _, event := range script {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := onEvent(event); err != nil {
			return err
		}
	}
	return nil
}

func (c *ScriptedClient) Calls() []StreamRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]StreamRequest(nil), c.calls...)
}

func (c *ScriptedClient) next(req StreamRequest) ([]StreamEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.calls = append(c.calls, req)
	index := len(c.calls) - 1
	if index < len(c.errs) && c.errs[index] != nil {
		return nil, c.errs[index]
	}
	if index >= len(c.scripts) {
		return nil, fmt.Errorf("scripted llm: missing script for call %d", index)
	}
	return append([]StreamEvent(nil), c.scripts[index]...), nil
}
