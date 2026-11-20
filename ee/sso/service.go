package sso

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/ee/sso/core"
	"github.com/superplanehq/superplane/ee/sso/oidc"
	"github.com/superplanehq/superplane/ee/sso/scim"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func invalid(message string) error {
	return status.Error(codes.InvalidArgument, message)
}

func conflict(message string) error {
	return status.Error(codes.AlreadyExists, message)
}

func notFound(message string) error {
	return status.Error(codes.NotFound, message)
}

func precondition(message string) error {
	return status.Error(codes.FailedPrecondition, message)
}

// TXTLookup reads DNS TXT records. Tests supply a fake.
type TXTLookup interface {
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

type netLookup struct{}

func (netLookup) LookupTXT(ctx context.Context, name string) ([]string, error) {
	resolver := net.Resolver{}
	return resolver.LookupTXT(ctx, name)
}

// Config builds the Enterprise SSO service.
type Config struct {
	Encryptor    crypto.Encryptor
	Entitlements licensing.Entitlements
	Auth         authorization.Authorization
	Signer       *jwt.Signer
	HTTP         *registry.HTTPContext
	BaseURL      string
	Lookup       TXTLookup
	Now          func() time.Time
}

// Service is the Enterprise implementation of organization SSO.
type Service struct {
	Encryptor          crypto.Encryptor
	Entitlements       licensing.Entitlements
	Auth               authorization.Authorization
	Signer             *jwt.Signer
	HTTP               *registry.HTTPContext
	BaseURL            string
	Lookup             TXTLookup
	Now                func() time.Time
	adapter            *oidc.Adapter
	limits             *ipLimiter
	beforeIdentityLock func(tx *gorm.DB, id uuid.UUID)
}

// New returns the SSO service that pkg/server registers.
func New(cfg Config) *Service {
	if cfg.Lookup == nil {
		cfg.Lookup = netLookup{}
	}
	service := &Service{
		Encryptor:    cfg.Encryptor,
		Entitlements: cfg.Entitlements,
		Auth:         cfg.Auth,
		Signer:       cfg.Signer,
		HTTP:         cfg.HTTP,
		BaseURL:      strings.TrimRight(cfg.BaseURL, "/"),
		Lookup:       cfg.Lookup,
		Now:          cfg.Now,
		limits:       newIPLimiter(),
	}
	service.adapter = &oidc.Adapter{
		BaseURL:   service.BaseURL,
		Doer:      cfg.HTTP,
		Encryptor: cfg.Encryptor,
		AllowHTTP: service.allowHTTP,
		Now:       cfg.Now,
	}
	return service
}

func (s *Service) SCIMHandler() http.Handler {
	return scim.NewHandler(s.pipeline(), s.BaseURL)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) pipeline() *core.Pipeline {
	return &core.Pipeline{
		Encryptor:          s.Encryptor,
		Entitlements:       s.Entitlements,
		Auth:               s.Auth,
		Now:                s.Now,
		BeforeIdentityLock: s.beforeIdentityLock,
	}
}

func (s *Service) allowHTTP(ctx context.Context) bool {
	metadata, err := models.GetInstallationMetadata(database.DB(ctx))
	if err != nil || metadata == nil {
		return false
	}
	return metadata.AllowPrivateNetworkAccess
}

func (s *Service) requireLicense(ctx context.Context, orgID uuid.UUID) error {
	if err := licensing.Require(ctx, s.Entitlements, orgID, licensing.FeatureSSO); err != nil {
		return grpcerrors.PermissionDenied(err, licensing.ErrNotLicensed.Error())
	}
	return nil
}
