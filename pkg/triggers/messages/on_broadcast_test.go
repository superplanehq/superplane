package messages

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestOnBroadcast_Setup(t *testing.T) {
	sourceApp := &core.App{
		ID:   uuid.New().String(),
		Name: "Source App",
	}
	otherApp := &core.App{
		ID:   uuid.New().String(),
		Name: "Other App",
	}
	listenerCanvasID := uuid.New().String()

	t.Run("creates subscription on first setup", func(t *testing.T) {
		apps := &contexts.AppContext{CanvasID: listenerCanvasID}
		apps.RegisterApp(sourceApp)
		metadataCtx := &contexts.MetadataContext{}

		trigger := &OnBroadcast{}
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"app": sourceApp.ID},
			Apps:          apps,
			Metadata:      metadataCtx,
		})
		require.NoError(t, err)
		require.Equal(t, []string{sourceApp.ID}, apps.SubscribeCalls)
		require.Equal(t, 0, apps.UnsubscribeCalls)

		metadata, ok := metadataCtx.Metadata.(OnBroadcastMetadata)
		require.True(t, ok)
		require.NotNil(t, metadata.App)
		require.Equal(t, sourceApp.ID, metadata.App.ID)
		require.Equal(t, sourceApp.Name, metadata.App.Name)
	})

	t.Run("keeps subscription when app is unchanged", func(t *testing.T) {
		apps := &contexts.AppContext{CanvasID: listenerCanvasID}
		apps.RegisterApp(sourceApp)
		metadataCtx := &contexts.MetadataContext{}

		trigger := &OnBroadcast{}
		ctx := core.TriggerContext{
			Configuration: map[string]any{"app": sourceApp.ID},
			Apps:          apps,
			Metadata:      metadataCtx,
		}

		require.NoError(t, trigger.Setup(ctx))
		require.NoError(t, trigger.Setup(ctx))
		require.Equal(t, []string{sourceApp.ID}, apps.SubscribeCalls)
		require.Equal(t, 0, apps.UnsubscribeCalls)
	})

	t.Run("keeps subscription when republished by app name", func(t *testing.T) {
		apps := &contexts.AppContext{CanvasID: listenerCanvasID}
		apps.RegisterApp(sourceApp)
		metadataCtx := &contexts.MetadataContext{}

		trigger := &OnBroadcast{}
		ctx := core.TriggerContext{
			Configuration: map[string]any{"app": sourceApp.ID},
			Apps:          apps,
			Metadata:      metadataCtx,
		}

		require.NoError(t, trigger.Setup(ctx))

		ctx.Configuration = map[string]any{"app": sourceApp.Name}
		require.NoError(t, trigger.Setup(ctx))
		require.Equal(t, []string{sourceApp.ID}, apps.SubscribeCalls)
		require.Equal(t, 0, apps.UnsubscribeCalls)
	})

	t.Run("changes subscription when app changes", func(t *testing.T) {
		apps := &contexts.AppContext{CanvasID: listenerCanvasID}
		apps.RegisterApp(sourceApp)
		apps.RegisterApp(otherApp)
		metadataCtx := &contexts.MetadataContext{}

		trigger := &OnBroadcast{}
		ctx := core.TriggerContext{
			Configuration: map[string]any{"app": sourceApp.ID},
			Apps:          apps,
			Metadata:      metadataCtx,
		}

		require.NoError(t, trigger.Setup(ctx))

		ctx.Configuration = map[string]any{"app": otherApp.ID}
		require.NoError(t, trigger.Setup(ctx))
		require.Equal(t, []string{sourceApp.ID, otherApp.ID}, apps.SubscribeCalls)
		require.Equal(t, 1, apps.UnsubscribeCalls)

		metadata, ok := metadataCtx.Metadata.(OnBroadcastMetadata)
		require.True(t, ok)
		require.Equal(t, otherApp.ID, metadata.App.ID)
	})

	t.Run("unsubscribes when app lookup fails on update", func(t *testing.T) {
		apps := &contexts.AppContext{CanvasID: listenerCanvasID}
		apps.RegisterApp(sourceApp)
		metadataCtx := &contexts.MetadataContext{}
		missingAppID := uuid.New().String()

		trigger := &OnBroadcast{}
		ctx := core.TriggerContext{
			Configuration: map[string]any{"app": sourceApp.ID},
			Apps:          apps,
			Metadata:      metadataCtx,
		}

		require.NoError(t, trigger.Setup(ctx))

		ctx.Configuration = map[string]any{"app": missingAppID}
		err := trigger.Setup(ctx)
		require.Error(t, err)
		require.Equal(t, []string{sourceApp.ID}, apps.SubscribeCalls)
		require.Equal(t, 1, apps.UnsubscribeCalls)
	})

	t.Run("returns error when app is missing on create", func(t *testing.T) {
		apps := &contexts.AppContext{CanvasID: listenerCanvasID}
		metadataCtx := &contexts.MetadataContext{}

		trigger := &OnBroadcast{}
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"app": uuid.New().String()},
			Apps:          apps,
			Metadata:      metadataCtx,
		})
		require.Error(t, err)
		require.Empty(t, apps.SubscribeCalls)
		require.Equal(t, 0, apps.UnsubscribeCalls)
	})

	t.Run("returns error when app is required", func(t *testing.T) {
		apps := &contexts.AppContext{CanvasID: listenerCanvasID}
		metadataCtx := &contexts.MetadataContext{}

		trigger := &OnBroadcast{}
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"app": ""},
			Apps:          apps,
			Metadata:      metadataCtx,
		})
		require.ErrorContains(t, err, "app is required")
	})
}

func TestOnBroadcast_Cleanup(t *testing.T) {
	apps := &contexts.AppContext{CanvasID: uuid.New().String()}

	trigger := &OnBroadcast{}
	require.NoError(t, trigger.Cleanup(core.TriggerContext{Apps: apps}))
	require.Equal(t, 1, apps.UnsubscribeCalls)
}

func TestOnBroadcast_OnAppMessage(t *testing.T) {
	events := &contexts.EventContext{}
	message := map[string]any{"payload": map[string]any{"message": "hello"}}

	trigger := &OnBroadcast{}
	err := trigger.OnAppMessage(core.AppMessageContext{
		Message: message,
		Events:  events,
	})
	require.NoError(t, err)
	require.Len(t, events.Payloads, 1)
	require.Equal(t, "app.broadcast", events.Payloads[0].Type)
	require.Equal(t, message, events.Payloads[0].Data)
}

// Delivery filtering owns acceptance before a trigger event or run exists.
func TestOnBroadcast_Filter(t *testing.T) {
	message := map[string]any{"app": map[string]any{"id": "source"}, "node": map[string]any{"id": "producer"}, "payload": map[string]any{"type": "completed"}}
	for _, test := range []struct {
		name, filter string
		emits, fails bool
	}{
		{name: "omitted", emits: true},
		{name: "blank", filter: "  ", emits: true},
		{name: "match", filter: `message.payload.type == "completed"`, emits: true},
		{name: "nonmatch", filter: `message.payload.type == "other"`},
		{name: "nonboolean", filter: `message.payload.type`, fails: true},
		{name: "runtime error", filter: `len(message.payload.absent.value) > 0`, fails: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			events := &contexts.EventContext{}
			err := (&OnBroadcast{}).OnAppMessage(core.AppMessageContext{Message: message, Configuration: map[string]any{"filter": test.filter}, Events: events})
			if test.fails {
				require.Error(t, err)
				require.Empty(t, events.Payloads)
				return
			}
			require.NoError(t, err)
			if !test.emits {
				require.Empty(t, events.Payloads)
				return
			}
			require.Len(t, events.Payloads, 1)
			require.Equal(t, message, events.Payloads[0].Data)
		})
	}
}

func TestOnBroadcast_RejectsInvalidFilterBeforeSubscription(t *testing.T) {
	for _, filter := range []string{`message.payload.type ==`, `run().id == "x"`, `$["node"].data == true`, `root().data == true`, `42`} {
		apps := &contexts.AppContext{}
		err := (&OnBroadcast{}).Setup(core.TriggerContext{Configuration: map[string]any{"app": "source", "filter": filter}, Apps: apps, Metadata: &contexts.MetadataContext{}})
		require.ErrorContains(t, err, "filter")
		require.Empty(t, apps.SubscribeCalls)
	}
}
