package productanalytics

import (
	"testing"

	"github.com/google/uuid"
	posthog "github.com/posthog/posthog-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingQueue struct {
	messages []posthog.Message
	closed   bool
	err      error
}

func (q *recordingQueue) Enqueue(message posthog.Message) error {
	q.messages = append(q.messages, message)
	return q.err
}

func (q *recordingQueue) Close() error {
	q.closed = true
	return q.err
}

func TestClientCaptureAddsFactoryEnvelope(t *testing.T) {
	queue := &recordingQueue{}
	client := newClient(queue)
	eventID := uuid.New()

	client.Capture(Event{
		Name:           EventFactoryTaskCreated,
		EventID:        eventID,
		DistinctID:     "account-123",
		OrganizationID: "organization-123",
		FactoryID:      "factory-123",
		TaskID:         "task-123",
		ActorType:      ActorUser,
		Source:         SourceUI,
		Properties: Properties{
			"creation_source": "manual",
			"organization_id": "wrong-organization",
		},
	})

	require.Len(t, queue.messages, 1)
	capture, ok := queue.messages[0].(posthog.Capture)
	require.True(t, ok)
	assert.Equal(t, string(EventFactoryTaskCreated), capture.Event)
	assert.Equal(t, eventID.String(), capture.Uuid)
	assert.Equal(t, "account-123", capture.DistinctId)
	assert.Equal(t, "organization-123", capture.Groups[OrganizationGroup])
	assert.Equal(t, 1, capture.Properties["schema_version"])
	assert.Equal(t, "organization-123", capture.Properties["organization_id"])
	assert.Equal(t, "factory-123", capture.Properties["factory_id"])
	assert.Equal(t, "task-123", capture.Properties["task_id"])
	assert.Equal(t, string(ActorUser), capture.Properties["actor_type"])
	assert.Equal(t, string(SourceUI), capture.Properties["source"])
	assert.Equal(t, "manual", capture.Properties["creation_source"])
}

func TestClientCaptureRejectsIncompleteEvents(t *testing.T) {
	queue := &recordingQueue{}
	client := newClient(queue)

	client.Capture(Event{Name: EventFactoryTaskCreated})

	assert.Empty(t, queue.messages)
}

func TestNoopClientDoesNotEnableAnalytics(t *testing.T) {
	client, err := New("")
	require.NoError(t, err)

	assert.False(t, client.Enabled())
	client.Capture(Event{Name: EventFactoryTaskCreated})
	assert.NoError(t, client.Close())
}

func TestClientCloseFlushesQueue(t *testing.T) {
	queue := &recordingQueue{}
	client := newClient(queue)

	require.NoError(t, client.Close())
	assert.True(t, queue.closed)
}

func TestCloseCurrentFlushesInstalledClientOnce(t *testing.T) {
	queue := &recordingQueue{}
	previous := SetCurrent(newClient(queue))
	t.Cleanup(func() {
		SetCurrent(previous)
	})

	require.NoError(t, CloseCurrent())
	require.NoError(t, CloseCurrent())

	assert.True(t, queue.closed)
	assert.False(t, Enabled())
}

func TestClientCaptureLogsEnqueueFailure(t *testing.T) {
	queue := &recordingQueue{err: assert.AnError}
	client := newClient(queue)

	client.Capture(Event{
		Name:           EventFactoryTaskMerged,
		EventID:        uuid.New(),
		DistinctID:     "account-123",
		OrganizationID: "organization-123",
		ActorType:      ActorExternal,
		Source:         SourceGitHubWebhook,
	})

	require.Len(t, queue.messages, 1)
}

func TestClientCaptureRejectsUnknownEvent(t *testing.T) {
	queue := &recordingQueue{}
	client := newClient(queue)

	client.Capture(Event{
		Name:           "canvas:node_add",
		EventID:        uuid.New(),
		DistinctID:     "account-123",
		OrganizationID: "organization-123",
		ActorType:      ActorUser,
		Source:         SourceUI,
	})

	assert.Empty(t, queue.messages)
}

func TestStableEventIDUsesAllParts(t *testing.T) {
	first := StableEventID(EventFactoryTaskStarted, "task-123", "run-456")
	same := StableEventID(EventFactoryTaskStarted, "task-123", "run-456")
	different := StableEventID(EventFactoryTaskStarted, "task-123", "run-789")

	assert.Equal(t, first, same)
	assert.NotEqual(t, first, different)
}
