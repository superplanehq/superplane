package models

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryAgentResourceKindMCPServer   = "mcp_server"
	FactoryAgentResourceKindSkill       = "skill"
	FactoryAgentResourceKindSetup       = "setup_command"
	FactoryAgentResourceKindContextRepo = "context_repo"

	FactoryAgentResourceSetupName = "setup"

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

	MaxFactorySetupCommands         = 100
	MaxFactorySetupCommandBytes     = 32 * 1024
	MaxFactorySetupCommandLineBytes = 2000
	MaxEnabledFactoryContextRepos   = 20

	factoryAgentResourceNameUniqueConstraint = "idx_factory_agent_resources_factory_name"
)

var (
	ErrFactorySystemSkillProtected               = errors.New("system skills cannot be deleted, renamed, or disabled")
	ErrFactorySystemSkillMetadataInvalid         = errors.New("system skills require a template, a positive version, and an enabled skill")
	ErrFactoryAgentResourceNotFound              = errors.New("factory agent resource not found")
	ErrFactoryAgentResourceKindInvalid           = errors.New("factory agent resource kind is not valid")
	ErrFactoryAgentResourceNameInvalid           = errors.New("factory agent resource name is not valid")
	ErrFactoryAgentResourceNameReserved          = errors.New("factory agent resource name is reserved")
	ErrFactoryAgentResourceNameSetupReserved     = errors.New("factory agent resource name is reserved for setup commands")
	ErrFactoryAgentResourceNameTaken             = errors.New("factory agent resource name already exists")
	ErrFactoryAgentResourceAuthInvalid           = errors.New("factory agent resource auth is not valid")
	ErrFactoryAgentResourceURLRequired           = errors.New("MCP URL is required")
	ErrFactoryAgentResourceURLTaken              = errors.New("factory agent resource URL already connected")
	ErrFactoryAgentResourceHeaderInvalid         = errors.New("MCP header is not valid")
	ErrFactoryAgentResourceKindNotSupported      = errors.New("factory agent resource kind is not supported yet")
	ErrFactoryAgentResourceMCPCapReached         = errors.New("workspace already has the maximum number of enabled MCP connections")
	ErrFactoryAgentResourceSecretNotFound        = errors.New("factory agent resource secret not found")
	ErrFactoryAgentResourceMarkdownRequired      = errors.New("skill markdown is required")
	ErrFactoryAgentResourceMarkdownTooLarge      = errors.New("skill markdown is too large")
	ErrFactoryAgentResourceCommandsRequired      = errors.New("setup commands are required")
	ErrFactoryAgentResourceCommandTooLarge       = errors.New("setup commands are too large")
	ErrFactoryAgentResourceTooManyCommands       = errors.New("too many setup commands")
	ErrFactoryAgentResourceRepositoryRequired    = errors.New("repository is required")
	ErrFactoryAgentResourceRepositoryInvalid     = errors.New("repository must use the owner/name format")
	ErrFactoryAgentResourceRepositoryTaken       = errors.New("this repository is already added")
	ErrFactoryAgentResourceContextRepoCapReached = errors.New("workspace already has the maximum number of context repositories")
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
	SystemSkill         *FactorySystemSkill          `json:"systemSkill,omitempty"`
	Transport           string                       `json:"transport,omitempty"`
	URL                 string                       `json:"url,omitempty"`
	Auth                string                       `json:"auth,omitempty"`
	Headers             []FactoryAgentResourceHeader `json:"headers,omitempty"`
	Source              string                       `json:"source,omitempty"`
	Repository          string                       `json:"repository,omitempty"`
	Ref                 string                       `json:"ref,omitempty"`
	Path                string                       `json:"path,omitempty"`
	Markdown            string                       `json:"markdown,omitempty"`
	Commands            []string                     `json:"commands,omitempty"`
	VCSProvider         string                       `json:"vcsProvider,omitempty"`
	DisabledTools       []string                     `json:"disabledTools,omitempty"`
	ToolsDefaultApplied bool                         `json:"toolsDefaultApplied,omitempty"`
}

// FactorySystemSkill records the supplied template separately from editable content.
// It is internal metadata; public resource mutation requests cannot set it.
type FactorySystemSkill struct {
	TemplateID  string `json:"templateId"`
	Version     int32  `json:"version"`
	ContentHash string `json:"contentHash"`
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

func (r *FactoryAgentResource) IsSystemSkill() bool {
	return r.Kind == FactoryAgentResourceKindSkill && r.Config.Data().SystemSkill != nil
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
	if name == FactoryAgentResourceSetupName {
		return ErrFactoryAgentResourceNameSetupReserved
	}
	return nil
}

func ValidateFactoryAgentResourceKind(kind string) error {
	switch kind {
	case FactoryAgentResourceKindMCPServer,
		FactoryAgentResourceKindSkill,
		FactoryAgentResourceKindSetup,
		FactoryAgentResourceKindContextRepo:
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
	if c.SystemSkill != nil && (strings.TrimSpace(c.SystemSkill.TemplateID) == "" || c.SystemSkill.Version < 1) {
		return ErrFactorySystemSkillMetadataInvalid
	}

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

func NormalizeSetupCommands(commands []string) []string {
	out := make([]string, 0, len(commands))
	for _, command := range commands {
		trimmed := strings.TrimSpace(command)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func ParseSetupCommandsText(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	return NormalizeSetupCommands(lines)
}

func SetupCommandsText(commands []string) string {
	return strings.Join(NormalizeSetupCommands(commands), "\n")
}

func (c FactoryAgentResourceConfig) ValidateSetup() error {
	commands := NormalizeSetupCommands(c.Commands)
	if len(commands) == 0 {
		return ErrFactoryAgentResourceCommandsRequired
	}
	if len(commands) > MaxFactorySetupCommands {
		return ErrFactoryAgentResourceTooManyCommands
	}
	total := 0
	for _, command := range commands {
		if len(command) > MaxFactorySetupCommandLineBytes {
			return ErrFactoryAgentResourceCommandTooLarge
		}
		total += len(command)
	}
	if total > MaxFactorySetupCommandBytes {
		return ErrFactoryAgentResourceCommandTooLarge
	}
	return nil
}

func (c FactoryAgentResourceConfig) NormalizedSetup() FactoryAgentResourceConfig {
	c.Commands = NormalizeSetupCommands(c.Commands)
	c.Transport = ""
	c.URL = ""
	c.Auth = ""
	c.Headers = nil
	c.Source = ""
	c.Repository = ""
	c.Ref = ""
	c.Path = ""
	c.Markdown = ""
	c.VCSProvider = ""
	c.DisabledTools = nil
	c.ToolsDefaultApplied = false
	return c
}

func NormalizeContextRepoName(repository string) string {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return ""
	}
	name := strings.ToLower(repository)
	name = strings.ReplaceAll(name, "/", "-")
	name = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(name, "-")
	name = regexp.MustCompile(`-+`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return "repo"
	}
	if name[0] >= '0' && name[0] <= '9' {
		name = "repo-" + name
	}
	if len(name) > 63 {
		name = name[:63]
		name = strings.Trim(name, "-")
	}
	if name == "" {
		return "repo"
	}
	return name
}

func CanonicalContextRepo(repository string) string {
	return strings.ToLower(strings.TrimSpace(repository))
}

// uniqueFactoryAgentResourceName returns base when no other resource in the
// factory uses it, otherwise the first free suffixed variant (base-2,
// base-3, ...) that fits the 63-character name limit.
func (f *Factory) uniqueFactoryAgentResourceName(tx *gorm.DB, base string, exceptID uuid.UUID) (string, error) {
	candidate := base
	for i := 2; ; i++ {
		var count int64
		query := tx.Model(&FactoryAgentResource{}).Where(
			"organization_id = ? AND factory_id = ? AND name = ?",
			f.OrganizationID, f.ID, candidate,
		)
		if exceptID != uuid.Nil {
			query = query.Where("id <> ?", exceptID)
		}
		if err := query.Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		if i > 99 {
			return "", ErrFactoryAgentResourceNameTaken
		}
		suffix := "-" + strconv.Itoa(i)
		trimmed := base
		if len(trimmed) > 63-len(suffix) {
			trimmed = strings.Trim(trimmed[:63-len(suffix)], "-")
		}
		candidate = trimmed + suffix
	}
}

func (c FactoryAgentResourceConfig) ValidateContextRepo() error {
	repository := strings.TrimSpace(c.Repository)
	if repository == "" {
		return ErrFactoryAgentResourceRepositoryRequired
	}
	if !factoryOnboardingRepositoryPattern.MatchString(repository) {
		return ErrFactoryAgentResourceRepositoryInvalid
	}
	if provider := strings.TrimSpace(c.VCSProvider); provider != "" {
		if err := ValidateFactoryOnboardingVCSProvider(provider); err != nil {
			return err
		}
	}
	return nil
}

func (c FactoryAgentResourceConfig) NormalizedContextRepo() FactoryAgentResourceConfig {
	c.Repository = strings.TrimSpace(c.Repository)
	c.VCSProvider = strings.TrimSpace(c.VCSProvider)
	if c.VCSProvider == "" {
		c.VCSProvider = ProviderGitHub
	}
	c.Transport = ""
	c.URL = ""
	c.Auth = ""
	c.Headers = nil
	c.Source = ""
	c.Ref = ""
	c.Path = ""
	c.Markdown = ""
	c.Commands = nil
	c.DisabledTools = nil
	c.ToolsDefaultApplied = false
	return c
}

func CanonicalMCPServerURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.ToLower(strings.TrimRight(trimmed, "/"))
	}
	parsed.Fragment = ""
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	if strings.HasSuffix(parsed.Path, "/") {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}
	return parsed.String()
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

func (r *FactoryAgentResource) MCPConnectionEstablished() bool {
	if r.Kind != FactoryAgentResourceKindMCPServer {
		return false
	}
	if r.Config.Data().MCPAuth() != FactoryAgentResourceAuthOAuth {
		return true
	}
	return r.OAuthState() == FactoryAgentResourceOAuthConnected
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
	if config.SystemSkill != nil && (kind != FactoryAgentResourceKindSkill || !enabled) {
		return nil, ErrFactorySystemSkillMetadataInvalid
	}
	if err := ValidateFactoryAgentResourceKind(kind); err != nil {
		return nil, err
	}
	switch kind {
	case FactoryAgentResourceKindSkill:
		name = NormalizeFactoryAgentResourceName(name)
		if err := ValidateFactoryAgentResourceName(name); err != nil {
			return nil, err
		}
		if err := config.ValidateSkill(); err != nil {
			return nil, err
		}
		config = config.NormalizedSkill()
		if config.SystemSkill != nil {
			metadata := *config.SystemSkill
			metadata.TemplateID = strings.TrimSpace(metadata.TemplateID)
			metadata.ContentHash = fmt.Sprintf("%x", sha256.Sum256([]byte(config.Markdown)))
			config.SystemSkill = &metadata
		}
	case FactoryAgentResourceKindSetup:
		name = FactoryAgentResourceSetupName
		if err := config.ValidateSetup(); err != nil {
			return nil, err
		}
		config = config.NormalizedSetup()
	case FactoryAgentResourceKindContextRepo:
		if err := config.ValidateContextRepo(); err != nil {
			return nil, err
		}
		config = config.NormalizedContextRepo()
		if strings.TrimSpace(name) == "" {
			name = NormalizeContextRepoName(config.Repository)
			unique, err := f.uniqueFactoryAgentResourceName(tx, name, uuid.Nil)
			if err != nil {
				return nil, err
			}
			name = unique
		} else {
			name = NormalizeFactoryAgentResourceName(name)
		}
		if err := ValidateFactoryAgentResourceName(name); err != nil {
			return nil, err
		}
	default:
		name = NormalizeFactoryAgentResourceName(name)
		if err := ValidateFactoryAgentResourceName(name); err != nil {
			return nil, err
		}
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
		if enabled && kind == FactoryAgentResourceKindContextRepo {
			if err := f.ensureContextRepoCapacity(inner, uuid.Nil); err != nil {
				return err
			}
		}
		if kind == FactoryAgentResourceKindMCPServer {
			if err := f.ensureUniqueConnectedMCPURL(inner, uuid.Nil, config.URL); err != nil {
				return err
			}
		}
		if kind == FactoryAgentResourceKindContextRepo {
			if err := f.ensureUniqueContextRepo(inner, uuid.Nil, config.Repository); err != nil {
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

func (f *Factory) FindSetupCommands(tx *gorm.DB) (*FactoryAgentResource, error) {
	var resource FactoryAgentResource
	err := tx.Where(
		"organization_id = ? AND factory_id = ? AND kind = ?",
		f.OrganizationID,
		f.ID,
		FactoryAgentResourceKindSetup,
	).First(&resource).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFactoryAgentResourceNotFound
	}
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

func (f *Factory) ListEnabledSetupCommands(tx *gorm.DB) ([]string, error) {
	resource, err := f.FindSetupCommands(tx)
	if err != nil {
		if errors.Is(err, ErrFactoryAgentResourceNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if !resource.Enabled {
		return nil, nil
	}
	return NormalizeSetupCommands(resource.Config.Data().Commands), nil
}

func (f *Factory) ListContextRepos(tx *gorm.DB) ([]FactoryAgentResource, error) {
	var resources []FactoryAgentResource
	err := tx.Where(
		"organization_id = ? AND factory_id = ? AND kind = ?",
		f.OrganizationID,
		f.ID,
		FactoryAgentResourceKindContextRepo,
	).Order("name asc").Find(&resources).Error
	if err != nil {
		return nil, err
	}
	return resources, nil
}

func (f *Factory) ListEnabledContextRepos(tx *gorm.DB) ([]FactoryAgentResource, error) {
	var resources []FactoryAgentResource
	err := tx.Where(
		"organization_id = ? AND factory_id = ? AND kind = ? AND enabled = ?",
		f.OrganizationID,
		f.ID,
		FactoryAgentResourceKindContextRepo,
		true,
	).Order("name asc").Find(&resources).Error
	if err != nil {
		return nil, err
	}
	return resources, nil
}

func (r *FactoryAgentResource) Update(tx *gorm.DB, name *string, enabled *bool, config *FactoryAgentResourceConfig) error {
	if r.IsSystemSkill() && ((name != nil && NormalizeFactoryAgentResourceName(*name) != r.Name) || (enabled != nil && !*enabled)) {
		return ErrFactorySystemSkillProtected
	}
	return tx.Transaction(func(inner *gorm.DB) error {
		updates := map[string]any{
			"updated_at": time.Now(),
		}
		if name != nil {
			if r.Kind == FactoryAgentResourceKindSetup {
				// The setup singleton keeps a fixed name.
				if normalized := NormalizeFactoryAgentResourceName(*name); normalized != FactoryAgentResourceSetupName && normalized != r.Name {
					return ErrFactoryAgentResourceNameInvalid
				}
			} else {
				normalized := NormalizeFactoryAgentResourceName(*name)
				if err := ValidateFactoryAgentResourceName(normalized); err != nil {
					return err
				}
				updates["name"] = normalized
				r.Name = normalized
			}
		}
		if enabled != nil {
			if *enabled && r.Kind == FactoryAgentResourceKindMCPServer {
				factory := &Factory{ID: r.FactoryID, OrganizationID: r.OrganizationID}
				if err := factory.ensureEnabledMCPCapacity(inner, r.ID); err != nil {
					return err
				}
			}
			if *enabled && r.Kind == FactoryAgentResourceKindContextRepo {
				factory := &Factory{ID: r.FactoryID, OrganizationID: r.OrganizationID}
				if err := factory.ensureContextRepoCapacity(inner, r.ID); err != nil {
					return err
				}
			}
			updates["enabled"] = *enabled
			r.Enabled = *enabled
		}
		if config != nil {
			if r.Kind == FactoryAgentResourceKindSkill {
				next := *config
				// Content edits cannot change the template identity or its baseline.
				next.SystemSkill = r.Config.Data().SystemSkill
				config = &next
				if err := config.ValidateSkill(); err != nil {
					return err
				}
				normalized := config.NormalizedSkill()
				config = &normalized
			} else if r.Kind == FactoryAgentResourceKindSetup {
				if err := config.ValidateSetup(); err != nil {
					return err
				}
				normalized := config.NormalizedSetup()
				config = &normalized
			} else if r.Kind == FactoryAgentResourceKindContextRepo {
				if err := config.ValidateContextRepo(); err != nil {
					return err
				}
				normalized := config.NormalizedContextRepo()
				config = &normalized
				factory := &Factory{ID: r.FactoryID, OrganizationID: r.OrganizationID}
				if err := factory.ensureUniqueContextRepo(inner, r.ID, config.Repository); err != nil {
					return err
				}
				// Keep the derived name in sync when the repository changes and the
				// caller did not provide an explicit name override.
				if name == nil {
					derived := NormalizeContextRepoName(config.Repository)
					if derived != "" && derived != r.Name {
						unique, err := factory.uniqueFactoryAgentResourceName(inner, derived, r.ID)
						if err != nil {
							return err
						}
						updates["name"] = unique
						r.Name = unique
					}
				}
			} else if err := config.ValidateMCP(); err != nil {
				return err
			} else {
				normalized := config.NormalizedMCP()
				config = &normalized
				factory := &Factory{ID: r.FactoryID, OrganizationID: r.OrganizationID}
				if err := factory.ensureUniqueConnectedMCPURL(inner, r.ID, config.URL); err != nil {
					return err
				}
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
	if r.IsSystemSkill() {
		return ErrFactorySystemSkillProtected
	}
	if err := tx.Where("resource_id = ?", r.ID).Delete(&FactoryAgentResourceSecret{}).Error; err != nil {
		return err
	}
	return tx.Delete(r).Error
}

func (r *FactoryAgentResource) SetOAuthStatus(tx *gorm.DB, status, message string, connectedBy *uuid.UUID) error {
	return tx.Transaction(func(inner *gorm.DB) error {
		if status == FactoryAgentResourceOAuthConnected {
			factory := &Factory{ID: r.FactoryID, OrganizationID: r.OrganizationID}
			if err := factory.ensureUniqueConnectedMCPURL(inner, r.ID, r.Config.Data().URL); err != nil {
				return err
			}
		}
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
		return inner.Model(r).Updates(updates).Error
	})
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

func (f *Factory) ensureUniqueConnectedMCPURL(tx *gorm.DB, exceptID uuid.UUID, rawURL string) error {
	canonical := CanonicalMCPServerURL(rawURL)
	if canonical == "" {
		return nil
	}
	if err := f.lockAgentResourceCapacity(tx); err != nil {
		return err
	}
	resources, err := f.ListAgentResources(tx, FactoryAgentResourceKindMCPServer)
	if err != nil {
		return err
	}
	for i := range resources {
		resource := &resources[i]
		if exceptID != uuid.Nil && resource.ID == exceptID {
			continue
		}
		if !resource.MCPConnectionEstablished() {
			continue
		}
		if CanonicalMCPServerURL(resource.Config.Data().URL) == canonical {
			return ErrFactoryAgentResourceURLTaken
		}
	}
	return nil
}

func (f *Factory) ensureContextRepoCapacity(tx *gorm.DB, exceptID uuid.UUID) error {
	if err := f.lockAgentResourceCapacity(tx); err != nil {
		return err
	}
	query := tx.Model(&FactoryAgentResource{}).Where(
		"factory_id = ? AND kind = ? AND enabled = ?",
		f.ID,
		FactoryAgentResourceKindContextRepo,
		true,
	)
	if exceptID != uuid.Nil {
		query = query.Where("id <> ?", exceptID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count >= MaxEnabledFactoryContextRepos {
		return fmt.Errorf("%w", ErrFactoryAgentResourceContextRepoCapReached)
	}
	return nil
}

func (f *Factory) ensureUniqueContextRepo(tx *gorm.DB, exceptID uuid.UUID, repository string) error {
	canonical := CanonicalContextRepo(repository)
	if canonical == "" {
		return nil
	}
	if err := f.lockAgentResourceCapacity(tx); err != nil {
		return err
	}
	resources, err := f.ListAgentResources(tx, FactoryAgentResourceKindContextRepo)
	if err != nil {
		return err
	}
	for i := range resources {
		resource := &resources[i]
		if exceptID != uuid.Nil && resource.ID == exceptID {
			continue
		}
		if CanonicalContextRepo(resource.Config.Data().Repository) == canonical {
			return ErrFactoryAgentResourceRepositoryTaken
		}
	}
	return nil
}
