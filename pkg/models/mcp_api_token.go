package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const MCPAPITokenPrefix = "sp_mcp_"

var MCPGrantedScopes = []string{
	"work_orders:read",
	"work_orders:create",
	"work_orders:update",
}

var (
	ErrMCPAPITokenNotFound     = errors.New("mcp api token not found")
	ErrMCPAPITokenNameRequired = errors.New("mcp api token name is required")
)

type MCPAPIToken struct {
	ID             uuid.UUID
	Name           string
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	Resource       string
	Scopes         datatypes.JSONSlice[string]
	TokenHash      string
	CreatedAt      time.Time
	LastUsedAt     *time.Time
}

func NewMCPAPIToken(userID, organizationID, factoryID uuid.UUID, name, resource, tokenHash string, scopes []string) *MCPAPIToken {
	return &MCPAPIToken{
		UserID:         userID,
		OrganizationID: organizationID,
		FactoryID:      factoryID,
		Name:           strings.TrimSpace(name),
		Resource:       strings.TrimSpace(resource),
		TokenHash:      tokenHash,
		Scopes:         datatypes.NewJSONSlice(append([]string{}, scopes...)),
	}
}

func (MCPAPIToken) TableName() string {
	return "mcp_api_tokens"
}

func CreateMCPAPIToken(tx *gorm.DB, token *MCPAPIToken) error {
	if token == nil || strings.TrimSpace(token.Name) == "" {
		return ErrMCPAPITokenNameRequired
	}
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}
	return tx.Create(token).Error
}

func FindMCPAPITokenByHash(tx *gorm.DB, tokenHash string) (*MCPAPIToken, error) {
	var token MCPAPIToken
	err := tx.Where("token_hash = ?", tokenHash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMCPAPITokenNotFound
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func FindMCPAPITokenByID(tx *gorm.DB, id uuid.UUID) (*MCPAPIToken, error) {
	var token MCPAPIToken
	err := tx.Where("id = ?", id).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMCPAPITokenNotFound
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func FindMCPAPITokenForFactory(tx *gorm.DB, organizationID, factoryID, id uuid.UUID) (*MCPAPIToken, error) {
	var token MCPAPIToken
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND organization_id = ? AND factory_id = ?", id, organizationID, factoryID).
		First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMCPAPITokenNotFound
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func ListMCPAPITokensForFactory(tx *gorm.DB, organizationID, factoryID uuid.UUID) ([]MCPAPIToken, error) {
	var tokens []MCPAPIToken
	err := tx.
		Where("organization_id = ? AND factory_id = ?", organizationID, factoryID).
		Order("created_at DESC").
		Find(&tokens).Error
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func TouchMCPAPITokenLastUsed(tx *gorm.DB, id uuid.UUID, when time.Time) error {
	return tx.Model(&MCPAPIToken{}).Where("id = ?", id).Update("last_used_at", when).Error
}

func (t *MCPAPIToken) HardDelete(tx *gorm.DB) error {
	return tx.Delete(t).Error
}

func DeleteMCPAPITokensForAccount(tx *gorm.DB, accountID uuid.UUID) error {
	userIDs := tx.Unscoped().Model(&User{}).Select("id").Where("account_id = ?", accountID)
	return tx.Where("user_id IN (?)", userIDs).Delete(&MCPAPIToken{}).Error
}

func DeleteMCPAPITokensForFactory(tx *gorm.DB, factoryID uuid.UUID) error {
	return tx.Where("factory_id = ?", factoryID).Delete(&MCPAPIToken{}).Error
}
