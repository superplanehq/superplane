package compute

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test_OnVMInstance_Metadata(t *testing.T) {
	trigger := &OnVMInstance{}
	assert.Equal(t, "gcp.compute.onVMInstance", trigger.Name())
	assert.Equal(t, "Compute • On VM Instance", trigger.Label())
	assert.Equal(t, "Listen to GCP Compute Engine VM instance lifecycle events", trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
	assert.Equal(t, "gcp", trigger.Icon())
	assert.Equal(t, "gray", trigger.Color())

	assert.Len(t, trigger.Actions(), 1)
	assert.Equal(t, "provisionSink", trigger.Actions()[0].Name)
}

func Test_OnVMInstance_Configuration(t *testing.T) {
	trigger := &OnVMInstance{}
	fields := trigger.Configuration()
	assert.Nil(t, fields)
}

func Test_OnVMInstance_ExampleData(t *testing.T) {
	trigger := &OnVMInstance{}
	data := trigger.ExampleData()
	assert.Equal(t, computeServiceName, data["serviceName"])
	assert.Equal(t, instancesInsertMethod, data["methodName"])
	assert.Equal(t, "projects/my-project/zones/us-central1-a/instances/my-vm", data["resourceName"])
}

func Test_OnVMInstance_SinkFilter(t *testing.T) {
	assert.Contains(t, SinkFilter, "compute.googleapis.com")
	assert.Contains(t, SinkFilter, "v1.compute.instances.insert")
	assert.Contains(t, SinkFilter, "beta.compute.instances.insert")
	assert.Contains(t, SinkFilter, "compute.instances.insert")
}

func Test_OnVMInstance_OnIntegrationMessage(t *testing.T) {
	trigger := &OnVMInstance{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("wrong service name does not emit", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  "storage.googleapis.com",
				"methodName":   instancesInsertMethod,
				"resourceName": "projects/p/zones/z/instances/vm1",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("wrong method name does not emit", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  computeServiceName,
				"methodName":   "v1.compute.instances.delete",
				"resourceName": "projects/p/zones/z/instances/vm1",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("VM insert event emits", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  computeServiceName,
				"methodName":   instancesInsertMethod,
				"resourceName": "projects/my-proj/zones/us-central1-a/instances/my-vm",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, EmittedEventType, events.Payloads[0].Type)
	})

	t.Run("beta method name emits", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  computeServiceName,
				"methodName":   instancesInsertMethodBeta,
				"resourceName": "projects/p/zones/z/instances/vm1",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, EmittedEventType, events.Payloads[0].Type)
	})

	t.Run("short method name emits", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  computeServiceName,
				"methodName":   instancesInsertMethodShort,
				"resourceName": "projects/p/zones/z/instances/vm1",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, EmittedEventType, events.Payloads[0].Type)
	})
}

func Test_sanitizeSinkID(t *testing.T) {
	assert.Equal(t, "abc123", sanitizeSinkID("ABC-123"))
	assert.Equal(t, "abc-def", sanitizeSinkID("abc-def"))
	assert.Equal(t, "a", sanitizeSinkID("a!@#$%^&*()"))

	long := ""
	for i := 0; i < 100; i++ {
		long += "a"
	}
	assert.LessOrEqual(t, len(sanitizeSinkID(long)), 80)
}
