package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryAgentAssignmentStatePending   = "pending"
	FactoryAgentAssignmentStateStarted   = "started"
	FactoryAgentAssignmentStateCompleted = "completed"
	FactoryAgentAssignmentStateFailed    = "failed"
)

var ErrFactoryAgentNotFound = errors.New("factory agent not found")
var ErrFactoryAgentNameAlreadyExists = errors.New("factory agent name already exists")

const factoryAgentNameUniqueConstraint = "factory_agents_factory_id_name_key"

type FactoryAgentMachine struct {
	Type string `json:"type"`
}

type FactoryAgentEnvSource struct {
	Source      string                      `json:"source"`
	Secret      *FactoryAgentSecretRef      `json:"secret,omitempty"`
	Integration *FactoryAgentIntegrationRef `json:"integration,omitempty"`
}

type FactoryAgentSecretRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type FactoryAgentIntegrationRef struct {
	ID   *string `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}

type FactoryAgentEnvVar struct {
	Source string                 `json:"source"`
	Secret *FactoryAgentSecretRef `json:"secret,omitempty"`
	Value  *string                `json:"value,omitempty"`
}

type FactoryAgentSpec struct {
	Kind    string                  `json:"kind,omitempty"`
	Model   string                  `json:"model,omitempty"`
	Machine FactoryAgentMachine     `json:"machine,omitempty"`
	EnvFrom []FactoryAgentEnvSource `json:"envFrom,omitempty"`
	Env     []FactoryAgentEnvVar    `json:"env,omitempty"`
}

type FactoryAgent struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	Name           string
	Description    string
	Spec           datatypes.JSONType[FactoryAgentSpec]
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (FactoryAgent) TableName() string {
	return "factory_agents"
}

func CreateFactoryAgent(
	tx *gorm.DB,
	organizationID, factoryID uuid.UUID,
	name, description string,
	spec FactoryAgentSpec,
) (*FactoryAgent, error) {
	now := time.Now()
	agent := &FactoryAgent{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		FactoryID:      factoryID,
		Name:           name,
		Description:    description,
		Spec:           datatypes.NewJSONType(spec),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(agent).Error; err != nil {
		return nil, mapFactoryAgentNameUniqueConstraintError(err)
	}

	return agent, nil
}

func mapFactoryAgentNameUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == factoryAgentNameUniqueConstraint {
		return ErrFactoryAgentNameAlreadyExists
	}

	return err
}

func FindFactoryAgent(tx *gorm.DB, organizationID, factoryID, agentID uuid.UUID) (*FactoryAgent, error) {
	var agent FactoryAgent
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND id = ?", organizationID, factoryID, agentID).
		First(&agent).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryAgentNotFound
		}
		return nil, err
	}

	return &agent, nil
}

func ListFactoryAgents(tx *gorm.DB, organizationID, factoryID uuid.UUID) ([]FactoryAgent, error) {
	var agents []FactoryAgent
	err := tx.
		Where("organization_id = ? AND factory_id = ?", organizationID, factoryID).
		Order("name ASC").
		Order("id ASC").
		Find(&agents).
		Error
	if err != nil {
		return nil, err
	}

	return agents, nil
}
