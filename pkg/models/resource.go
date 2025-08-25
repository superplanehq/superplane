package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type Resource struct {
	ID            uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	IntegrationID uuid.UUID
	ParentID      *uuid.UUID
	ExternalID    string
	ResourceName  string `gorm:"column:name"`
	ResourceType  string `gorm:"column:type"`
	CreatedAt     *time.Time
}

func (r *Resource) Id() string {
	return r.ExternalID
}

func (r *Resource) Name() string {
	return r.ResourceName
}

func (r *Resource) Type() string {
	return r.ResourceType
}

func (r *Resource) ListEventSources() ([]EventSource, error) {
	var eventSources []EventSource
	err := database.Conn().
		Where("resource_id = ?", r.ID).
		Find(&eventSources).
		Error

	if err != nil {
		return nil, err
	}

	return eventSources, nil
}

func (r *Resource) FindEventSource() (*EventSource, error) {
	return r.FindEventSourceInTransaction(database.Conn())
}

func (r *Resource) FindEventSourceInTransaction(tx *gorm.DB) (*EventSource, error) {
	var eventSource EventSource
	err := tx.
		Where("resource_id = ?", r.ID).
		First(&eventSource).
		Error

	if err != nil {
		return nil, err
	}

	return &eventSource, nil
}

func (r *Resource) FindChildren() ([]Resource, error) {
	var resources []Resource
	err := database.Conn().
		Where("parent_id = ?", r.ID).
		Find(&resources).
		Error

	if err != nil {
		return nil, err
	}

	return resources, nil
}

func FindResourceByID(id uuid.UUID) (*Resource, error) {
	return FindResourceByIDInTransaction(database.Conn(), id)
}

func FindResourceByIDInTransaction(tx *gorm.DB, id uuid.UUID) (*Resource, error) {
	var resource Resource

	err := tx.
		Where("id = ?", id).
		First(&resource).
		Error

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

func FindResource(integrationID uuid.UUID, resourceType, name string) (*Resource, error) {
	return FindResourceInTransaction(database.Conn(), integrationID, resourceType, name)
}

func FindResourceInTransaction(tx *gorm.DB, integrationID uuid.UUID, resourceType, name string) (*Resource, error) {
	var resource Resource

	err := tx.
		Where("integration_id = ?", integrationID).
		Where("type = ?", resourceType).
		Where("name = ?", name).
		First(&resource).
		Error

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

func CountOtherEventSourcesUsingResource(resourceID, excludeEventSourceID uuid.UUID) (int64, error) {
	var count int64

	// Count other event sources using this resource directly OR using any child of this resource
	err := database.Conn().
		Model(&EventSource{}).
		Where("resource_id = ? OR resource_id IN (SELECT id FROM resources WHERE parent_id = ?)", resourceID, resourceID).
		Where("id != ?", excludeEventSourceID).
		Count(&count).
		Error

	return count, err
}

func CountEventSourcesUsingResource(resourceID uuid.UUID) (int64, error) {
	var count int64

	// Count event sources using this resource directly OR using any child of this resource
	err := database.Conn().
		Model(&EventSource{}).
		Where("resource_id = ? OR resource_id IN (SELECT id FROM resources WHERE parent_id = ?)", resourceID, resourceID).
		Count(&count).
		Error

	return count, err
}

func CountStagesUsingResource(resourceID uuid.UUID) (int64, error) {
	var count int64

	// Count stages using this resource directly OR using any child of this resource
	err := database.Conn().
		Model(&Stage{}).
		Where("resource_id = ? OR resource_id IN (SELECT id FROM resources WHERE parent_id = ?)", resourceID, resourceID).
		Count(&count).
		Error

	return count, err
}

func CountOtherStagesUsingResource(resourceID, excludeStageID uuid.UUID) (int64, error) {
	var count int64

	// Count other stages using this resource directly OR using any child of this resource
	err := database.Conn().
		Model(&Stage{}).
		Where("resource_id = ? OR resource_id IN (SELECT id FROM resources WHERE parent_id = ?)", resourceID, resourceID).
		Where("id != ?", excludeStageID).
		Count(&count).
		Error

	return count, err
}

// DeleteResourceWithChildren deletes a resource and all its child resources in a transaction
func DeleteResourceWithChildren(resourceID uuid.UUID) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		// Find the resource to delete
		var resource Resource
		err := tx.Where("id = ?", resourceID).First(&resource).Error
		if err != nil {
			return err
		}

		// Find all child resources
		var childResources []Resource
		err = tx.Where("parent_id = ?", resourceID).Find(&childResources).Error
		if err != nil {
			return err
		}

		// Delete child resources first (to respect foreign key constraints)
		if len(childResources) > 0 {
			err = tx.Delete(&childResources).Error
			if err != nil {
				return err
			}
		}

		// Delete the parent resource
		err = tx.Delete(&resource).Error
		if err != nil {
			return err
		}

		return nil
	})
}
