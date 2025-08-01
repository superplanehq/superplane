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
