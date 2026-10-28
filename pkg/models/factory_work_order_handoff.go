package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PendingHandOff struct {
	LineID         uuid.UUID `json:"line_id"`
	Column         string    `json:"column"`
	PullRequestURL string    `json:"pull_request_url,omitempty"`
}

func (o *FactoryWorkOrder) SetPendingHandOff(tx *gorm.DB, pending *PendingHandOff) error {
	var value *datatypes.JSONType[PendingHandOff]
	if pending != nil {
		saved := datatypes.NewJSONType(*pending)
		value = &saved
	}
	if err := tx.Model(o).Omit("Assignees").Update("pending_handoff", value).Error; err != nil {
		return err
	}
	o.PendingHandOff = value
	return nil
}
