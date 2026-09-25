package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryAgentResourceKindMCPServer = "mcp_server"
	FactoryAgentResourceKindSkill     = "skill"

	FactoryAgentResourceAuthHeaders = "headers"
	FactoryAgentResourceAuthOAuth   = "oauth"

	FactoryAgentResourceSourceInline = "inline"

	FactoryAgentResourceOAuthNotConnected   = "not_connected"
	FactoryAgentResourceOAuthConnected      = "connected"
	FactoryAgentResourceOAuthNeedsReconnect = "needs_reconnect"
	FactoryAgentResourceOAuthVendorRejected = "vendor_rejected"

	FactoryAgentResourceSecretRefreshToken = "refresh_token"
	FactoryAgentResourceSecretAccessToken  = "access_token"
	FactoryAgentResourceSecretClientSecret = "client_secret"
	FactoryAgentResourceSecretCodeVerifier = "code_verifier"

	ReservedFactoryAgentResourceName = "superplane"

	MaxEnabledFactoryMCPServers = 20

	MaxFactoryAgentSkillMarkdownBytes = 64 * 1024

	factoryAgentResourceNameUniqueConstraint = "idx_factory_agent_resources_factory_name"
)

var (
	ErrFactoryAgentResourceNotFound         = errors.New("factory agent resource not found")
	ErrFactoryAgentResourceKindInvalid      = errors.New("factory agent resource kind is not valid")
	ErrFactoryAgentResourceNameInvalid      = errors.New("factory agent resource name is not valid")
	ErrFactoryAgentResourceNameReserved     = errors.New("factory agent resource name is reserved")
	ErrFactoryAgentResourceNameTaken        = errors.New("factory agent resource name already exists")
	ErrFactoryAgentResourceAuthInvalid      = errors.New("factory agent resource auth is not valid")
	ErrFactoryAgentResourceURLRequired      = errors.New("MCP URL is required")
	ErrFactoryAgentResourceHeaderInvalid    = errors.New("MCP header is not valid")
	ErrFactoryAgentResourceKindNotSupported = errors.New("factory agent resource kind is not supported yet")
	ErrFactoryAgentResourceMCPCapReached    = errors.New("workspace already has the maximum number of enabled MCP connections")
	ErrFactoryAgentResourceSecretNotFound   = errors.New("factory agent resource secret not found")
	ErrFactoryAgentResourceMarkdownRequired = errors.New("skill markdown is required")
	ErrFactoryAgentResourceMarkdownTooLarge = errors.New("skill markdown is too large")
)

var factoryAgentResourceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

// FactoryAgentResource is one workspace-scoped agent capability.
type FactoryAgentResource struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	FactoryID          uuid.UUID
	Kind               string
	Name               string
	Enabled            bool
	Config             datatypes.JSONType[FactoryAgentResourceConfig]
	OAuthStatus        string                                                `gorm:"column:oauth_status"`
	OAuthError         string                                                `gorm:"column:oauth_error"`
	OAuthConnectedBy   *uuid.UUID                                            `gorm:"column:oauth_connected_by"`
	OAuthConnectedAt   *time.Time                                            `gorm:"column:oauth_connected_at"`
	OAuthMetadata      datatypes.JSONType[FactoryAgentResourceOAuthMetadata] `gorm:"column:oauth_metadata"`
	OAuthPendingState  string                                                `gorm:"column:oauth_pending_state"`
	OAuthPendingExpiry *time.Time                                            `gorm:"column:oauth_pending_expiry"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type FactoryAgentResourceConfig struct {
	Transport     string                       `json:"transport,omitempty"`
	URL           string                       `json:"url,omitempty"`
	Auth          string                       `json:"auth,omitempty"`
	Headers       []FactoryAgentResourceHeader `json:"headers,omitempty"`
	Source        string                       `json:"source,omitempty"`
	Repository    string                       `json:"repository,omitempty"`
	Ref           string                       `json:"ref,omitempty"`
	Path          string                       `json:"path,omitempty"`
	Markdown      string                       `json:"markdown,omitempty"`
	DisabledTools []string                     `json:"disabledTools,omitempty"`
}

type FactoryAgentResourceHeader struct {
	Name       string `json:"name"`
	SecretName string `json:"secretName"`
	SecretKey  string `json:"secretKey"`
}

type FactoryAgentResourceOAuthMetadata struct {
	AuthorizationEndpoint string `json:"authorizationEndpoint,omitempty"`
	TokenEndpoint         string `json:"tokenEndpoint,omitempty"`
	RevocationEndpoint    string `json:"revocationEndpoint,omitempty"`
	RegistrationEndpoint  string `json:"registrationEndpoint,omitempty"`
	ClientID              string `json:"clientId,omitempty"`
	Resource              string `json:"resource,omitempty"`
}

type FactoryAgentResourceSecret struct {
	ID         uuid.UUID
	ResourceID uuid.UUID
	Name       string
	Value      []byte
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (FactoryAgentResource) TableName() string {
	return "factory_agent_resources"
}

func (FactoryAgentResourceSecret) TableName() string {
	return "factory_agent_resource_secrets"
}

func NormalizeFactoryAgentResourceName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func ValidateFactoryAgentResourceName(name string) error {
	if !factoryAgentResourceNamePattern.MatchString(name) {
		return ErrFactoryAgentResourceNameInvalid
	}
	if name == ReservedFactoryAgentResourceName {
		return ErrFactoryAgentResourceNameReserved
	}
	return nil
}

func ValidateFactoryAgentResourceKind(kind string) error {
	switch kind {
	case FactoryAgentResourceKindMCPServer, FactoryAgentResourceKindSkill:
		return nil
	default:
		return ErrFactoryAgentResourceKindInvalid
	}
}

func (c FactoryAgentResourceConfig) MCPAuth() string {
	auth := strings.TrimSpace(c.Auth)
	if auth == "" {
		return FactoryAgentResourceAuthHeaders
	}
	return auth
}

// InvalidatesOAuth reports whether the next MCP config cannot reuse stored
// OAuth tokens. A URL or auth change points at a different authorization
// server, so SuperPlane must drop the previous grant.
func (c FactoryAgentResourceConfig) InvalidatesOAuth(next FactoryAgentResourceConfig) bool {
	if strings.TrimSpace(c.URL) != strings.TrimSpace(next.URL) {
		return true
	}
	return c.MCPAuth() != next.MCPAuth()
}

func (c FactoryAgentResourceConfig) ValidateMCP() error {
	if strings.TrimSpace(c.URL) == "" {
		return ErrFactoryAgentResourceURLRequired
	}
	switch c.MCPAuth() {
	case FactoryAgentResourceAuthHeaders:
		for _, header := range c.Headers {
			if strings.TrimSpace(header.Name) == "" || strings.TrimSpace(header.SecretName) == "" || strings.TrimSpace(header.SecretKey) == "" {
				return ErrFactoryAgentResourceHeaderInvalid
			}
		}
	case FactoryAgentResourceAuthOAuth:
		if len(c.Headers) > 0 {
			return ErrFactoryAgentResourceHeaderInvalid
		}
	default:
		return ErrFactoryAgentResourceAuthInvalid
	}
	return nil
}

func (c FactoryAgentResourceConfig) ValidateSkill() error {
	source := strings.TrimSpace(c.Source)
	if source != "" && source != FactoryAgentResourceSourceInline {
		return ErrFactoryAgentResourceKindNotSupported
	}
	markdown := strings.TrimSpace(c.Markdown)
	if markdown == "" {
		return ErrFactoryAgentResourceMarkdownRequired
	}
	if len(markdown) > MaxFactoryAgentSkillMarkdownBytes {
		return ErrFactoryAgentResourceMarkdownTooLarge
	}
	return nil
}

func (c FactoryAgentResourceConfig) NormalizedSkill() FactoryAgentResourceConfig {
	c.Source = FactoryAgentResourceSourceInline
	c.Markdown = strings.TrimSpace(c.Markdown)
	c.Transport = ""
	c.URL = ""
	c.Auth = ""
	c.Headers = nil
	c.DisabledTools = nil
	return c
}

func (c FactoryAgentResourceConfig) NormalizedMCP() FactoryAgentResourceConfig {
	c.DisabledTools = NormalizeDisabledTools(c.DisabledTools)
	return c
}

func NormalizeDisabledTools(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (r *FactoryAgentResource) OAuthState() string {
	if strings.TrimSpace(r.OAuthStatus) == "" {
		if r.Config.Data().MCPAuth() == FactoryAgentResourceAuthOAuth {
			return FactoryAgentResourceOAuthNotConnected
		}
		return ""
	}
	return r.OAuthStatus
}

func mapFactoryAgentResourceNameError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == factoryAgentResourceNameUniqueConstraint {
		return ErrFactoryAgentResourceNameTaken
	}
	return err
}

func (f *Factory) CreateAgentResource(tx *gorm.DB, kind, name string, enabled bool, config FactoryAgentResourceConfig) (*FactoryAgentResource, error) {
	name = NormalizeFactoryAgentResourceName(name)
	if err := ValidateFactoryAgentResourceKind(kind); err != nil {
		return nil, err
	}
	if err := ValidateFactoryAgentResourceName(name); err != nil {
		return nil, err
	}
	switch kind {
	case FactoryAgentResourceKindSkill:
		if err := config.ValidateSkill(); err != nil {
			return nil, err
		}
		config = config.NormalizedSkill()
	default:
		if err := config.ValidateMCP(); err != nil {
			return nil, err
		}
		config = config.NormalizedMCP()
	}
	now := time.Now()
	resource := &FactoryAgentResource{
		ID:             uuid.New(),
		OrganizationID: f.OrganizationID,
		FactoryID:      f.ID,
		Kind:           kind,
		Name:           name,
		Enabled:        enabled,
		Config:         datatypes.NewJSONType(config),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if kind == FactoryAgentResourceKindMCPServer && config.MCPAuth() == FactoryAgentResourceAuthOAuth {
		resource.OAuthStatus = FactoryAgentResourceOAuthNotConnected
	}

	err := tx.Transaction(func(inner *gorm.DB) error {
		if enabled && kind == FactoryAgentResourceKindMCPServer {
			if err := f.ensureEnabledMCPCapacity(inner, uuid.Nil); err != nil {
				return err
			}
		}
		if err := inner.Clauses(clause.Returning{}).Create(resource).Error; err != nil {
			return mapFactoryAgentResourceNameError(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resource, nil
}

func (f *Factory) FindAgentResource(tx *gorm.DB, resourceID uuid.UUID) (*FactoryAgentResource, error) {
	var resource FactoryAgentResource
	err := tx.Where("organization_id = ? AND factory_id = ? AND id = ?", f.OrganizationID, f.ID, resourceID).First(&resource).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFactoryAgentResourceNotFound
	}
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

func (f *Factory) ListAgentResources(tx *gorm.DB, kind string) ([]FactoryAgentResource, error) {
	query := tx.Where("organization_id = ? AND factory_id = ?", f.OrganizationID, f.ID)
	if strings.TrimSpace(kind) != "" {
		query = query.Where("kind = ?", kind)
	}
	var resources []FactoryAgentResource
	if err := query.Order("name asc").Find(&resources).Error; err != nil {
		return nil, err
	}
	return resources, nil
}

func (f *Factory) ListEnabledMCPServers(tx *gorm.DB) ([]FactoryAgentResource, error) {
	var resources []FactoryAgentResource
	err := tx.Where(
		"organization_id = ? AND factory_id = ? AND kind = ? AND enabled = ?",
		f.OrganizationID,
		f.ID,
		FactoryAgentResourceKindMCPServer,
		true,
	).Order("name asc").Find(&resources).Error
	if err != nil {
		return nil, err
	}
	return resources, nil
}

func (f *Factory) ListEnabledSkills(tx *gorm.DB) ([]FactoryAgentResource, error) {
	var resources []FactoryAgentResource
	err := tx.Where(
		"organization_id = ? AND factory_id = ? AND kind = ? AND enabled = ?",
		f.OrganizationID,
		f.ID,
		FactoryAgentResourceKindSkill,
		true,
	).Order("name asc").Find(&resources).Error
	if err != nil {
		return nil, err
	}
	return resources, nil
}

func (r *FactoryAgentResource) Update(tx *gorm.DB, name *string, enabled *bool, config *FactoryAgentResourceConfig) error {
	return tx.Transaction(func(inner *gorm.DB) error {
		updates := map[string]any{
			"updated_at": time.Now(),
		}
		if name != nil {
			normalized := NormalizeFactoryAgentResourceName(*name)
			if err := ValidateFactoryAgentResourceName(normalized); err != nil {
				return err
			}
			updates["name"] = normalized
			r.Name = normalized
		}
		if enabled != nil {
			if *enabled && r.Kind == FactoryAgentResourceKindMCPServer {
				factory := &Factory{ID: r.FactoryID, OrganizationID: r.OrganizationID}
				if err := factory.ensureEnabledMCPCapacity(inner, r.ID); err != nil {
					return err
				}
			}
			updates["enabled"] = *enabled
			r.Enabled = *enabled
		}
		if config != nil {
			if r.Kind == FactoryAgentResourceKindSkill {
				if err := config.ValidateSkill(); err != nil {
					return err
				}
				normalized := config.NormalizedSkill()
				config = &normalized
			} else if err := config.ValidateMCP(); err != nil {
				return err
			} else {
				normalized := config.NormalizedMCP()
				config = &normalized
			}
			if r.Kind == FactoryAgentResourceKindMCPServer && r.Config.Data().InvalidatesOAuth(*config) {
				if err := r.DeleteSecrets(inner); err != nil {
					return err
				}
				for key, value := range r.oauthResetUpdates(config.MCPAuth()) {
					updates[key] = value
				}
			} else if r.Kind == FactoryAgentResourceKindMCPServer && config.MCPAuth() != FactoryAgentResourceAuthOAuth {
				updates["oauth_status"] = ""
				updates["oauth_error"] = ""
				r.OAuthStatus = ""
				r.OAuthError = ""
			} else if r.Kind == FactoryAgentResourceKindMCPServer && r.OAuthStatus == "" {
				updates["oauth_status"] = FactoryAgentResourceOAuthNotConnected
				r.OAuthStatus = FactoryAgentResourceOAuthNotConnected
			}
			updates["config"] = datatypes.NewJSONType(*config)
			r.Config = datatypes.NewJSONType(*config)
		}

		err := inner.Model(r).Clauses(clause.Returning{}).Updates(updates).Error
		return mapFactoryAgentResourceNameError(err)
	})
}

func (r *FactoryAgentResource) oauthResetUpdates(nextAuth string) map[string]any {
	status := ""
	if nextAuth == FactoryAgentResourceAuthOAuth {
		status = FactoryAgentResourceOAuthNotConnected
	}
	r.OAuthStatus = status
	r.OAuthError = ""
	r.OAuthConnectedBy = nil
	r.OAuthConnectedAt = nil
	r.OAuthMetadata = datatypes.NewJSONType(FactoryAgentResourceOAuthMetadata{})
	r.OAuthPendingState = ""
	r.OAuthPendingExpiry = nil
	return map[string]any{
		"oauth_status":         status,
		"oauth_error":          "",
		"oauth_connected_by":   nil,
		"oauth_connected_at":   nil,
		"oauth_metadata":       r.OAuthMetadata,
		"oauth_pending_state":  "",
		"oauth_pending_expiry": nil,
	}
}

func (r *FactoryAgentResource) Delete(tx *gorm.DB) error {
	if err := tx.Where("resource_id = ?", r.ID).Delete(&FactoryAgentResourceSecret{}).Error; err != nil {
		return err
	}
	return tx.Delete(r).Error
}

func (r *FactoryAgentResource) SetOAuthStatus(tx *gorm.DB, status, message string, connectedBy *uuid.UUID) error {
	now := time.Now()
	updates := map[string]any{
		"oauth_status": status,
		"oauth_error":  strings.TrimSpace(message),
		"updated_at":   now,
	}
	r.OAuthStatus = status
	r.OAuthError = strings.TrimSpace(message)
	if status == FactoryAgentResourceOAuthConnected {
		updates["oauth_connected_at"] = now
		r.OAuthConnectedAt = &now
		if connectedBy != nil {
			updates["oauth_connected_by"] = *connectedBy
			r.OAuthConnectedBy = connectedBy
		}
	}
	if status == FactoryAgentResourceOAuthNotConnected {
		updates["oauth_connected_at"] = nil
		updates["oauth_connected_by"] = nil
		r.OAuthConnectedAt = nil
		r.OAuthConnectedBy = nil
	}
	return tx.Model(r).Updates(updates).Error
}

func (r *FactoryAgentResource) SetOAuthConnector(tx *gorm.DB, userID uuid.UUID) error {
	r.OAuthConnectedBy = &userID
	return tx.Model(r).Updates(map[string]any{
		"oauth_connected_by": userID,
		"updated_at":         time.Now(),
	}).Error
}

func (r *FactoryAgentResource) SetOAuthMetadata(tx *gorm.DB, metadata FactoryAgentResourceOAuthMetadata) error {
	r.OAuthMetadata = datatypes.NewJSONType(metadata)
	return tx.Model(r).Updates(map[string]any{
		"oauth_metadata": r.OAuthMetadata,
		"updated_at":     time.Now(),
	}).Error
}

func (r *FactoryAgentResource) SetOAuthPending(tx *gorm.DB, state string, expiresAt time.Time) error {
	r.OAuthPendingState = state
	r.OAuthPendingExpiry = &expiresAt
	return tx.Model(r).Updates(map[string]any{
		"oauth_pending_state":  state,
		"oauth_pending_expiry": expiresAt,
		"updated_at":           time.Now(),
	}).Error
}

func (r *FactoryAgentResource) ClearOAuthPending(tx *gorm.DB) error {
	r.OAuthPendingState = ""
	r.OAuthPendingExpiry = nil
	return tx.Model(r).Updates(map[string]any{
		"oauth_pending_state":  "",
		"oauth_pending_expiry": nil,
		"updated_at":           time.Now(),
	}).Error
}

func FindFactoryAgentResourceByOAuthState(tx *gorm.DB, state string) (*FactoryAgentResource, error) {
	state = strings.TrimSpace(state)
	if state == "" {
		return nil, ErrFactoryAgentResourceNotFound
	}
	var resource FactoryAgentResource
	err := tx.Where("oauth_pending_state = ?", state).First(&resource).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFactoryAgentResourceNotFound
	}
	if err != nil {
		return nil, err
	}
	if resource.OAuthPendingExpiry != nil && time.Now().After(*resource.OAuthPendingExpiry) {
		return nil, ErrFactoryAgentResourceNotFound
	}
	return &resource, nil
}

func FindFactoryIDForCanvas(tx *gorm.DB, orgID uuid.UUID, canvasID uuid.UUID) (*uuid.UUID, error) {
	var canvas Canvas
	err := tx.Select("factory_id").Where("organization_id = ? AND id = ?", orgID, canvasID).First(&canvas).Error
	if err != nil {
		return nil, err
	}
	return canvas.FactoryID, nil
}

func (r *FactoryAgentResource) UpsertSecret(tx *gorm.DB, name string, value []byte) error {
	now := time.Now()
	secret := FactoryAgentResourceSecret{
		ID:         uuid.New(),
		ResourceID: r.ID,
		Name:       name,
		Value:      value,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "resource_id"}, {Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&secret).Error
}

func (r *FactoryAgentResource) FindSecret(tx *gorm.DB, name string) (*FactoryAgentResourceSecret, error) {
	var secret FactoryAgentResourceSecret
	err := tx.Where("resource_id = ? AND name = ?", r.ID, name).First(&secret).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFactoryAgentResourceSecretNotFound
	}
	if err != nil {
		return nil, err
	}
	return &secret, nil
}

func (r *FactoryAgentResource) DeleteSecret(tx *gorm.DB, name string) error {
	return tx.Where("resource_id = ? AND name = ?", r.ID, name).Delete(&FactoryAgentResourceSecret{}).Error
}

func (r *FactoryAgentResource) DeleteSecrets(tx *gorm.DB) error {
	return tx.Where("resource_id = ?", r.ID).Delete(&FactoryAgentResourceSecret{}).Error
}

func (f *Factory) lockAgentResourceCapacity(tx *gorm.DB) error {
	var locked Factory
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where("id = ? AND organization_id = ?", f.ID, f.OrganizationID).
		First(&locked).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrFactoryNotFound
	}
	return err
}

func (f *Factory) ensureEnabledMCPCapacity(tx *gorm.DB, exceptID uuid.UUID) error {
	if err := f.lockAgentResourceCapacity(tx); err != nil {
		return err
	}
	query := tx.Model(&FactoryAgentResource{}).Where(
		"factory_id = ? AND kind = ? AND enabled = ?",
		f.ID,
		FactoryAgentResourceKindMCPServer,
		true,
	)
	if exceptID != uuid.Nil {
		query = query.Where("id <> ?", exceptID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count >= MaxEnabledFactoryMCPServers {
		return fmt.Errorf("%w", ErrFactoryAgentResourceMCPCapReached)
	}
	return nil
}
