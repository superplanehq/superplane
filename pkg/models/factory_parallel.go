package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const DefaultFactoryMaxParallelTasks = 50

var ErrMaxParallelFactoryTasksInvalid = errors.New("maximum parallel factory tasks must be at least 1")

func SetInstallationMaxParallelFactoryTasks(tx *gorm.DB, max int) error {
	if max < 1 {
		return ErrMaxParallelFactoryTasksInvalid
	}
	if _, err := findOrCreateInstallationMetadata(tx); err != nil {
		return err
	}

	return tx.Model(&InstallationMetadata{}).
		Where("id = ?", installationMetadataID).
		Updates(map[string]any{
			"max_parallel_factory_tasks": max,
			"updated_at":                 time.Now(),
		}).Error
}

func SetOrganizationMaxParallelFactoryTasks(tx *gorm.DB, orgID uuid.UUID, max *int) error {
	if max != nil && *max < 1 {
		return ErrMaxParallelFactoryTasksInvalid
	}

	var value any
	if max != nil {
		value = *max
	}

	return tx.Model(&Organization{}).
		Where("id = ?", orgID).
		Updates(map[string]any{
			"max_parallel_factory_tasks": value,
			"updated_at":                 time.Now(),
		}).Error
}

func ResolveOrganizationFactoryMaxParallelTasks(tx *gorm.DB, orgID uuid.UUID) (int, error) {
	org, err := FindOrganizationByIDInTransaction(tx, orgID.String())
	if err != nil {
		return 0, err
	}
	if org.MaxParallelFactoryTasks != nil {
		return *org.MaxParallelFactoryTasks, nil
	}

	installation, err := GetInstallationMetadata(tx)
	if err != nil {
		return 0, err
	}
	if installation.MaxParallelFactoryTasks < 1 {
		return DefaultFactoryMaxParallelTasks, nil
	}
	return installation.MaxParallelFactoryTasks, nil
}

func lockFactoryForAdmission(tx *gorm.DB, factoryID uuid.UUID) (*Factory, error) {
	var factory Factory
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", factoryID).
		First(&factory).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryNotFound
		}
		return nil, err
	}
	return &factory, nil
}

func lockFactoryAndLineForAdmission(tx *gorm.DB, lineID uuid.UUID) (*FactoryLine, error) {
	var line FactoryLine
	err := tx.Where("id = ?", lineID).First(&line).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryLineNotFound
		}
		return nil, err
	}

	if _, err := lockFactoryForAdmission(tx, line.FactoryID); err != nil {
		return nil, err
	}

	return lockFactoryLineForStepAdmission(tx, lineID)
}

func countActiveFactoryExecutions(tx *gorm.DB, factoryID uuid.UUID) (int64, error) {
	var count int64
	err := tx.
		Model(&FactoryWorkOrderExecution{}).
		Where("factory_id = ?", factoryID).
		Where("status IN ?", []string{
			FactoryWorkOrderExecutionStatusPending,
			FactoryWorkOrderExecutionStatusRunning,
		}).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func factoryAtParallelCapacity(tx *gorm.DB, orgID, factoryID uuid.UUID) (bool, error) {
	max, err := ResolveOrganizationFactoryMaxParallelTasks(tx, orgID)
	if err != nil {
		return false, err
	}
	active, err := countActiveFactoryExecutions(tx, factoryID)
	if err != nil {
		return false, err
	}
	return active >= int64(max), nil
}
