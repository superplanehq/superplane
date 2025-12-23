# Application Development Guide

This guide explains how to add new applications and extend existing applications with new triggers and components in Superplane.

## Table of Contents

- [Overview](#overview)
- [Application Structure](#application-structure)
- [Creating a New Application](#creating-a-new-application)
- [Adding Triggers](#adding-triggers)
- [Adding Components](#adding-components)
- [Adding Frontend Mappers](#adding-frontend-mappers)
- [Example: GitHub Issues Trigger](#example-github-issues-trigger)

## Overview

Applications in Superplane are integrations with external services that allow users to trigger workflows and interact with those services. Applications consist of:

- **Backend implementation** (Go): Located in `pkg/applications/<app-name>/`
- **Frontend mappers** (TypeScript): Located in `web_src/src/pages/workflowv2/mappers/<app-name>/`

Each application can have:
- **Triggers**: Event sources that start workflow executions (e.g., "On Pull Request", "On Pipeline Done")
- **Components**: Actions that can be executed as part of workflows (e.g., "Run Workflow")

## Application Structure

### Backend Structure

Applications are organized in `pkg/applications/` with the following structure:

```
pkg/applications/
├── github/
│   ├── github.go           # Main application implementation
│   ├── client.go           # API client (if needed)
│   ├── on_pull_request.go  # Trigger implementation
│   ├── on_push.go          # Another trigger
│   └── on_issue.go         # Yet another trigger
└── semaphore/
    ├── semaphore.go
    ├── client.go
    └── ...
```

### Frontend Structure

Frontend mappers are organized in `web_src/src/pages/workflowv2/mappers/`:

```
web_src/src/pages/workflowv2/mappers/
├── github/
│   ├── index.ts            # Exports all trigger/component renderers
│   ├── on_pull_request.ts  # Trigger renderer
│   ├── on_push.ts
│   └── on_issue.ts
└── semaphore/
    └── ...
```

## Creating a New Application

To create a new application, you need to:

1. **Create the application package** in `pkg/applications/<app-name>/`

2. **Implement the main application file** (`<app-name>.go`):

```go
package myapp

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterApplication("myapp", &MyApp{})
}

type MyApp struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

type Metadata struct {
	// Store application-level metadata
}

func (a *MyApp) Name() string {
	return "myapp"
}

func (a *MyApp) Label() string {
	return "My Application"
}

func (a *MyApp) Icon() string {
	return "icon-name"
}

func (a *MyApp) Description() string {
	return "Description of what this application does"
}

func (a *MyApp) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Your API key",
			Required:    true,
		},
	}
}

func (a *MyApp) Components() []core.Component {
	return []core.Component{
		// Add your components here
	}
}

func (a *MyApp) Triggers() []core.Trigger {
	return []core.Trigger{
		// Add your triggers here
	}
}

func (a *MyApp) Sync(ctx core.SyncContext) error {
	// Validate configuration and set up the application
	// Set state to "ready" when done: ctx.AppInstallation.SetState("ready", "")
	return nil
}

func (a *MyApp) HandleRequest(ctx core.HTTPRequestContext) {
	// Handle incoming HTTP requests (e.g., OAuth callbacks, webhooks)
}
```

3. **Register the application** in the `init()` function (shown above)

## Adding Triggers

Triggers listen to external events and start workflow executions. Here's how to add a new trigger:

### 1. Create the Trigger File

Create a new file in your application package (e.g., `on_event.go`):

```go
package myapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnEvent struct{}

type OnEventMetadata struct {
	// Store trigger-specific metadata
	Resource string `json:"resource"`
}

type OnEventConfiguration struct {
	Resource string   `json:"resource"`
	Actions  []string `json:"action"`
}

func (t *OnEvent) Name() string {
	return "myapp.onEvent"
}

func (t *OnEvent) Label() string {
	return "On Event"
}

func (t *OnEvent) Description() string {
	return "Listen to event occurrences"
}

func (t *OnEvent) Icon() string {
	return "icon-name"
}

func (t *OnEvent) Color() string {
	return "gray"
}

func (t *OnEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "resource",
			Label:    "Resource",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "actions",
			Label:    "Actions",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"created"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "created"},
						{Label: "Updated", Value: "updated"},
						{Label: "Deleted", Value: "deleted"},
					},
				},
			},
		},
	}
}

func (t *OnEvent) Setup(ctx core.TriggerContext) error {
	var metadata OnEventMetadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// If metadata is already set, trigger is already setup
	if metadata.Resource != "" {
		return nil
	}

	config := OnEventConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Validate configuration
	if config.Resource == "" {
		return fmt.Errorf("resource is required")
	}

	// Store metadata
	metadata.Resource = config.Resource
	err = ctx.MetadataContext.Set(metadata)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	// Request webhook if needed
	return ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		EventType: "event",
		Resource:  config.Resource,
	})
}

func (t *OnEvent) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnEvent) HandleAction(ctx core.TriggerActionContext) error {
	return nil
}

func (t *OnEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// Validate webhook signature
	signature := ctx.Headers.Get("X-Signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	// Verify the signature
	secret, err := ctx.WebhookContext.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, err
	}

	// Parse the webhook payload
	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Filter by action type
	config := OnEventConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	action, ok := data["action"]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("missing action")
	}

	if !slices.Contains(config.Actions, action.(string)) {
		return http.StatusOK, nil
	}

	// Emit the event to trigger workflow execution
	err = ctx.EventContext.Emit(data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
```

### 2. Register the Trigger

Add the trigger to your application's `Triggers()` method:

```go
func (a *MyApp) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnEvent{},
	}
}
```

### 3. Implement Webhook Setup (if needed)

If your triggers or components require webhooks, implement the webhook setup methods in your main application file:

```go
type WebhookConfiguration struct {
	EventType string `json:"eventType"`
	Resource  string `json:"resource"`
}

// CompareWebhookConfig defines when two webhook configurations are equal.
// This is used to determine if an existing webhook can be reused.
func (a *MyApp) CompareWebhookConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	// Define equality based on your application's webhook configuration.
	// Webhooks with matching configurations can be shared across multiple triggers/components.
	return configA.Resource == configB.Resource && configA.EventType == configB.EventType, nil
}

// SetupWebhook creates a webhook in the external service.
// This is called by the webhook provisioner for pending webhook records.
func (a *MyApp) SetupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) (any, error) {
	// Create webhook in the external service
	// Return metadata about the created webhook (e.g., webhook ID)
	return nil, nil
}

// CleanupWebhook deletes a webhook from the external service.
// This is called by the webhook cleanup worker for deleted webhook records.
func (a *MyApp) CleanupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) error {
	// Delete webhook from the external service using the metadata
	return nil
}
```

**Webhook Logic Overview:**

The webhook management logic is centralized in `AppInstallationContext.RequestWebhook()`. When a trigger or component requests a webhook:

1. The context lists all existing webhooks for the app installation
2. For each existing webhook, it calls your application's `CompareWebhookConfig()` to check if configurations match
3. If a match is found, the node is associated with the existing webhook
4. If no match is found, a new webhook is created

This means multiple triggers and components can share the same webhook if they have matching configurations, reducing the number of webhooks created in external services.

## Adding Components

Components are actions that can be executed as part of workflows. The process is similar to triggers:

1. Create a new file for your component (e.g., `do_action.go`)
2. Implement the `core.Component` interface
3. Register it in your application's `Components()` method

## Adding Frontend Mappers

Frontend mappers render triggers and components in the UI. They define how events are displayed and what information is shown to users.

### 1. Create the Mapper File

Create a new file in `web_src/src/pages/workflowv2/mappers/<app-name>/` (e.g., `on_event.ts`):

```typescript
import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import appIcon from "@/assets/icons/integrations/<app-name>.svg";
import { TriggerProps } from "@/ui/trigger";

interface OnEventMetadata {
  resource: string;
}

interface OnEventConfiguration {
  actions: string[];
}

interface OnEventEventData {
  action?: string;
  // Add other fields from your webhook payload
}

/**
 * Renderer for the "myapp.onEvent" trigger
 */
export const onEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data as OnEventEventData;

    return {
      title: `Event occurred`,
      subtitle: eventData?.action || "",
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data as OnEventEventData;

    return {
      Action: eventData?.action || "",
      // Add other relevant fields
    };
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as unknown as OnEventMetadata;
    const configuration = node.configuration as unknown as OnEventConfiguration;
    const metadataItems = [];

    if (metadata?.resource) {
      metadataItems.push({
        icon: "database",
        label: metadata.resource,
      });
    }

    if (configuration?.actions) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.actions.join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: appIcon,
      iconBackground: "bg-white",
      iconColor: getColorClass(trigger.color),
      headerColor: getBackgroundColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnEventEventData;

      props.lastEventData = {
        title: `Event occurred`,
        subtitle: eventData?.action || "",
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
```

### 2. Register the Mapper

Update or create `index.ts` in your application's mapper directory:

```typescript
import { ComponentBaseMapper, TriggerRenderer } from "../types";
import { onEventTriggerRenderer } from "./on_event";

export const componentMappers: Record<string, ComponentBaseMapper> = {};

export const triggerRenderers: Record<string, TriggerRenderer> = {
  onEvent: onEventTriggerRenderer,
};
```

## Example: GitHub Issues Trigger

Here's a complete example of the GitHub Issues trigger that was recently added:

### Backend (`pkg/applications/github/on_issue.go`)

The trigger implements:
- Configuration fields for repository and action filtering
- Setup method to validate and store metadata
- Webhook handling with signature verification
- Action filtering to only emit events for configured actions

Key features:
- Supports 16 different issue action types (opened, closed, labeled, etc.)
- Validates that the repository is accessible to the GitHub app installation
- Filters events by action type before emitting

### Frontend (`web_src/src/pages/workflowv2/mappers/github/on_issue.ts`)

The mapper provides:
- `getTitleAndSubtitle`: Formats event display as "#123 - Issue title"
- `getRootEventValues`: Extracts key fields (URL, Title, Action, Author, State)
- `getTriggerProps`: Renders the trigger node with repository and action metadata

## Testing

After implementing your application:

1. **Build and format**: Run `make format.go && make lint && make check.build.app`
2. **Test the backend**: Run `make test`
3. **Test the UI**: Run `make check.build.ui`
4. **E2E tests**: Consider adding E2E tests (see [e2e_tests.md](e2e_tests.md))

## Best Practices

1. **Use descriptive names**: Trigger and component names should clearly indicate what they do
2. **Validate configuration**: Always validate configuration in the `Setup()` method
3. **Handle errors gracefully**: Return appropriate HTTP status codes and error messages
4. **Use metadata for caching**: Store frequently accessed data in metadata to avoid repeated API calls
5. **Filter early**: Filter events as early as possible to avoid unnecessary processing
6. **Default configuration**: The default configuration should be thought out in a way to cover the most common use case, and to avoid generating unnecessary events. For example, the `github.onPush` trigger filters by only the commits on the `main` branch by default.
7. **Verify signatures**: Always verify webhook signatures to ensure authenticity
8. **Document action types**: Clearly document all available action types for triggers
9. **Consistent styling**: Follow the existing patterns for frontend mappers

## References

- GitHub application: `pkg/applications/github/`
- Semaphore application: `pkg/applications/semaphore/`
- Core interfaces: `pkg/core/`
- Frontend mappers: `web_src/src/pages/workflowv2/mappers/`
