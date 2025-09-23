package workers

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type ResourceCleanupService struct {
	Registry *registry.Registry
}

func NewResourceCleanupService(registry *registry.Registry) *ResourceCleanupService {
	return &ResourceCleanupService{
		Registry: registry,
	}
}

func (s *ResourceCleanupService) CleanupUnusedResource(oldResourceID, excludeStageID uuid.UUID) error {
	oldResource, err := models.FindResourceByID(oldResourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warningf("Old resource %s not found, skipping cleanup", oldResourceID)
			return nil
		}
		log.Errorf("Error finding old resource %s: %v", oldResourceID, err)
		return err
	}

	logger := log.WithField("old_resource_id", oldResourceID)

	externalEventSourceCount, err := models.CountExternalEventSourcesUsingResource(oldResourceID)
	if err != nil {
		logger.Errorf("Error counting external event sources using resource: %v", err)
		return err
	}

	stagesCount, err := models.CountOtherStagesUsingResource(oldResourceID, excludeStageID)
	if err != nil {
		logger.Errorf("Error counting stages using resource: %v", err)
		return err
	}

	totalUsages := externalEventSourceCount + stagesCount
	if totalUsages > 0 {
		logger.Infof("Resource is used by %d external event sources and %d stages, skipping cleanup",
			externalEventSourceCount, stagesCount)
		return nil
	}

	integration, err := models.FindIntegrationByID(oldResource.IntegrationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Info("Integration not found, skipping cleanup")
			return nil
		}
		logger.Errorf("Error finding integration: %v", err)
		return err
	}

	resourceManager, err := s.Registry.NewResourceManager(context.Background(), integration)
	if err != nil {
		logger.Errorf("Error creating resource manager: %v", err)
		return err
	}

	childResources, err := oldResource.FindChildren()
	if err != nil {
		logger.Errorf("Error finding child resources: %v", err)
		return err
	}

	for _, childResource := range childResources {
		logger.Infof("Cleaning up webhook resource %s", childResource.Id())
		err = resourceManager.CleanupWebhook(oldResource, &childResource)
		if err != nil {
			logger.Errorf("Error cleaning up webhook: %v", err)
		} else {
			logger.Infof("Successfully cleaned up webhook %s", childResource.Id())
		}
	}

	err = models.DeleteResourceWithChildren(oldResourceID)
	if err != nil {
		logger.Errorf("Error deleting old resource and children from database: %v", err)
		return err
	}

	logger.Infof("Successfully cleaned up and deleted old resource %s and %d child resources", oldResourceID, len(childResources))

	return nil
}

func (s *ResourceCleanupService) CleanupUnusedResourceWithModel(oldResource *models.Resource, excludeStageID uuid.UUID) error {
	logger := log.WithField("old_resource_id", oldResource.ID)

	externalEventSourceCount, err := models.CountExternalEventSourcesUsingResource(oldResource.ID)
	if err != nil {
		logger.Errorf("Error counting external event sources using resource: %v", err)
		return err
	}

	stagesCount, err := models.CountOtherStagesUsingResource(oldResource.ID, excludeStageID)
	if err != nil {
		logger.Errorf("Error counting stages using resource: %v", err)
		return err
	}

	totalUsages := externalEventSourceCount + stagesCount
	if totalUsages > 0 {
		logger.Infof("Resource is used by %d external event sources and %d stages, skipping cleanup",
			externalEventSourceCount, stagesCount)
		return nil
	}

	err = models.DeleteResourceWithChildren(oldResource.ID)
	if err != nil {
		logger.Errorf("Error deleting old resource and children from database: %v", err)
		return err
	}

	logger.Infof("Successfully deleted old resource %s from database", oldResource.ID)

	return nil
}

func (s *ResourceCleanupService) CleanupUnusedResourceWithModelInTransaction(tx *gorm.DB, oldResource *models.Resource, excludeStageID uuid.UUID) error {
	logger := log.WithField("old_resource_id", oldResource.ID)

	externalEventSourceCount, err := models.CountExternalEventSourcesUsingResource(oldResource.ID)
	if err != nil {
		logger.Errorf("Error counting external event sources using resource: %v", err)
		return err
	}

	stagesCount, err := models.CountOtherStagesUsingResource(oldResource.ID, excludeStageID)
	if err != nil {
		logger.Errorf("Error counting stages using resource: %v", err)
		return err
	}

	totalUsages := externalEventSourceCount + stagesCount
	if totalUsages > 0 {
		logger.Infof("Resource is used by %d external event sources and %d stages, skipping cleanup",
			externalEventSourceCount, stagesCount)
		return nil
	}

	err = models.DeleteResourceWithChildrenInTransaction(tx, oldResource.ID)
	if err != nil {
		logger.Errorf("Error deleting old resource and children from database: %v", err)
		return err
	}

	logger.Infof("Successfully deleted old resource %s from database", oldResource.ID)

	return nil
}

func (s *ResourceCleanupService) CleanupStageWebhooks(stage *models.Stage) error {
	if stage.ResourceID == nil {
		return nil
	}

	// Use the existing resource cleanup service to handle webhook cleanup
	// We pass uuid.Nil as excludeStageID since we're deleting the stage itself
	return s.CleanupUnusedResource(*stage.ResourceID, uuid.Nil)
}

func (s *ResourceCleanupService) CleanupEventSourceWebhooks(eventSource *models.EventSource, resource *models.Resource) (bool, error) {
	if eventSource.ResourceID == nil {
		return false, nil
	}

	if resource == nil {
		return false, nil
	}

	logger := log.WithField("event_source_id", eventSource.ID)

	count, err := models.CountExternalEventSourcesUsingResource(*eventSource.ResourceID)
	if err != nil {
		return false, err
	}

	// Do not clean up if there are other event sources using this resource
	if count > 1 {
		return false, nil
	}

	integration, err := models.FindIntegrationByID(resource.IntegrationID)
	if err != nil {
		return false, err
	}

	resourceManager, err := s.Registry.NewResourceManager(context.Background(), integration)
	if err != nil {
		return false, err
	}

	childResources, err := resource.FindChildren()
	if err != nil {
		return false, err
	}

	for _, childResource := range childResources {
		logger.Infof("Cleaning up webhook resource %s", childResource.Id())
		err = resourceManager.CleanupWebhook(resource, &childResource)
		if err != nil {
			logger.Errorf("Error cleaning up webhook: %v", err)
		} else {
			logger.Infof("Successfully cleaned up webhook %s", childResource.Id())
		}
	}

	return true, nil
}
