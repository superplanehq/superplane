package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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

	ConnectionGroupFieldSetResultTimedOut    = "timed-out"
	ConnectionGroupFieldSetResultReceivedAll = "received-all"

	ConnectionGroupTimeoutBehaviorNone = "none"
	ConnectionGroupTimeoutBehaviorEmit = "emit"
	ConnectionGroupTimeoutBehaviorDrop = "drop"

	MinConnectionGroupTimeout = 60    // 1 minute
	MaxConnectionGroupTimeout = 86400 // 1 day
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

	hash, err := crypto.SHA256ForMap(fieldSet)
	if err != nil {
		return nil, "", err
	}

	return fieldSet, hash, nil
}

func (g *ConnectionGroup) ShouldEmit(tx *gorm.DB, fieldSet *ConnectionGroupFieldSet) (bool, error) {
	allConnections, err := ListConnectionsInTransaction(tx, g.ID, ConnectionTargetTypeConnectionGroup)
	if err != nil {
		return false, fmt.Errorf("error listing connections: %v", err)
	}

	missing, err := fieldSet.MissingConnections(tx, g, allConnections)
	if err != nil {
		return false, err
	}

	fields := fieldSet.FieldSet.Data()
	if len(missing) > 0 {
		log.Infof("Connection group %s has missing connections for field set %s - %v: %v",
			g.Name, fieldSet.ID.String(), fields, sourceNamesFromConnections(missing),
		)

		return false, nil
	}

	log.Infof("All connections received for group %s and field set %s - %v", g.Name, fieldSet.ID.String(), fields)
	return true, nil
}

func (g *ConnectionGroup) Emit(tx *gorm.DB, fieldSet *ConnectionGroupFieldSet, result string) error {
	eventData, err := fieldSet.BuildEvent(tx, result)
	if err != nil {
		return fmt.Errorf("error building connection group event: %v", err)
	}

	_, err = CreateEventInTransaction(tx, g.ID, g.Name, SourceTypeConnectionGroup, eventData, []byte(`{}`))
	if err != nil {
		return err
	}

	return fieldSet.UpdateState(tx, ConnectionGroupFieldSetStateProcessed, result)
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

func sourceNamesFromConnections(connections []Connection) []string {
	sourceNames := []string{}
	for _, connection := range connections {
		sourceNames = append(sourceNames, connection.SourceName)
	}
	return sourceNames
}
