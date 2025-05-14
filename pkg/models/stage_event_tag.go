package models

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	TagStateUnknown   = "unknown"
	TagStateHealthy   = "healthy"
	TagStateUnhealthy = "unhealthy"
)

type StageEventTag struct {
	Name         string
	Value        string
	StageEventID uuid.UUID
	State        string
}

func CreateStageEventTag(name, value string, stageEventID uuid.UUID) error {
	return CreateStageEventTagInTransaction(database.Conn(), name, value, stageEventID)
}

func CreateStageEventTagInTransaction(tx *gorm.DB, name, value string, stageEventID uuid.UUID) error {
	v := StageEventTag{
		Name:         name,
		Value:        value,
		StageEventID: stageEventID,
		State:        TagStateUnknown,
	}

	return tx.Create(&v).Error
}

type StageTag struct {
	StageID    uuid.UUID
	EventState string
	TagName    string
	TagValue   string
	TagState   string
}

func UpdateTagState(name, value, state string) error {
	return UpdateTagStateInTransaction(database.Conn(), name, value, state)
}

func UpdateTagStateInTransaction(tx *gorm.DB, name, value, state string) error {
	return tx.
		Table("stage_event_tags").
		Where("name = ?", name).
		Where("value = ?", value).
		Update("state", state).
		Error
}

func UpdateStageEventTagStateInBulk(tx *gorm.DB, stageEventID uuid.UUID, state string, tags map[string]string) error {
	records := []StageEventTag{}
	for tagName, tagValue := range tags {
		records = append(records, StageEventTag{
			Name:         tagName,
			Value:        tagValue,
			State:        state,
			StageEventID: stageEventID,
		})
	}

	return tx.
		Clauses(clause.OnConflict{
			OnConstraint: "stage_event_tags_pkey",
			UpdateAll:    true,
		}).
		Create(&records).
		Error
}

func ListStageTags(name, value string, states []string, stageID, stageEventID string) ([]StageTag, error) {
	var values []StageTag

	query := database.Conn().
		Table("stage_event_tags AS t").
		Joins("INNER JOIN stage_events AS e ON e.id = t.stage_event_id").
		Select("e.stage_id, t.name as tag_name, t.value as tag_value, e.state as event_state, t.state as tag_state")

	if name != "" {
		query = query.Where("t.name = ?", name)
	}

	if value != "" {
		query = query.Where("t.value = ?", value)
	}

	if len(states) > 0 {
		query = query.Where("t.state IN ?", states)
	}

	if stageID != "" {
		query = query.Where("e.stage_id = ?", stageID)
	}

	if stageEventID != "" {
		query = query.Where("t.stage_event_id = ?", stageEventID)
	}

	err := query.Find(&values).Error
	if err != nil {
		return nil, err
	}

	return values, nil
}
