package enterprise

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
)

// SSOAccessSession is the account-cookie data enforcement and sign-out need.
type SSOAccessSession struct {
	AccountID uuid.UUID
	SSO       string
}

// SSORequired tells the caller to send the browser to the identity provider.
// A nil result from CheckOrganizationAccess means access is allowed.
type SSORequired struct {
	StartURL string
}

// SSO is the organization Single Sign-On capability. Core calls this
// interface. The Enterprise implementation lives in ee/sso. Sign-in,
// enforcement, sign-out, and deprovision of an existing connection do not
// require a license. Operations that create or expand access do.
type SSO interface {
	AuthHandler() http.Handler
	SCIMHandler() http.Handler

	CheckOrganizationAccess(ctx context.Context, user *models.User, session SSOAccessSession) (*SSORequired, error)
	LogoutRedirect(ctx context.Context, accountID uuid.UUID, session SSOAccessSession) (string, bool)
	ActiveConnectionExists(ctx context.Context) (bool, error)

	GetConnection(ctx context.Context, orgID uuid.UUID) (*SSOConnection, error)
	PutConnection(ctx context.Context, orgID, accountID uuid.UUID, input SSOConnectionInput) (*SSOConnection, error)
	ActivateConnection(ctx context.Context, orgID uuid.UUID) (*SSOConnection, error)
	DisableConnection(ctx context.Context, orgID uuid.UUID) (*SSOConnection, error)
	DeleteConnection(ctx context.Context, orgID uuid.UUID) error

	ListDomains(ctx context.Context, orgID uuid.UUID) ([]SSODomain, error)
	CreateDomain(ctx context.Context, orgID, accountID uuid.UUID, domain string) (*SSODomain, error)
	VerifyDomain(ctx context.Context, orgID, domainID uuid.UUID) (*SSODomain, error)
	DeleteDomain(ctx context.Context, orgID, domainID uuid.UUID) error

	ListSCIMTokens(ctx context.Context, orgID uuid.UUID) ([]SCIMToken, error)
	CreateSCIMToken(ctx context.Context, orgID, accountID uuid.UUID, description string) (*CreatedSCIMToken, error)
	RevokeSCIMToken(ctx context.Context, orgID, tokenID uuid.UUID) error

	AdminGet(ctx context.Context, orgID uuid.UUID) (*AdminSSO, error)
	AdminDisable(ctx context.Context, orgID uuid.UUID) (*AdminSSO, error)
	AdminVerifyDomain(ctx context.Context, orgID, domainID, accountID uuid.UUID) (*SSODomain, error)
}

// SSOConnection is the organization view of one identity-provider connection.
// It never includes the client secret.
type SSOConnection struct {
	Exists                bool
	ID                    uuid.UUID
	OrganizationID        uuid.UUID
	Protocol              string
	Preset                string
	DisplayName           string
	Status                string
	Enforcement           string
	AllowOwnerBypass      bool
	JITEnabled            bool
	DefaultRole           string
	MaxSessionAgeSeconds  int32
	IDPLogoutEnabled      bool
	Issuer                string
	ClientID              string
	Scopes                []string
	SubjectClaim          string
	EmailClaim            string
	NameClaim             string
	ClientSecretSet       bool
	LastTestedAt          *time.Time
	LastTest              *SSOTestResult
	RedirectURI           string
	InitiateLoginURI      string
	PostLogoutRedirectURI string
	BackChannelLogoutURI  string
	SCIMBaseURL           string
}

// SSOConnectionInput is a full replacement of the editable connection fields.
// An empty client secret keeps the stored secret.
type SSOConnectionInput struct {
	Preset               string
	DisplayName          string
	Issuer               string
	ClientID             string
	ClientSecret         string
	Scopes               []string
	SubjectClaim         string
	EmailClaim           string
	NameClaim            string
	JITEnabled           bool
	DefaultRole          string
	Enforcement          string
	AllowOwnerBypass     bool
	MaxSessionAgeSeconds int32
	IDPLogoutEnabled     bool
}

// SSOTestResult is the last test sign-in. It never contains tokens.
type SSOTestResult struct {
	OK      bool
	Error   string
	Subject string
	Email   string
	Name    string
}

// SSODomain is one organization domain and its verification record.
type SSODomain struct {
	ID                 uuid.UUID
	Domain             string
	VerificationToken  string
	TXTName            string
	TXTValue           string
	VerifiedAt         *time.Time
	VerificationMethod string
}

// SCIMToken is a stored provisioning token. The secret is not included.
type SCIMToken struct {
	ID          uuid.UUID
	Description string
	CreatedAt   time.Time
	LastUsedAt  *time.Time
}

// CreatedSCIMToken is returned once, when the token is created.
type CreatedSCIMToken struct {
	Token  SCIMToken
	Secret string
}

// AdminSSO is the installation-admin view of one organization.
type AdminSSO struct {
	Connection AdminSSOConnection
	Domains    []AdminSSODomain
}

// AdminSSOConnection is the connection summary shown to installation admins.
type AdminSSOConnection struct {
	Exists      bool
	ID          uuid.UUID
	Status      string
	Enforcement string
	DisplayName string
	Protocol    string
}

// AdminSSODomain is one domain row in the installation-admin view.
type AdminSSODomain struct {
	ID                 uuid.UUID
	Domain             string
	Verified           bool
	VerificationMethod string
}

type communitySSO struct{}

func (communitySSO) AuthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
}

func (communitySSO) SCIMHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
}

func (communitySSO) CheckOrganizationAccess(context.Context, *models.User, SSOAccessSession) (*SSORequired, error) {
	return nil, nil
}

func (communitySSO) LogoutRedirect(context.Context, uuid.UUID, SSOAccessSession) (string, bool) {
	return "", false
}

func (communitySSO) ActiveConnectionExists(context.Context) (bool, error) {
	return false, nil
}

func (communitySSO) GetConnection(context.Context, uuid.UUID) (*SSOConnection, error) {
	return &SSOConnection{}, nil
}

func (communitySSO) PutConnection(context.Context, uuid.UUID, uuid.UUID, SSOConnectionInput) (*SSOConnection, error) {
	return nil, notLicensed()
}

func (communitySSO) ActivateConnection(context.Context, uuid.UUID) (*SSOConnection, error) {
	return nil, notLicensed()
}

func (communitySSO) DisableConnection(context.Context, uuid.UUID) (*SSOConnection, error) {
	return nil, notLicensed()
}

func (communitySSO) DeleteConnection(context.Context, uuid.UUID) error {
	return notLicensed()
}

func (communitySSO) ListDomains(context.Context, uuid.UUID) ([]SSODomain, error) {
	return []SSODomain{}, nil
}

func (communitySSO) CreateDomain(context.Context, uuid.UUID, uuid.UUID, string) (*SSODomain, error) {
	return nil, notLicensed()
}

func (communitySSO) VerifyDomain(context.Context, uuid.UUID, uuid.UUID) (*SSODomain, error) {
	return nil, notLicensed()
}

func (communitySSO) DeleteDomain(context.Context, uuid.UUID, uuid.UUID) error {
	return notLicensed()
}

func (communitySSO) ListSCIMTokens(context.Context, uuid.UUID) ([]SCIMToken, error) {
	return []SCIMToken{}, nil
}

func (communitySSO) CreateSCIMToken(context.Context, uuid.UUID, uuid.UUID, string) (*CreatedSCIMToken, error) {
	return nil, notLicensed()
}

func (communitySSO) RevokeSCIMToken(context.Context, uuid.UUID, uuid.UUID) error {
	return notLicensed()
}

func (communitySSO) AdminGet(context.Context, uuid.UUID) (*AdminSSO, error) {
	return &AdminSSO{}, nil
}

func (communitySSO) AdminDisable(context.Context, uuid.UUID) (*AdminSSO, error) {
	return nil, notLicensed()
}

func (communitySSO) AdminVerifyDomain(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*SSODomain, error) {
	return nil, notLicensed()
}
