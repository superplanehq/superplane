package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SSOProtocolOIDC = "oidc"

	SSOPresetOkta    = "okta"
	SSOPresetEntra   = "entra"
	SSOPresetGeneric = "generic"

	SSOStatusDraft    = "draft"
	SSOStatusActive   = "active"
	SSOStatusDisabled = "disabled"

	SSOEnforcementOptional = "optional"
	SSOEnforcementRequired = "required"

	SSOProvisionedByJIT  = "jit"
	SSOProvisionedBySCIM = "scim"

	DomainVerificationDNS   = "dns"
	DomainVerificationAdmin = "admin"

	SSOClientSecretAssociatedData = "sso_connection_client_secret"
	SSOIDTokenAssociatedData      = "sso_identity_id_token"

	SSOMinSessionAgeSeconds = 900
	SSOMaxSessionAgeSeconds = 604800
	SSODefaultSessionAge    = 86400
	SSOMaxActiveSCIMTokens  = 5
)

// OIDCConnectionConfig is the protocol document stored in sso_connections.config.
type OIDCConnectionConfig struct {
	Issuer       string   `json:"issuer"`
	ClientID     string   `json:"client_id"`
	Scopes       []string `json:"scopes"`
	SubjectClaim string   `json:"subject_claim"`
	EmailClaim   string   `json:"email_claim"`
	NameClaim    string   `json:"name_claim"`
}

// SSOConnection is one identity-provider connection owned by an organization.
type SSOConnection struct {
	ID                   uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID       uuid.UUID
	Protocol             string
	Preset               string
	DisplayName          string
	Status               string
	Enforcement          string
	AllowOwnerBypass     bool
	JITEnabled           bool
	DefaultRole          string
	MaxSessionAgeSeconds int
	IDPLogoutEnabled     bool `gorm:"column:idp_logout_enabled"`
	Config               datatypes.JSON
	ClientSecret         []byte
	LastTestedAt         *time.Time
	LastTestResult       datatypes.JSON
	CreatedBy            *uuid.UUID
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// SSOTestResultDocument is the JSON stored in last_test_result.
type SSOTestResultDocument struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Subject string `json:"subject,omitempty"`
	Email   string `json:"email,omitempty"`
	Name    string `json:"name,omitempty"`
}

// DefaultOIDCConfig applies preset defaults. Presets only fill empty claims.
func DefaultOIDCConfig(preset string) OIDCConnectionConfig {
	subject := "sub"
	if preset == SSOPresetEntra {
		subject = "oid"
	}
	return OIDCConnectionConfig{
		Scopes:       []string{"openid", "email", "profile"},
		SubjectClaim: subject,
		EmailClaim:   "email",
		NameClaim:    "name",
	}
}

func (SSOConnection) TableName() string { return "sso_connections" }

func (c *SSOConnection) OIDCConfig() (OIDCConnectionConfig, error) {
	cfg := DefaultOIDCConfig(c.Preset)
	if len(c.Config) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(c.Config, &cfg); err != nil {
		return OIDCConnectionConfig{}, err
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid", "email", "profile"}
	}
	if cfg.SubjectClaim == "" {
		cfg.SubjectClaim = DefaultOIDCConfig(c.Preset).SubjectClaim
	}
	if cfg.EmailClaim == "" {
		cfg.EmailClaim = "email"
	}
	if cfg.NameClaim == "" {
		cfg.NameClaim = "name"
	}
	return cfg, nil
}

func (c *SSOConnection) SetOIDCConfig(cfg OIDCConnectionConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	c.Config = raw
	return nil
}

func (c *SSOConnection) TestResult() (*SSOTestResultDocument, error) {
	if len(c.LastTestResult) == 0 {
		return nil, nil
	}
	var result SSOTestResultDocument
	if err := json.Unmarshal(c.LastTestResult, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func LockSSOConnection(tx *gorm.DB, id uuid.UUID) (*SSOConnection, error) {
	var connection SSOConnection
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&connection).Error
	if err != nil {
		return nil, err
	}
	return &connection, nil
}

func FindSSOConnectionByOrganization(tx *gorm.DB, orgID uuid.UUID) (*SSOConnection, error) {
	var connection SSOConnection
	err := tx.Where("organization_id = ?", orgID).First(&connection).Error
	if err != nil {
		return nil, err
	}
	return &connection, nil
}

func FindSSOConnectionByID(tx *gorm.DB, id uuid.UUID) (*SSOConnection, error) {
	var connection SSOConnection
	err := tx.Where("id = ?", id).First(&connection).Error
	if err != nil {
		return nil, err
	}
	return &connection, nil
}

func FindActiveSSOConnection(tx *gorm.DB, orgID uuid.UUID) (*SSOConnection, error) {
	var connection SSOConnection
	err := tx.Where("organization_id = ? AND status = ?", orgID, SSOStatusActive).First(&connection).Error
	if err != nil {
		return nil, err
	}
	return &connection, nil
}

func ActiveSSOConnectionExists(tx *gorm.DB) (bool, error) {
	var count int64
	err := tx.Model(&SSOConnection{}).Where("status = ?", SSOStatusActive).Count(&count).Error
	return count > 0, err
}

func (c *SSOConnection) Save(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	return tx.Save(c).Error
}

func DeleteSSOConnection(tx *gorm.DB, id uuid.UUID) error {
	return tx.Where("id = ?", id).Delete(&SSOConnection{}).Error
}

// ValidSSOPreset reports whether the preset name is one this release stores.
func ValidSSOPreset(preset string) bool {
	switch preset {
	case SSOPresetOkta, SSOPresetEntra, SSOPresetGeneric:
		return true
	default:
		return false
	}
}

// ValidSSORole reports whether the role is a built-in organization role.
func ValidSSORole(role string) bool {
	return role == RoleOrgAdmin || role == RoleOrgMaintainer || role == RoleOrgOperator
}

// ValidSSOEnforcement reports whether the enforcement mode is known.
func ValidSSOEnforcement(enforcement string) bool {
	return enforcement == SSOEnforcementOptional || enforcement == SSOEnforcementRequired
}

func NormalizeSSOPreset(preset string) string {
	switch strings.TrimSpace(strings.ToLower(preset)) {
	case SSOPresetOkta:
		return SSOPresetOkta
	case SSOPresetEntra:
		return SSOPresetEntra
	default:
		return SSOPresetGeneric
	}
}
