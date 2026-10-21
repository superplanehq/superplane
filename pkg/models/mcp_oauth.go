package models

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
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
	FactoryIDs          datatypes.JSONSlice[string] `gorm:"column:factory_ids"`
	Scopes              datatypes.JSONSlice[string]
	ExpiresAt           time.Time
	CreatedAt           time.Time
}

func (MCPOAuthCode) TableName() string {
	return "mcp_oauth_codes"
}

func (code MCPOAuthCode) WorkspaceIDs() []uuid.UUID {
	return workspaceIDsFromStorage(code.FactoryIDs, code.FactoryID)
}

type MCPOAuthRefreshToken struct {
	ID             uuid.UUID
	TokenHash      string
	ClientID       string
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	FactoryIDs     datatypes.JSONSlice[string] `gorm:"column:factory_ids"`
	Resource       string
	Scopes         datatypes.JSONSlice[string]
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

func (MCPOAuthRefreshToken) TableName() string {
	return "mcp_oauth_refresh_tokens"
}

func (token MCPOAuthRefreshToken) WorkspaceIDs() []uuid.UUID {
	return workspaceIDsFromStorage(token.FactoryIDs, token.FactoryID)
}

func (token MCPOAuthRefreshToken) WorkspaceCount() int32 {
	count := len(token.WorkspaceIDs())
	if count == 0 {
		return 1
	}
	return int32(count)
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
	if err := assignStoredFactoryIDs(&code.FactoryIDs, &code.FactoryID); err != nil {
		return err
	}
	return tx.Create(code).Error
}

func FindMCPOAuthCode(tx *gorm.DB, codeHash string, now time.Time) (*MCPOAuthCode, error) {
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
	return &code, nil
}

func DeleteMCPOAuthCode(tx *gorm.DB, code *MCPOAuthCode) error {
	return tx.Delete(code).Error
}

func CreateMCPOAuthRefreshToken(tx *gorm.DB, token *MCPOAuthRefreshToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}
	if err := assignStoredFactoryIDs(&token.FactoryIDs, &token.FactoryID); err != nil {
		return err
	}
	return tx.Create(token).Error
}

func FindMCPOAuthRefreshToken(tx *gorm.DB, tokenHash string, now time.Time) (*MCPOAuthRefreshToken, error) {
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
	return &token, nil
}

func DeleteMCPOAuthRefreshToken(tx *gorm.DB, token *MCPOAuthRefreshToken) error {
	return tx.Delete(token).Error
}

func ListMCPOAuthRefreshTokensForFactory(tx *gorm.DB, organizationID, factoryID uuid.UUID, now time.Time) ([]MCPOAuthRefreshToken, error) {
	contains, args, err := factoryContainsClause(factoryID)
	if err != nil {
		return nil, err
	}
	var tokens []MCPOAuthRefreshToken
	err = tx.
		Where("organization_id = ? AND expires_at > ?", organizationID, now).
		Where(contains, args...).
		Order("created_at DESC").
		Find(&tokens).Error
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func FindMCPOAuthRefreshTokenForFactory(tx *gorm.DB, organizationID, factoryID, id uuid.UUID) (*MCPOAuthRefreshToken, error) {
	contains, args, err := factoryContainsClause(factoryID)
	if err != nil {
		return nil, err
	}
	var token MCPOAuthRefreshToken
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND organization_id = ?", id, organizationID).
		Where(contains, args...).
		First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMCPOAuthRefreshNotFound
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func HasMCPOAuthRefreshTokenForClient(
	tx *gorm.DB,
	organizationID, factoryID, userID uuid.UUID,
	clientID string,
	now time.Time,
) (bool, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return false, nil
	}
	var count int64
	contains, args, err := factoryContainsClause(factoryID)
	if err != nil {
		return false, err
	}
	queryArgs := append([]any{organizationID, userID, clientID, now}, args...)
	err = tx.Model(&MCPOAuthRefreshToken{}).
		Where(
			"organization_id = ? AND user_id = ? AND client_id = ? AND expires_at > ? AND "+contains,
			queryArgs...,
		).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func ListActiveMCPOAuthRefreshTokensForClient(
	tx *gorm.DB,
	organizationID, userID uuid.UUID,
	clientID string,
	now time.Time,
) ([]MCPOAuthRefreshToken, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, nil
	}
	var tokens []MCPOAuthRefreshToken
	err := tx.
		Where(
			"organization_id = ? AND user_id = ? AND client_id = ? AND expires_at > ?",
			organizationID,
			userID,
			clientID,
			now,
		).
		Find(&tokens).Error
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func DeleteMCPOAuthRefreshTokensForWorkspaceSet(
	tx *gorm.DB,
	organizationID, userID uuid.UUID,
	clientID string,
	factoryIDs []uuid.UUID,
) error {
	return deleteMCPOAuthRowsForWorkspaceSet(tx, &MCPOAuthRefreshToken{}, organizationID, userID, clientID, factoryIDs)
}

func DeleteMCPOAuthCodesForWorkspaceSet(
	tx *gorm.DB,
	organizationID, userID uuid.UUID,
	clientID string,
	factoryIDs []uuid.UUID,
) error {
	return deleteMCPOAuthRowsForWorkspaceSet(tx, &MCPOAuthCode{}, organizationID, userID, clientID, factoryIDs)
}

func DeleteMCPOAuthRefreshTokensForClient(
	tx *gorm.DB,
	organizationID, factoryID, userID uuid.UUID,
	clientID string,
) error {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil
	}
	return tx.
		Where(
			"organization_id = ? AND factory_id = ? AND user_id = ? AND client_id = ?",
			organizationID,
			factoryID,
			userID,
			clientID,
		).
		Delete(&MCPOAuthRefreshToken{}).Error
}

func DeleteMCPOAuthCodesForClient(
	tx *gorm.DB,
	organizationID, factoryID, userID uuid.UUID,
	clientID string,
) error {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil
	}
	return tx.
		Where(
			"organization_id = ? AND factory_id = ? AND user_id = ? AND client_id = ?",
			organizationID,
			factoryID,
			userID,
			clientID,
		).
		Delete(&MCPOAuthCode{}).Error
}

func ListMCPOAuthClientsByClientIDs(tx *gorm.DB, clientIDs []string) ([]MCPOAuthClient, error) {
	if len(clientIDs) == 0 {
		return nil, nil
	}
	var clients []MCPOAuthClient
	err := tx.Where("client_id IN ?", clientIDs).Find(&clients).Error
	if err != nil {
		return nil, err
	}
	return clients, nil
}

func NormalizeFactoryIDs(ids []uuid.UUID) ([]uuid.UUID, error) {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("workspace is required")
	}
	slices.SortFunc(out, func(left, right uuid.UUID) int {
		return cmp.Compare(left.String(), right.String())
	})
	return out, nil
}

func assignStoredFactoryIDs(stored *datatypes.JSONSlice[string], factoryID *uuid.UUID) error {
	parsed := parseFactoryIDStrings(*stored)
	var normalized []uuid.UUID
	var err error
	if len(parsed) > 0 {
		normalized, err = NormalizeFactoryIDs(parsed)
	} else {
		normalized, err = NormalizeFactoryIDs([]uuid.UUID{*factoryID})
	}
	if err != nil {
		return err
	}
	*factoryID = normalized[0]
	*stored = datatypes.NewJSONSlice(factoryIDStrings(normalized))
	return nil
}

func workspaceIDsFromStorage(stored datatypes.JSONSlice[string], factoryID uuid.UUID) []uuid.UUID {
	parsed := parseFactoryIDStrings(stored)
	if len(parsed) == 0 && factoryID != uuid.Nil {
		parsed = []uuid.UUID{factoryID}
	}
	normalized, err := NormalizeFactoryIDs(parsed)
	if err != nil {
		return nil
	}
	return normalized
}

func parseFactoryIDStrings(stored datatypes.JSONSlice[string]) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(stored))
	for _, raw := range stored {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil || id == uuid.Nil {
			continue
		}
		out = append(out, id)
	}
	return out
}

func factoryIDStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

func factoryContainsClause(factoryID uuid.UUID) (string, []any, error) {
	raw, err := json.Marshal([]string{factoryID.String()})
	if err != nil {
		return "", nil, err
	}
	return "(factory_ids @> ?::jsonb OR ((factory_ids IS NULL OR factory_ids = '[]'::jsonb) AND factory_id = ?))", []any{raw, factoryID}, nil
}

func factorySetClause(ids []uuid.UUID) (string, []any, error) {
	normalized, err := NormalizeFactoryIDs(ids)
	if err != nil {
		return "", nil, err
	}
	raw, err := json.Marshal(factoryIDStrings(normalized))
	if err != nil {
		return "", nil, err
	}
	if len(normalized) == 1 {
		return "(factory_ids = ?::jsonb OR ((factory_ids IS NULL OR factory_ids = '[]'::jsonb) AND factory_id = ?))", []any{raw, normalized[0]}, nil
	}
	return "factory_ids = ?::jsonb", []any{raw}, nil
}

func deleteMCPOAuthRowsForWorkspaceSet(
	tx *gorm.DB,
	model any,
	organizationID, userID uuid.UUID,
	clientID string,
	factoryIDs []uuid.UUID,
) error {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil
	}
	setClause, args, err := factorySetClause(factoryIDs)
	if err != nil {
		return err
	}
	queryArgs := append([]any{organizationID, userID, clientID}, args...)
	return tx.
		Where("organization_id = ? AND user_id = ? AND client_id = ? AND "+setClause, queryArgs...).
		Delete(model).Error
}
