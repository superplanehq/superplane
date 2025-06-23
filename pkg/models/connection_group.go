package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ConnectionGroupPolicyTypeAll              = "all"
	ConnectionGroupPolicyTypeMajority         = "majority"
	ConnectionGroupTimeoutBehaviorFail        = "fail"
	ConnectionGroupTimeoutBehaviorDrop        = "drop"
	ConnectionGroupTimeoutBehaviorEmitPartial = "emit-partial"
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

type ConnectionGroupSpec struct {
	Keys   []ConnectionGroupKeyDefinition `json:"keys"`
	Policy ConnectionGroupPolicy          `json:"policy"`
}

type ConnectionGroupKeyDefinition struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

type ConnectionGroupPolicy struct {
	Type            string `json:"type"`
	Timeout         string `json:"timeout"`
	TimeoutBehavior string `json:"timeoutBehavior"`
}

type ConnectionGroupEvent struct {
	ID                uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	ConnectionGroupID uuid.UUID
	EventID           uuid.UUID
	SourceID          uuid.UUID
	SourceName        string
	SourceType        string
	State             string
	CreatedAt         *time.Time
}

type ConnectionGroupKey struct {
	ID                uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	ConnectionGroupID uuid.UUID
	SourceID          uuid.UUID
	Name              string
	Value             string
}

func CreateConnectionGroupEvent(tx *gorm.DB, connectionGroupID uuid.UUID, event *Event) (*ConnectionGroupEvent, error) {
	now := time.Now()
	ID := uuid.New()

	connectionGroupEvent := ConnectionGroupEvent{
		ID:                ID,
		ConnectionGroupID: connectionGroupID,
		EventID:           event.ID,
		SourceID:          event.SourceID,
		SourceName:        event.SourceName,
		SourceType:        event.SourceType,
		CreatedAt:         &now,
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

func FindConnectionsWithGroupKey(tx *gorm.DB, groupID uuid.UUID, name, value string) ([]string, error) {
	var connections []string
	err := tx.
		Table("connection_group_keys").
		Where("connection_group_id = ?", groupID).
		Where("key = ?", name).
		Where("value = ?", value).
		Find(&connections).
		Error

	if err != nil {
		return nil, err
	}

	return connections, nil
}
