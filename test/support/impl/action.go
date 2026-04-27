package impl

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DummyActionOptions struct {
	Name              string
	Hooks             []core.Hook
	SetupFunc         func(ctx core.SetupContext) error
	ProcessQueueFunc  func(ctx core.ProcessQueueContext) (*uuid.UUID, error)
	ExecuteFunc       func(ctx core.ExecutionContext) error
	HandleHookFunc    func(ctx core.ActionHookContext) error
	HandleWebhookFunc func(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error)
	CancelFunc        func(ctx core.ExecutionContext) error
	CleanupFunc       func(ctx core.SetupContext) error
}

type DummyAction struct {
	name              string
	hooks             []core.Hook
	setupFunc         func(ctx core.SetupContext) error
	processQueueFunc  func(ctx core.ProcessQueueContext) (*uuid.UUID, error)
	executeFunc       func(ctx core.ExecutionContext) error
	handleHookFunc    func(ctx core.ActionHookContext) error
	handleWebhookFunc func(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error)
	cancelFunc        func(ctx core.ExecutionContext) error
	cleanupFunc       func(ctx core.SetupContext) error
}

func NewDummyAction(options DummyActionOptions) *DummyAction {
	name := options.Name
	if name == "" {
		name = "dummy"
	}

	return &DummyAction{
		name:              name,
		hooks:             options.Hooks,
		setupFunc:         options.SetupFunc,
		processQueueFunc:  options.ProcessQueueFunc,
		executeFunc:       options.ExecuteFunc,
		handleHookFunc:    options.HandleHookFunc,
		handleWebhookFunc: options.HandleWebhookFunc,
		cancelFunc:        options.CancelFunc,
		cleanupFunc:       options.CleanupFunc,
	}
}

func (t *DummyAction) Name() string {
	return t.name
}

func (t *DummyAction) Label() string {
	return "dummy"
}

func (t *DummyAction) Description() string {
	return "Just a dummy component used in unit tests"
}

func (t *DummyAction) Documentation() string {
	return ""
}

func (t *DummyAction) Icon() string {
	return "dummy"
}

func (t *DummyAction) Color() string {
	return "dummy"
}

func (t *DummyAction) ExampleOutput() map[string]any {
	return nil
}

func (t *DummyAction) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (t *DummyAction) Configuration() []configuration.Field {
	return nil
}

func (t *DummyAction) Setup(ctx core.SetupContext) error {
	if t.setupFunc == nil {
		return nil
	}
	return t.setupFunc(ctx)
}

func (t *DummyAction) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	if t.processQueueFunc == nil {
		return nil, nil
	}
	return t.processQueueFunc(ctx)
}

func (t *DummyAction) Execute(ctx core.ExecutionContext) error {
	if t.executeFunc == nil {
		return nil
	}
	return t.executeFunc(ctx)
}

func (t *DummyAction) Hooks() []core.Hook {
	return t.hooks
}

func (t *DummyAction) HandleHook(ctx core.ActionHookContext) error {
	if t.handleHookFunc == nil {
		return nil
	}
	return t.handleHookFunc(ctx)
}

func (t *DummyAction) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	if t.handleWebhookFunc == nil {
		return 200, nil, nil
	}
	return t.handleWebhookFunc(ctx)
}

func (t *DummyAction) Cancel(ctx core.ExecutionContext) error {
	if t.cancelFunc == nil {
		return nil
	}
	return t.cancelFunc(ctx)
}

func (t *DummyAction) Cleanup(ctx core.SetupContext) error {
	if t.cleanupFunc == nil {
		return nil
	}
	return t.cleanupFunc(ctx)
}
