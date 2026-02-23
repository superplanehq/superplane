package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Script struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Label          string
	Description    string
	Source         string
	Manifest       datatypes.JSON
	Status         string
	CreatedBy      *uuid.UUID
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

const (
	ScriptStatusDraft  = "draft"
	ScriptStatusActive = "active"
	ScriptStatusError  = "error"
)

func FindScript(orgID, id string) (*Script, error) {
	return FindScriptInTransaction(database.Conn(), orgID, id)
}

func FindScriptInTransaction(tx *gorm.DB, orgID, id string) (*Script, error) {
	var script Script
	err := tx.
		Where("organization_id = ?", orgID).
		Where("id = ?", id).
		First(&script).
		Error

	if err != nil {
		return nil, err
	}

	return &script, nil
}

func FindScriptsByOrganization(orgID string) ([]Script, error) {
	return FindScriptsByOrganizationInTransaction(database.Conn(), orgID)
}

func FindScriptsByOrganizationInTransaction(tx *gorm.DB, orgID string) ([]Script, error) {
	var scripts []Script
	err := tx.
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Find(&scripts).
		Error

	if err != nil {
		return nil, err
	}

	return scripts, nil
}

func FindActiveScripts() ([]Script, error) {
	return FindActiveScriptsInTransaction(database.Conn())
}

func FindActiveScriptsInTransaction(tx *gorm.DB) ([]Script, error) {
	var scripts []Script
	err := tx.
		Where("status = ?", ScriptStatusActive).
		Find(&scripts).
		Error

	if err != nil {
		return nil, err
	}

	return scripts, nil
}

func FindScriptByName(orgID uuid.UUID, name string) (*Script, error) {
	return FindScriptByNameInTransaction(database.Conn(), orgID, name)
}

func FindScriptByNameInTransaction(tx *gorm.DB, orgID uuid.UUID, name string) (*Script, error) {
	var script Script
	err := tx.
		Where("organization_id = ?", orgID).
		Where("name = ?", name).
		First(&script).
		Error

	if err != nil {
		return nil, err
	}

	return &script, nil
}
