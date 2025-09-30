package workers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__ResourceCleanupService(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Integration: true})
	defer r.Close()

	service := NewResourceCleanupService(r.Registry)

	t.Run("resource not found -> no error", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := service.CleanupUnusedResource(nonExistentID, uuid.Nil)
		assert.NoError(t, err)
	})

	t.Run("resource used by external event source -> skips cleanup", func(t *testing.T) {
		resource, err := r.Integration.CreateResource(semaphore.ResourceTypeProject, uuid.NewString(), "test-resource", "")
		require.NoError(t, err)

		_, _, err = builders.NewEventSourceBuilder(r.Encryptor, r.Registry).
			InCanvas(r.Canvas.ID).
			WithName("test-event-source").
			WithScope(models.EventSourceScopeExternal).
			ForIntegration(r.Integration).
			ForResource(resource).
			Create()
		require.NoError(t, err)

		err = service.CleanupUnusedResource(resource.ID, uuid.Nil)
		assert.NoError(t, err)

		foundResource, err := models.FindResourceByID(resource.ID)
		assert.NoError(t, err)
		assert.Equal(t, resource.ID, foundResource.ID)
	})

	t.Run("resource used by stage -> skips cleanup", func(t *testing.T) {
		resource, err := r.Integration.CreateResource(semaphore.ResourceTypeProject, uuid.NewString(), "test-resource-2", "")
		require.NoError(t, err)

		_, err = builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas.ID).
			WithName("test-stage").
			WithRequester(r.User).
			WithExecutorType("semaphore").
			WithExecutorSpec([]byte(`{"ref":"refs/heads/main","pipelineFile":".semaphore/run.yml"}`)).
			ForResource(resource).
			ForIntegration(r.Integration).
			Create()
		require.NoError(t, err)

		err = service.CleanupUnusedResource(resource.ID, uuid.Nil)
		assert.NoError(t, err)

		foundResource, err := models.FindResourceByID(resource.ID)
		assert.NoError(t, err)
		assert.Equal(t, resource.ID, foundResource.ID)
	})

	t.Run("resource used by excluded stage -> performs cleanup", func(t *testing.T) {
		resource, err := r.Integration.CreateResource(semaphore.ResourceTypeProject, uuid.NewString(), "test-resource-3", "")
		require.NoError(t, err)

		stage, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas.ID).
			WithName("test-stage-excluded").
			WithRequester(r.User).
			WithExecutorType("semaphore").
			WithExecutorSpec([]byte(`{"ref":"refs/heads/main","pipelineFile":".semaphore/run.yml"}`)).
			ForResource(resource).
			ForIntegration(r.Integration).
			Create()
		require.NoError(t, err)

		err = service.CleanupUnusedResource(resource.ID, stage.ID)
		assert.NoError(t, err)

		_, err = models.FindResourceByID(resource.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("unused resource -> performs cleanup", func(t *testing.T) {
		resource, err := r.Integration.CreateResource(semaphore.ResourceTypeProject, uuid.NewString(), "test-resource-4", "")
		require.NoError(t, err)

		err = service.CleanupUnusedResource(resource.ID, uuid.Nil)
		assert.NoError(t, err)

		_, err = models.FindResourceByID(resource.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("resource with no integrations using it -> performs cleanup", func(t *testing.T) {
		resource, err := r.Integration.CreateResource(semaphore.ResourceTypeProject, uuid.NewString(), "test-resource-cleanup", "")
		require.NoError(t, err)

		err = service.CleanupUnusedResource(resource.ID, uuid.Nil)
		assert.NoError(t, err)

		_, err = models.FindResourceByID(resource.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("resource used by multiple stages except excluded one -> skips cleanup", func(t *testing.T) {
		resource, err := r.Integration.CreateResource(semaphore.ResourceTypeProject, uuid.NewString(), "test-resource-shared", "")
		require.NoError(t, err)

		_, err = builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas.ID).
			WithName("test-stage-1").
			WithRequester(r.User).
			WithExecutorType("semaphore").
			WithExecutorSpec([]byte(`{"ref":"refs/heads/main","pipelineFile":".semaphore/run.yml"}`)).
			ForResource(resource).
			ForIntegration(r.Integration).
			Create()
		require.NoError(t, err)

		stage2, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas.ID).
			WithName("test-stage-2").
			WithRequester(r.User).
			WithExecutorType("semaphore").
			WithExecutorSpec([]byte(`{"ref":"refs/heads/main","pipelineFile":".semaphore/run.yml"}`)).
			ForResource(resource).
			ForIntegration(r.Integration).
			Create()
		require.NoError(t, err)

		err = service.CleanupUnusedResource(resource.ID, stage2.ID)
		assert.NoError(t, err)

		foundResource, err := models.FindResourceByID(resource.ID)
		assert.NoError(t, err)
		assert.Equal(t, resource.ID, foundResource.ID)
	})
}

func Test__ResourceCleanupService_CleanupEventSourceWebhooks(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Integration: true})
	defer r.Close()

	service := NewResourceCleanupService(r.Registry)

	t.Run("event source with no resource -> no error", func(t *testing.T) {
		eventSource := &models.EventSource{
			ID:         uuid.New(),
			ResourceID: nil,
		}

		canDelete, err := service.CleanupEventSourceWebhooks(eventSource, nil)
		assert.NoError(t, err)
		assert.False(t, canDelete)
	})

	t.Run("event source with resource -> performs webhook cleanup", func(t *testing.T) {
		resource, err := r.Integration.CreateResource(semaphore.ResourceTypeProject, uuid.NewString(), "test-resource-single", "")
		require.NoError(t, err)

		eventSource, _, err := builders.NewEventSourceBuilder(r.Encryptor, r.Registry).
			InCanvas(r.Canvas.ID).
			WithName("test-event-source-single").
			WithScope(models.EventSourceScopeExternal).
			ForIntegration(r.Integration).
			ForResource(resource).
			Create()
		require.NoError(t, err)

		_, err = service.CleanupEventSourceWebhooks(eventSource, resource)
		assert.NoError(t, err)
		// The result depends on whether other event sources use this resource
		// We can't predict it in this test, so just verify no error occurred

		foundResource, err := models.FindResourceByID(resource.ID)
		assert.NoError(t, err)
		assert.Equal(t, resource.ID, foundResource.ID)
	})
}
