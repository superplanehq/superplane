package models

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrMCPOAuthClientNotFound  = errors.New("mcp oauth client not found")
	ErrMCPOAuthCodeNotFound    = errors.New("mcp oauth code not found")
	ErrMCPOAuthRefreshNotFound = errors.New("mcp oauth refresh token not found")
)

type MCPOAuthClient struct {
	ID           uuid.UUID
	ClientID     string
	ClientName   string
	RedirectURIs datatypes.JSONSlice[string]
	CreatedAt    time.Time
}

func (MCPOAuthClient) TableName() string {
	return "mcp_oauth_clients"
}

type MCPOAuthCode struct {
	ID                  uuid.UUID
	CodeHash            string
	ClientID            string
	RedirectURI         string
	Resource            string
	CodeChallenge       string
	CodeChallengeMethod string
	UserID              uuid.UUID
	OrganizationID      uuid.UUID
	FactoryID           uuid.UUID
	Scopes              datatypes.JSONSlice[string]
	ExpiresAt           time.Time
	CreatedAt           time.Time
}

func (MCPOAuthCode) TableName() string {
	return "mcp_oauth_codes"
}

type MCPOAuthRefreshToken struct {
	ID             uuid.UUID
	TokenHash      string
	ClientID       string
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	Resource       string
	Scopes         datatypes.JSONSlice[string]
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

func (MCPOAuthRefreshToken) TableName() string {
	return "mcp_oauth_refresh_tokens"
}

func HashMCPOAuthSecret(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func CreateMCPOAuthClient(tx *gorm.DB, clientID, clientName string, redirectURIs []string) (*MCPOAuthClient, error) {
	now := time.Now()
	client := &MCPOAuthClient{
		ID:           uuid.New(),
		ClientID:     clientID,
		ClientName:   clientName,
		RedirectURIs: datatypes.NewJSONSlice(redirectURIs),
		CreatedAt:    now,
	}
	if err := tx.Create(client).Error; err != nil {
		return nil, err
	}
	return client, nil
}

func FindMCPOAuthClient(tx *gorm.DB, clientID string) (*MCPOAuthClient, error) {
	var client MCPOAuthClient
	err := tx.Where("client_id = ?", clientID).First(&client).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMCPOAuthClientNotFound
	}
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func CreateMCPOAuthCode(tx *gorm.DB, code *MCPOAuthCode) error {
	if code.ID == uuid.Nil {
		code.ID = uuid.New()
	}
	if code.CreatedAt.IsZero() {
		code.CreatedAt = time.Now()
	}
	return tx.Create(code).Error
}

func ConsumeMCPOAuthCode(tx *gorm.DB, codeHash string, now time.Time) (*MCPOAuthCode, error) {
	var code MCPOAuthCode
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("code_hash = ? AND expires_at > ?", codeHash, now).
		First(&code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMCPOAuthCodeNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Delete(&code).Error; err != nil {
		return nil, err
	}
	return &code, nil
}

func CreateMCPOAuthRefreshToken(tx *gorm.DB, token *MCPOAuthRefreshToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}
	return tx.Create(token).Error
}

func ConsumeMCPOAuthRefreshToken(tx *gorm.DB, tokenHash string, now time.Time) (*MCPOAuthRefreshToken, error) {
	var token MCPOAuthRefreshToken
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("token_hash = ? AND expires_at > ?", tokenHash, now).
		First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMCPOAuthRefreshNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Delete(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}
