package models

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ConnectionGroupFieldSet struct {
	ID                uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	ConnectionGroupID uuid.UUID
	FieldSet          datatypes.JSONType[map[string]string]
	FieldSetHash      string
	State             string
	CreatedAt         *time.Time
}

type ConnectionGroupFieldSetEvent struct {
	ID                   uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	ConnectionGroupSetID uuid.UUID
	EventID              uuid.UUID
	SourceID             uuid.UUID
	SourceName           string
	SourceType           string
	ReceivedAt           *time.Time
}

func (g *ConnectionGroup) CreateFieldSet(tx *gorm.DB, fields map[string]string, hash string) (*ConnectionGroupFieldSet, error) {
	now := time.Now()
	fieldSet := &ConnectionGroupFieldSet{
		ConnectionGroupID: g.ID,
		FieldSet:          datatypes.NewJSONType(fields),
		FieldSetHash:      hash,
		State:             ConnectionGroupFieldSetStatePending,
		CreatedAt:         &now,
	}

	err := tx.Create(fieldSet).Error
	if err != nil {
		return nil, err
	}

	return fieldSet, nil
}

func (s *ConnectionGroupFieldSet) FindEvents() ([]ConnectionGroupFieldSetEvent, error) {
	var events []ConnectionGroupFieldSetEvent
	err := database.Conn().
		Where("connection_group_set_id = ?", s.ID).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func (s *ConnectionGroupFieldSet) UpdateState(tx *gorm.DB, state string) error {
	s.State = state
	return tx.Save(s).Error
}

type ConnectionGroupFieldSetEventWithData struct {
	SourceName string
	Raw        datatypes.JSON
}

func (s *ConnectionGroupFieldSet) FindEventsWithData(tx *gorm.DB) ([]ConnectionGroupFieldSetEventWithData, error) {
	var events []ConnectionGroupFieldSetEventWithData
	err := tx.
		Table("connection_group_field_set_events AS e").
		Joins("JOIN events AS ev ON ev.id = e.event_id").
		Select("e.source_name, ev.raw").
		Where("e.connection_group_set_id = ?", s.ID).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func (s *ConnectionGroupFieldSet) BuildEvent(tx *gorm.DB) ([]byte, error) {
	event := map[string]any{}

	//
	// Include fields from field set.
	//
	for k, v := range s.FieldSet.Data() {
		event[k] = v
	}

	//
	// Include events from connections.
	//
	events, err := s.FindEventsWithData(tx)
	if err != nil {
		return nil, err
	}

	eventMap := map[string]any{}
	for _, e := range events {
		var obj map[string]any
		err := json.Unmarshal(e.Raw, &obj)
		if err != nil {
			return nil, err
		}

		eventMap[e.SourceName] = obj
	}

	event["events"] = eventMap
	return json.Marshal(event)
}

func (s *ConnectionGroupFieldSet) AttachEvent(tx *gorm.DB, event *Event) (*ConnectionGroupFieldSetEvent, error) {
	now := time.Now()
	ID := uuid.New()

	connectionGroupEvent := ConnectionGroupFieldSetEvent{
		ID:                   ID,
		ConnectionGroupSetID: s.ID,
		EventID:              event.ID,
		SourceID:             event.SourceID,
		SourceName:           event.SourceName,
		SourceType:           event.SourceType,
		ReceivedAt:           &now,
	}

	err := tx.Create(&connectionGroupEvent).Error
	if err != nil {
		return nil, err
	}

	return &connectionGroupEvent, nil
}

func (s *ConnectionGroupFieldSet) MissingConnections(tx *gorm.DB, g *ConnectionGroup, allConnections []string) ([]string, error) {
	connectionsForFieldSet, err := g.FindConnectionsForFieldSet(tx, s.FieldSetHash)
	if err != nil {
		return nil, fmt.Errorf("error finding connections for field set %v: %v", s.FieldSet.Data(), err)
	}

	missing := []string{}
	for _, connID := range allConnections {
		if !slices.Contains(connectionsForFieldSet, connID) {
			missing = append(missing, connID)
		}
	}

	return missing, nil
}
