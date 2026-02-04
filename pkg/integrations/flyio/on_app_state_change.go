package flyio

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnAppStateChange struct{}

type OnAppStateChangeConfiguration struct {
	OrgSlug         string `json:"orgSlug" mapstructure:"orgSlug"`
	PollingInterval int    `json:"pollingInterval" mapstructure:"pollingInterval"`
}

type OnAppStateChangeMetadata struct {
	OrgSlug   string            `json:"orgSlug"`
	AppStates map[string]string `json:"appStates"` // appName -> status
	LastPoll  string            `json:"lastPoll"`
}

func (t *OnAppStateChange) Name() string {
	return "flyio.onAppStateChange"
}

func (t *OnAppStateChange) Label() string {
	return "On App State Change"
}

func (t *OnAppStateChange) Description() string {
	return "Listen to Fly.io App state changes via polling"
}

func (t *OnAppStateChange) Documentation() string {
	return `The On App State Change trigger starts a workflow execution when an App's status changes.

## How It Works

This trigger polls the Fly.io Apps API at a configurable interval to detect status changes. When an App's status changes or a new App is created/deleted, an event is emitted.

## Use Cases

- **Deployment monitoring**: React when Apps are deployed or suspended
- **Resource tracking**: Detect when new Apps are created in your organization
- **Alerting**: Get notified when Apps go into unexpected states

## Configuration

- **Organization Slug**: The Fly.io organization to monitor (defaults to 'personal')
- **Polling Interval**: How often to check for state changes (in seconds)

## App Statuses

Common Fly.io App statuses:
- **deployed**: App has running Machines
- **pending**: App is being set up
- **suspended**: App is suspended

## Event Data

Each state change event includes:
- **appName**: The App name
- **previousStatus**: The previous status (if known)
- **currentStatus**: The new status
- **machineCount**: Number of Machines in the App
- **eventType**: Type of change (created, updated, deleted)`
}

func (t *OnAppStateChange) Icon() string {
	return "activity"
}

func (t *OnAppStateChange) Color() string {
	return "purple"
}

func (t *OnAppStateChange) ExampleData() map[string]any {
	return map[string]any{
		"appName":        "my-fly-app",
		"previousStatus": "pending",
		"currentStatus":  "deployed",
		"machineCount":   2,
		"eventType":      "updated",
	}
}

func (t *OnAppStateChange) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "orgSlug",
			Label:       "Organization Slug",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "personal",
			Description: "The Fly.io organization to monitor for App state changes",
		},
		{
			Name:        "pollingInterval",
			Label:       "Polling Interval (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     60,
			Description: "How often to poll for state changes (minimum 30 seconds)",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(30),
					Max: intPtr(3600),
				},
			},
		},
	}
}

func (t *OnAppStateChange) Setup(ctx core.TriggerContext) error {
	config := OnAppStateChangeConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	orgSlug := config.OrgSlug
	if orgSlug == "" {
		orgSlug = "personal"
	}

	// Get existing metadata
	metadata := OnAppStateChangeMetadata{}
	_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)

	// If already set up for this org, skip
	if metadata.OrgSlug == orgSlug && metadata.AppStates != nil {
		return nil
	}

	// Initialize by fetching current app states
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	apps, err := client.ListApps(orgSlug)
	if err != nil {
		return fmt.Errorf("failed to list apps: %w", err)
	}

	// Store initial states
	appStates := make(map[string]string)
	for _, app := range apps {
		appStates[app.Name] = app.Status
	}

	now := time.Now().Format(time.RFC3339)
	if err := ctx.Metadata.Set(OnAppStateChangeMetadata{
		OrgSlug:   orgSlug,
		AppStates: appStates,
		LastPoll:  now,
	}); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	// Schedule first poll
	interval := config.PollingInterval
	if interval < 30 {
		interval = 60 // default
	}

	return ctx.Requests.ScheduleActionCall(
		"poll",
		map[string]any{},
		time.Duration(interval)*time.Second,
	)
}

func (t *OnAppStateChange) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			Description:    "Poll for app state changes",
			UserAccessible: false,
		},
	}
}

func (t *OnAppStateChange) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "poll":
		return nil, t.poll(ctx)
	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (t *OnAppStateChange) poll(ctx core.TriggerActionContext) error {
	config := OnAppStateChangeConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	orgSlug := config.OrgSlug
	if orgSlug == "" {
		orgSlug = "personal"
	}

	metadata := OnAppStateChangeMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	apps, err := client.ListApps(orgSlug)
	if err != nil {
		ctx.Logger.Warnf("Failed to list apps: %v - will retry", err)
		return t.scheduleNextPoll(ctx, config.PollingInterval)
	}

	// Compare with stored states
	newStates := make(map[string]string)
	for _, app := range apps {
		newStates[app.Name] = app.Status

		previousStatus, exists := metadata.AppStates[app.Name]

		// Emit event if status changed or new app appeared
		if !exists {
			// New app created
			event := map[string]any{
				"appName":        app.Name,
				"previousStatus": "",
				"currentStatus":  app.Status,
				"machineCount":   app.MachineCount,
				"eventType":      "created",
			}
			if err := ctx.Events.Emit("flyio.appStateChange", event); err != nil {
				ctx.Logger.Warnf("Failed to emit event: %v", err)
			}
		} else if previousStatus != app.Status {
			// Status changed
			event := map[string]any{
				"appName":        app.Name,
				"previousStatus": previousStatus,
				"currentStatus":  app.Status,
				"machineCount":   app.MachineCount,
				"eventType":      "updated",
			}
			if err := ctx.Events.Emit("flyio.appStateChange", event); err != nil {
				ctx.Logger.Warnf("Failed to emit event: %v", err)
			}
		}
	}

	// Check for deleted apps
	for appName, previousStatus := range metadata.AppStates {
		if _, exists := newStates[appName]; !exists {
			event := map[string]any{
				"appName":        appName,
				"previousStatus": previousStatus,
				"currentStatus":  "deleted",
				"eventType":      "deleted",
			}
			if err := ctx.Events.Emit("flyio.appStateChange", event); err != nil {
				ctx.Logger.Warnf("Failed to emit event: %v", err)
			}
		}
	}

	// Update metadata
	now := time.Now().Format(time.RFC3339)
	if err := ctx.Metadata.Set(OnAppStateChangeMetadata{
		OrgSlug:   orgSlug,
		AppStates: newStates,
		LastPoll:  now,
	}); err != nil {
		ctx.Logger.Warnf("Failed to update metadata: %v", err)
	}

	return t.scheduleNextPoll(ctx, config.PollingInterval)
}

func (t *OnAppStateChange) scheduleNextPoll(ctx core.TriggerActionContext, interval int) error {
	if interval < 30 {
		interval = 60
	}

	return ctx.Requests.ScheduleActionCall(
		"poll",
		map[string]any{},
		time.Duration(interval)*time.Second,
	)
}

func intPtr(v int) *int {
	return &v
}

func (t *OnAppStateChange) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// no-op - this trigger uses polling, not webhooks
	return http.StatusOK, nil
}

func (t *OnAppStateChange) Cleanup(ctx core.TriggerContext) error {
	return nil
}
