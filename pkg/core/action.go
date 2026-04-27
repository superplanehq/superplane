package core

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
)

type Action interface {

	/*
	 * The unique identifier for the component.
	 * This is how nodes reference it, and is used for registration.
	 */
	Name() string

	/*
	 * The label for the component.
	 * This is how nodes are displayed in the UI.
	 */
	Label() string

	/*
	 * A good description of what the component does.
	 * Helpful for documentation and user interfaces.
	 */
	Description() string

	/*
	 * Detailed markdown documentation explaining how to use the component.
	 * This should provide in-depth information about the component's purpose,
	 * configuration options, use cases, and examples.
	 */
	Documentation() string

	/*
	 * The icon for the component.
	 * This is used in the UI to represent the component.
	 */
	Icon() string

	/*
	 * The color for the component.
	 * This is used in the UI to represent the component.
	 */
	Color() string

	/*
	 * Example output data for the component.
	 */
	ExampleOutput() map[string]any

	/*
	 * The output channels used by the component.
	 * If none is returned, the 'default' one is used.
	 */
	OutputChannels(configuration any) []OutputChannel

	/*
	 * The configuration fields exposed by the component.
	 */
	Configuration() []configuration.Field

	/*
	 * Setup the component.
	 */
	Setup(ctx SetupContext) error

	/*
	 * ProcessQueueItem is called when a queue item for this component's node
	 * is ready to be processed. Implementations should create the appropriate
	 * execution or handle the item synchronously using the provided context.
	 */
	ProcessQueueItem(ctx ProcessQueueContext) (*uuid.UUID, error)

	/*
	 * Passes full execution control to the component.
	 *
	 * Component execution has full control over the execution state,
	 * so it is the responsibility of the component to control it.
	 *
	 * Components should finish the execution or move it to waiting state.
	 * Components can also implement async components by combining Execute() and HandleHook().
	 */
	Execute(ctx ExecutionContext) error

	/*
	 * Allows components to define and execute custom hooks.
	 */
	Hooks() []Hook
	HandleHook(ctx ActionHookContext) error

	/*
	 * Handler for webhooks.
	 */
	HandleWebhook(ctx WebhookRequestContext) (int, *WebhookResponseBody, error)

	/*
	 * Cancel allows components to handle cancellation of executions.
	 * Default behavior does nothing. Components can override to perform
	 * cleanup or cancel external resources.
	 */
	Cancel(ctx ExecutionContext) error

	/*
	 * Cleanup allows components to clean up resources after being removed from a canvas.
	 * Default behavior does nothing. Components can override to perform cleanup.
	 */
	Cleanup(ctx SetupContext) error
}

type OutputChannel struct {
	Name        string
	Label       string
	Description string
}

var DefaultOutputChannel = OutputChannel{Name: "default", Label: "Default"}
