package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ConnectionGroupFieldSetStatePending   = "pending"
	ConnectionGroupFieldSetStateProcessed = "processed"
	ConnectionGroupFieldSetStateDiscarded = "dicarded"

	ConnectionGroupFieldSetStateReasonTimeout = "timeout"
	ConnectionGroupFieldSetStateReasonOK      = "ok"

	ConnectionGroupTimeoutBehaviorNone = "none"
	ConnectionGroupTimeoutBehaviorEmit = "emit"
	ConnectionGroupTimeoutBehaviorDrop = "drop"

	MinConnectionGroupTimeout = 60    // 1 minute
	MaxConnectionGroupTimeout = 86400 // 1 day
)

type ConnectionGroup struct {
	ID          uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Name        string
	Description string
	CanvasID    uuid.UUID
	Spec        datatypes.JSONType[ConnectionGroupSpec]
	CreatedAt   *time.Time
	CreatedBy   uuid.UUID
	UpdatedAt   *time.Time
	UpdatedBy   uuid.UUID
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

	hash, err := crypto.SHA256ForMap(fieldSet)
	if err != nil {
		return nil, "", err
	}

	return fieldSet, hash, nil
}

func (g *ConnectionGroup) Emit(fieldSet *ConnectionGroupFieldSet, stateReason string, missingConnections []Connection) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		return g.EmitInTransaction(tx, fieldSet, stateReason, missingConnections)
	})
}

func (g *ConnectionGroup) EmitInTransaction(tx *gorm.DB, fieldSet *ConnectionGroupFieldSet, stateReason string, missingConnections []Connection) error {
	event, err := fieldSet.BuildEvent(tx, stateReason, missingConnections)
	if err != nil {
		return fmt.Errorf("error building connection group event: %v", err)
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshaling connection group event: %v", err)
	}

	_, err = CreateEventInTransaction(tx, g.ID, g.Name, SourceTypeConnectionGroup, event.Type, eventData, []byte(`{}`))
	if err != nil {
		return err
	}

	return fieldSet.UpdateState(tx, ConnectionGroupFieldSetStateProcessed, stateReason)
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

func (g *ConnectionGroup) FindFieldSetByID(ID uuid.UUID) (*ConnectionGroupFieldSet, error) {
	var fieldSet *ConnectionGroupFieldSet
	err := database.Conn().
		Where("id = ?", ID).
		First(&fieldSet).
		Error

	if err != nil {
		return nil, err
	}

	return fieldSet, nil
}

func (g *ConnectionGroup) FindPendingFieldSetByHash(tx *gorm.DB, hash string) (*ConnectionGroupFieldSet, error) {
	var fieldSet *ConnectionGroupFieldSet
	err := tx.
		Where("connection_group_id = ?", g.ID).
		Where("field_set_hash = ?", hash).
		Where("state = ?", ConnectionGroupFieldSetStatePending).
		First(&fieldSet).
		Error

	if err != nil {
		return nil, err
	}

	return fieldSet, nil
}

func ListPendingConnectionGroupFieldSets() ([]ConnectionGroupFieldSet, error) {
	var fieldSets []ConnectionGroupFieldSet
	err := database.Conn().
		Where("state = ?", ConnectionGroupFieldSetStatePending).
		Where("timeout_behavior != ?", ConnectionGroupTimeoutBehaviorNone).
		Find(&fieldSets).
		Error

	if err != nil {
		return nil, err
	}

	return fieldSets, nil
}

func (g *ConnectionGroup) FindConnectionsForFieldSet(tx *gorm.DB, fieldSet *ConnectionGroupFieldSet) ([]string, error) {
	var connections []string
	err := tx.
		Table("connection_group_field_set_events AS e").
		Joins("JOIN connection_group_field_sets AS f ON f.id = e.connection_group_set_id").
		Select("e.source_id").
		Where("f.connection_group_id = ?", g.ID).
		Where("f.id = ?", fieldSet.ID).
		Where("f.field_set_hash = ?", fieldSet.FieldSetHash).
		Find(&connections).
		Error

	if err != nil {
		return nil, err
	}

	return connections, nil
}

type ConnectionGroupSpec struct {
	GroupBy         *ConnectionGroupBySpec `json:"group_by"`
	Timeout         uint32                 `json:"timeout"`
	TimeoutBehavior string                 `json:"timeout_behavior"`
}

type ConnectionGroupBySpec struct {
	Fields []ConnectionGroupByField `json:"fields"`
}

type ConnectionGroupByField struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

func (c *Canvas) CreateConnectionGroup(
	name, description, createdBy string,
	connections []Connection,
	spec ConnectionGroupSpec,
) (*ConnectionGroup, error) {
	now := time.Now()
	ID := uuid.New()

	var connectionGroup *ConnectionGroup

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		connectionGroup = &ConnectionGroup{
			ID:          ID,
			CanvasID:    c.ID,
			Name:        name,
			Description: description,
			CreatedAt:   &now,
			CreatedBy:   uuid.Must(uuid.Parse(createdBy)),
			Spec:        datatypes.NewJSONType(spec),
		}

		err := tx.Clauses(clause.Returning{}).Create(&connectionGroup).Error
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				return ErrNameAlreadyUsed
			}

			return err
		}

		for _, i := range connections {
			connection := i
			connection.TargetID = ID
			connection.TargetType = ConnectionTargetTypeConnectionGroup
			connection.CanvasID = c.ID
			err := tx.Create(&connection).Error
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

func (g *ConnectionGroup) Update(connections []Connection, spec ConnectionGroupSpec) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("target_id = ?", g.ID).Delete(&Connection{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing connections: %v", err)
		}

		for _, connection := range connections {
			connection.TargetID = g.ID
			connection.TargetType = ConnectionTargetTypeConnectionGroup
			connection.CanvasID = g.CanvasID
			if err := tx.Create(&connection).Error; err != nil {
				return fmt.Errorf("failed to create connection: %v", err)
			}
		}

		now := time.Now()
		g.Spec = datatypes.NewJSONType(spec)
		g.UpdatedAt = &now
		err := tx.Save(g).Error
		if err != nil {
			return fmt.Errorf("failed to update connection group: %v", err)
		}

		return nil
	})
}

func FindConnectionGroupByName(canvasID string, name string) (*ConnectionGroup, error) {
	var connectionGroup ConnectionGroup

	err := database.Conn().
		Where("canvas_id = ?", canvasID).
		Where("name = ?", name).
		First(&connectionGroup).
		Error

	if err != nil {
		return nil, err
	}

	return &connectionGroup, nil
}

func FindConnectionGroupByID(canvasID string, id uuid.UUID) (*ConnectionGroup, error) {
	return FindConnectionGroupByIDInTransaction(database.Conn(), canvasID, id)
}

func FindConnectionGroupByIDInTransaction(tx *gorm.DB, canvasID string, id uuid.UUID) (*ConnectionGroup, error) {
	var connectionGroup ConnectionGroup

	err := database.Conn().
		Where("id = ?", id).
		Where("canvas_id = ?", canvasID).
		First(&connectionGroup).
		Error

	if err != nil {
		return nil, err
	}

	return &connectionGroup, nil
}
