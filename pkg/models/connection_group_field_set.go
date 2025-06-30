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
	Timeout           uint32
	TimeoutBehavior   string
	State             string
	Result            string
	CreatedAt         *time.Time
}

func (s *ConnectionGroupFieldSet) String() string {
	return fmt.Sprintf("%s - %v", s.ID.String(), s.FieldSet.Data())
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
	groupSpec := g.Spec.Data()
	now := time.Now()
	fieldSet := &ConnectionGroupFieldSet{
		ConnectionGroupID: g.ID,
		FieldSet:          datatypes.NewJSONType(fields),
		FieldSetHash:      hash,
		State:             ConnectionGroupFieldSetStatePending,
		Timeout:           groupSpec.Timeout,
		TimeoutBehavior:   groupSpec.TimeoutBehavior,
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

func (s *ConnectionGroupFieldSet) UpdateState(tx *gorm.DB, state, result string) error {
	s.State = state
	s.Result = result
	return tx.Save(s).Error
}

func (s *ConnectionGroupFieldSet) IsTimedOut(now time.Time) bool {
	if s.Timeout == 0 {
		return false
	}

	timeout := time.Duration(s.Timeout) * time.Second
	return now.Sub(*s.CreatedAt) > timeout
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

func (s *ConnectionGroupFieldSet) BuildEvent(tx *gorm.DB, result string) ([]byte, error) {
	event := map[string]any{"result": result}

	//
	// Include fields from field set.
	//
	fieldMap := map[string]string{}
	for k, v := range s.FieldSet.Data() {
		fieldMap[k] = v
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
	event["fields"] = fieldMap

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

func (s *ConnectionGroupFieldSet) MissingConnections(tx *gorm.DB, g *ConnectionGroup, allConnections []Connection) ([]Connection, error) {
	connectionsForFieldSet, err := g.FindConnectionsForFieldSet(tx, s)
	if err != nil {
		return nil, fmt.Errorf("error finding connections for field set %s - %v: %v", s.ID.String(), s.FieldSet.Data(), err)
	}

	missing := []Connection{}
	for _, connection := range allConnections {
		if !slices.Contains(connectionsForFieldSet, connection.SourceID.String()) {
			missing = append(missing, connection)
		}
	}

	return missing, nil
}
