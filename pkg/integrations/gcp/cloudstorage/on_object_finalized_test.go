package cloudstorage

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test_OnObjectFinalized_Metadata(t *testing.T) {
	trigger := &OnObjectFinalized{}
	assert.Equal(t, "gcp.cloudstorage.onObjectFinalized", trigger.Name())
	assert.Equal(t, "Cloud Storage • On Object Finalized", trigger.Label())
	assert.NotEmpty(t, trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
	assert.Equal(t, "gcp", trigger.Icon())
	assert.Equal(t, "gray", trigger.Color())

	assert.Len(t, trigger.Actions(), 1)
	assert.Equal(t, "provisionSink", trigger.Actions()[0].Name)
}

func Test_OnObjectFinalized_Configuration(t *testing.T) {
	trigger := &OnObjectFinalized{}
	fields := trigger.Configuration()
	require.Len(t, fields, 1)
	assert.Equal(t, "bucket", fields[0].Name)
	assert.False(t, fields[0].Required)
}

func Test_OnObjectFinalized_ExampleData(t *testing.T) {
	trigger := &OnObjectFinalized{}
	data := trigger.ExampleData()
	assert.Equal(t, storageServiceName, data["serviceName"])
	assert.Equal(t, objectsCreateMethod, data["methodName"])
	assert.Contains(t, data["resourceName"], "buckets/my-bucket/objects/")
}

func Test_OnObjectFinalized_SinkFilter(t *testing.T) {
	assert.Contains(t, SinkFilter, "storage.googleapis.com")
	assert.Contains(t, SinkFilter, "storage.objects.create")
}

func Test_OnObjectFinalized_OnIntegrationMessage(t *testing.T) {
	trigger := &OnObjectFinalized{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("wrong service name does not emit", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  "compute.googleapis.com",
				"methodName":   objectsCreateMethod,
				"resourceName": "projects/_/buckets/my-bucket/objects/file.json",
			},
			Logger:        logger,
			Events:        events,
			Configuration: map[string]any{},
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("wrong method name does not emit", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  storageServiceName,
				"methodName":   "storage.objects.delete",
				"resourceName": "projects/_/buckets/my-bucket/objects/file.json",
			},
			Logger:        logger,
			Events:        events,
			Configuration: map[string]any{},
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("object create event emits", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  storageServiceName,
				"methodName":   objectsCreateMethod,
				"resourceName": "projects/_/buckets/my-bucket/objects/path/to/file.json",
			},
			Logger:        logger,
			Events:        events,
			Configuration: map[string]any{},
		})
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, onObjectFinalizedEvent, events.Payloads[0].Type)
	})

	t.Run("bucket filter matches correct bucket", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  storageServiceName,
				"methodName":   objectsCreateMethod,
				"resourceName": "projects/_/buckets/my-bucket/objects/file.json",
			},
			Logger:        logger,
			Events:        events,
			Configuration: map[string]any{"bucket": "my-bucket"},
		})
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
	})

	t.Run("bucket filter rejects non-matching bucket", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  storageServiceName,
				"methodName":   objectsCreateMethod,
				"resourceName": "projects/_/buckets/other-bucket/objects/file.json",
			},
			Logger:        logger,
			Events:        events,
			Configuration: map[string]any{"bucket": "my-bucket"},
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("empty bucket filter matches all buckets", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  storageServiceName,
				"methodName":   objectsCreateMethod,
				"resourceName": "projects/_/buckets/any-bucket/objects/file.json",
			},
			Logger:        logger,
			Events:        events,
			Configuration: map[string]any{},
		})
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
	})
}

func Test_resourceMatchesBucket(t *testing.T) {
	assert.True(t, resourceMatchesBucket("projects/_/buckets/my-bucket/objects/file.json", "my-bucket"))
	assert.False(t, resourceMatchesBucket("projects/_/buckets/other-bucket/objects/file.json", "my-bucket"))
	assert.False(t, resourceMatchesBucket("projects/_/buckets/my-bucket-extra/objects/file.json", "my-bucket"))
	assert.True(t, resourceMatchesBucket("projects/_/buckets/my-bucket/objects/path/to/file.json", "my-bucket"))
}

func Test_sanitizeSinkID(t *testing.T) {
	assert.Equal(t, "abc-123", sanitizeSinkID("ABC-123"))
	assert.Equal(t, "abc-def", sanitizeSinkID("abc-def"))
	assert.Equal(t, "a", sanitizeSinkID("a!@#$%^&*()"))

	long := ""
	for i := 0; i < 100; i++ {
		long += "a"
	}
	assert.LessOrEqual(t, len(sanitizeSinkID(long)), 80)
}
