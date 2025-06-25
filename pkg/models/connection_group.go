package models

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ConnectionGroupEmitOnAll      = "all"
	ConnectionGroupEmitOnMajority = "majority"

	ConnectionGroupFieldSetStatePending   = "pending"
	ConnectionGroupFieldSetStateProcessed = "processed"
)

type ConnectionGroup struct {
	ID        uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Name      string
	CanvasID  uuid.UUID
	Spec      datatypes.JSONType[ConnectionGroupSpec]
	CreatedAt *time.Time
	CreatedBy uuid.UUID
	UpdatedAt *time.Time
	UpdatedBy uuid.UUID
}

func (g *ConnectionGroup) CalculateFieldSet(event *Event) (map[string]string, string, error) {
	fieldSet := map[string]string{}
	for _, fieldDef := range g.Spec.Data().GroupBy.Fields {
		value, err := event.EvaluateStringExpression(fieldDef.Expression)
		if err != nil {
			return nil, "", fmt.Errorf("error evaluating expression '%s' for connection group field %s: %v", fieldDef.Expression, fieldDef.Name, err)
		}

		fieldSet[fieldDef.Name] = value
	}

	hash, err := hashMap(fieldSet)
	if err != nil {
		return nil, "", err
	}

	return fieldSet, hash, nil
}

// TODO: move this to another package,
func hashMap(m map[string]string) (string, error) {
	//
	// Maps are not ordered, so we need to sort the key/value
	// pairs before hashing it. We do that by creating an array of key=value
	// pairs and sorting it.
	//
	var keyValues []string
	for k, v := range m {
		keyValues = append(keyValues, fmt.Sprintf("%s=%s", k, v))
	}

	sort.Strings(keyValues)

	//
	// Now, we join our list of key/value pairs and hash it.
	//
	h := sha256.New()
	_, err := h.Write([]byte(strings.Join(keyValues, ",")))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (g *ConnectionGroup) ListFieldSets() ([]ConnectionGroupFieldSet, error) {
	return g.ListFieldSetsInTransaction(database.Conn())
}

func (g *ConnectionGroup) ListFieldSetsInTransaction(tx *gorm.DB) ([]ConnectionGroupFieldSet, error) {
	var fieldSets []ConnectionGroupFieldSet
	err := tx.
		Where("connection_group_id = ?", g.ID).
		Order("created_at DESC").
		Find(&fieldSets).
		Error

	if err != nil {
		return nil, err
	}

	return fieldSets, nil
}

type ConnectionGroupSpec struct {
	GroupBy *ConnectionGroupBySpec `json:"group_by"`
}

type ConnectionGroupBySpec struct {
	Fields []ConnectionGroupByField `json:"fields"`
	EmitOn string                   `json:"emit_on"`
}

type ConnectionGroupByField struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

type ConnectionGroupFieldSet struct {
	ID                uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	ConnectionGroupID uuid.UUID
	FieldSet          datatypes.JSONType[map[string]string]
	FieldSetHash      string
	State             string
	CreatedAt         *time.Time
}

func CreateConnectionGroupFieldSet(tx *gorm.DB, connectionGroupID uuid.UUID, fields map[string]string, hash string) (*ConnectionGroupFieldSet, error) {
	now := time.Now()
	fieldSet := &ConnectionGroupFieldSet{
		ConnectionGroupID: connectionGroupID,
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

func FindConnectionGroupFieldSetByHash(tx *gorm.DB, connectionGroupID uuid.UUID, hash string) (*ConnectionGroupFieldSet, error) {
	var fieldSet *ConnectionGroupFieldSet
	err := tx.
		Where("connection_group_id = ?", connectionGroupID).
		Where("field_set_hash = ?", hash).
		First(&fieldSet).
		Error

	if err != nil {
		return nil, err
	}

	return fieldSet, nil
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

func CreateConnectionGroupFieldSetEvent(tx *gorm.DB, setID uuid.UUID, event *Event) (*ConnectionGroupFieldSetEvent, error) {
	now := time.Now()
	ID := uuid.New()

	connectionGroupEvent := ConnectionGroupFieldSetEvent{
		ID:                   ID,
		ConnectionGroupSetID: setID,
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

func (c *Canvas) CreateConnectionGroup(
	name, createdBy string,
	connections []Connection,
	spec ConnectionGroupSpec,
) (*ConnectionGroup, error) {
	now := time.Now()
	ID := uuid.New()

	var connectionGroup *ConnectionGroup

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		connectionGroup = &ConnectionGroup{
			ID:        ID,
			CanvasID:  c.ID,
			Name:      name,
			CreatedAt: &now,
			CreatedBy: uuid.Must(uuid.Parse(createdBy)),
			Spec:      datatypes.NewJSONType(spec),
		}

		err := tx.Clauses(clause.Returning{}).Create(&connectionGroup).Error
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				return ErrNameAlreadyUsed
			}

			return err
		}

		for _, i := range connections {
			c := i
			c.TargetID = ID
			c.TargetType = ConnectionTargetTypeConnectionGroup
			err := tx.Create(&c).Error
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return connectionGroup, nil
}

func FindConnectionGroupByID(tx *gorm.DB, id uuid.UUID) (*ConnectionGroup, error) {
	var connectionGroup *ConnectionGroup
	err := tx.First(&connectionGroup, id).Error
	if err != nil {
		return nil, err
	}

	return connectionGroup, nil
}

func FindConnectionsWithFieldSetHash(tx *gorm.DB, groupID uuid.UUID, hash string) ([]string, error) {
	var connections []string
	err := tx.
		Table("connection_group_field_set_events AS e").
		Joins("JOIN connection_group_field_sets AS f ON f.id = e.connection_group_set_id").
		Select("e.source_id").
		Where("f.connection_group_id = ?", groupID).
		Where("f.field_set_hash = ?", hash).
		Find(&connections).
		Error

	if err != nil {
		return nil, err
	}

	return connections, nil
}
