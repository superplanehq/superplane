package cloudstorage

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestOnObjectFinalized_Metadata(t *testing.T) {
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

func TestOnObjectFinalized_Configuration(t *testing.T) {
	trigger := &OnObjectFinalized{}
	fields := trigger.Configuration()
	require.Len(t, fields, 1)
	assert.Equal(t, "bucket", fields[0].Name)
	assert.False(t, fields[0].Required)
}

func TestOnObjectFinalized_ExampleData(t *testing.T) {
	trigger := &OnObjectFinalized{}
	data := trigger.ExampleData()
	assert.Equal(t, storageServiceName, data["serviceName"])
	assert.Equal(t, objectsCreateMethod, data["methodName"])
	assert.Contains(t, data["resourceName"].(string), "buckets/my-bucket/objects/")
}

func TestOnObjectFinalized_SinkFilter(t *testing.T) {
	assert.Contains(t, SinkFilter, "storage.googleapis.com")
	assert.Contains(t, SinkFilter, "storage.objects.create")
}

func TestOnObjectFinalized_OnIntegrationMessage(t *testing.T) {
	trigger := &OnObjectFinalized{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("wrong service name does not emit", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  "compute.googleapis.com",
				"methodName":   objectsCreateMethod,
				"resourceName": "projects/_/buckets/b/objects/o",
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
				"serviceName":  storageServiceName,
				"methodName":   "storage.objects.delete",
				"resourceName": "projects/_/buckets/b/objects/o",
			},
			Logger: logger,
			Events: events,
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
				"resourceName": "projects/_/buckets/my-bucket/objects/path/to/file.txt",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, EmittedEventType, events.Payloads[0].Type)
	})

	t.Run("bucket filter matches", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  storageServiceName,
				"methodName":   objectsCreateMethod,
				"resourceName": "projects/_/buckets/my-bucket/objects/file.txt",
			},
			Configuration: map[string]any{
				"bucket": "my-bucket",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
	})

	t.Run("bucket filter does not match", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"serviceName":  storageServiceName,
				"methodName":   objectsCreateMethod,
				"resourceName": "projects/_/buckets/other-bucket/objects/file.txt",
			},
			Configuration: map[string]any{
				"bucket": "my-bucket",
			},
			Logger: logger,
			Events: events,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})
}

func Test_resourceNameMatchesBucket(t *testing.T) {
	assert.True(t, resourceNameMatchesBucket("projects/_/buckets/my-bucket/objects/file.txt", "my-bucket"))
	assert.False(t, resourceNameMatchesBucket("projects/_/buckets/other-bucket/objects/file.txt", "my-bucket"))
	assert.False(t, resourceNameMatchesBucket("projects/_/buckets/my-bucket-2/objects/file.txt", "my-bucket"))
}

func Test_cloudstorage_sanitizeSinkID(t *testing.T) {
	assert.Equal(t, "abc-123", sanitizeSinkID("ABC-123"))
	assert.Equal(t, "abc-def", sanitizeSinkID("abc-def"))
	assert.Equal(t, "a", sanitizeSinkID("a!@#$%^&*()"))

	long := ""
	for i := 0; i < 100; i++ {
		long += "a"
	}
	assert.LessOrEqual(t, len(sanitizeSinkID(long)), 80)
}
