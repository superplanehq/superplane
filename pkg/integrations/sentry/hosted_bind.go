package sentry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const unclaimedHostedInstallTTL = 20 * time.Minute

// hostedSentryInstall is a SuperPlane-side copy of a public Sentry app
// install. SuperPlane reuses a ready connection in the same organization,
// or a webhook grant that the authenticated setup callback later claims.
type hostedSentryInstall struct {
	InstallationUUID string
	AccessToken      string
	RefreshToken     string
	TokenExpiresAt   string
	Organization     *OrganizationSummary
	Projects         []ProjectSummary
	Teams            []TeamSummary
	// Code is the one-time Sentry grant from installation.created or the
	// Redirect URL. Claiming this row requires the same code, so a known
	// installation UUID is not enough to steal the tokens.
	Code      string
	ExpiresAt time.Time
}

type hostedSentryInstallFinder func(organizationID, excludeID string) (*hostedSentryInstall, error)

var (
	hostedInstallEncryptor       crypto.Encryptor
	findReadyHostedSentryInstall hostedSentryInstallFinder = noopReadyHostedSentryInstall
	unclaimedHostedMu            sync.Mutex
	unclaimedHostedInstalls      = map[string]hostedSentryInstall{}
)

func noopReadyHostedSentryInstall(organizationID, excludeID string) (*hostedSentryInstall, error) {
	return nil, nil
}

// EnableHostedInstallBind lets Sync reuse a ready hosted Sentry connection in
// the same SuperPlane organization instead of opening Sentry again.
func EnableHostedInstallBind(encryptor crypto.Encryptor) {
	if encryptor == nil {
		return
	}
	hostedInstallEncryptor = encryptor
	findReadyHostedSentryInstall = lookupReadyHostedSentryInstall
}

func (s *Sentry) redirectHostedAppInstall(ctx core.HTTPRequestContext) {
	app, ok := HostedAppFromEnv()
	if !ok {
		http.Error(ctx.Response, "hosted Sentry app is not configured", http.StatusNotFound)
		return
	}

	metadata, ok := decodeHostedMetadata(ctx)
	if !ok {
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	bound, err := s.bindReadyHostedInstallIfPresent(core.SyncContext{
		Logger:          ctx.Logger,
		HTTP:            ctx.HTTP,
		Integration:     ctx.Integration,
		BaseURL:         ctx.BaseURL,
		WebhooksBaseURL: ctx.WebhooksBaseURL,
		OrganizationID:  ctx.OrganizationID,
	}, metadata)
	if err != nil {
		ctx.Logger.Errorf("failed to bind existing Sentry install: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}
	if bound {
		redirectToSetupReturn(ctx)
		return
	}

	http.Redirect(ctx.Response, ctx.Request, HostedAppExternalInstallURL(app.Slug), http.StatusSeeOther)
}

func (s *Sentry) bindReadyHostedInstallIfPresent(ctx core.SyncContext, pending Metadata) (bool, error) {
	install, err := s.findBindableHostedInstall(ctx)
	if err != nil {
		return false, err
	}
	if install == nil {
		return false, nil
	}

	if canCopyHostedInstall(*install, ctx.HTTP != nil) {
		if err := s.bindHostedInstallation(ctx, pending, *install); err != nil {
			return false, err
		}
		return true, nil
	}

	if err := s.adoptKnownHostedInstall(ctx, pending, *install); err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Errorf("failed to adopt known Sentry install: %v", err)
		}
		return false, nil
	}
	return true, nil
}

func (s *Sentry) findBindableHostedInstall(ctx core.SyncContext) (*hostedSentryInstall, error) {
	if ctx.OrganizationID == "" {
		return nil, nil
	}

	install, err := findReadyHostedSentryInstall(ctx.OrganizationID, ctx.Integration.ID().String())
	if err != nil {
		return nil, err
	}
	if install == nil || strings.TrimSpace(install.InstallationUUID) == "" || strings.TrimSpace(install.AccessToken) == "" {
		return nil, nil
	}
	return install, nil
}

func canCopyHostedInstall(install hostedSentryInstall, hasHTTP bool) bool {
	if strings.TrimSpace(install.AccessToken) == "" {
		return false
	}
	if install.Organization == nil || strings.TrimSpace(install.Organization.Slug) == "" {
		return false
	}
	return len(install.Projects) > 0 || !hasHTTP
}

func (s *Sentry) bindHostedInstallation(ctx core.SyncContext, pending Metadata, install hostedSentryInstall) error {
	if err := ctx.Integration.SetSecret(SecretAccessToken, []byte(install.AccessToken)); err != nil {
		return err
	}
	if install.RefreshToken != "" {
		if err := ctx.Integration.SetSecret(SecretRefreshToken, []byte(install.RefreshToken)); err != nil {
			return err
		}
	}

	pending.HostedApp = true
	pending.InstallationUUID = install.InstallationUUID
	pending.TokenExpiresAt = install.TokenExpiresAt
	pending.Organization = install.Organization
	pending.Projects = install.Projects
	pending.Teams = install.Teams
	ctx.Integration.SetMetadata(pending)
	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()
	return nil
}

func (s *Sentry) adoptKnownHostedInstall(ctx core.SyncContext, pending Metadata, install hostedSentryInstall) error {
	app, ok := HostedAppFromEnv()
	if !ok {
		return fmt.Errorf("hosted Sentry app is not configured")
	}

	orgSlug := ""
	if install.Organization != nil {
		orgSlug = install.Organization.Slug
	}
	tokens, err := tokensForKnownHostedInstall(ctx.HTTP, app, install)
	if err != nil {
		return err
	}
	return s.adoptHostedInstall(ctx, pending, tokens, install.InstallationUUID, orgSlug)
}

func rememberUnclaimedHostedInstall(install hostedSentryInstall) {
	install.InstallationUUID = strings.TrimSpace(install.InstallationUUID)
	install.Code = strings.TrimSpace(install.Code)
	if install.InstallationUUID == "" || install.Code == "" {
		return
	}
	if install.ExpiresAt.IsZero() {
		install.ExpiresAt = time.Now().Add(unclaimedHostedInstallTTL)
	}

	unclaimedHostedMu.Lock()
	defer unclaimedHostedMu.Unlock()
	purgeExpiredUnclaimedHostedInstallsLocked(time.Now())
	unclaimedHostedInstalls[install.InstallationUUID] = install
}

// ForgetKnownHostedInstallation drops a Sentry install after Sentry reports
// that the organization uninstalled the app.
func ForgetKnownHostedInstallation(installationUUID string) {
	installationUUID = strings.TrimSpace(installationUUID)
	if installationUUID == "" {
		return
	}

	unclaimedHostedMu.Lock()
	defer unclaimedHostedMu.Unlock()
	delete(unclaimedHostedInstalls, installationUUID)
}

func takeUnclaimedHostedInstall(installationUUID, code string) *hostedSentryInstall {
	installationUUID = strings.TrimSpace(installationUUID)
	code = strings.TrimSpace(code)
	if installationUUID == "" || code == "" {
		return nil
	}

	unclaimedHostedMu.Lock()
	defer unclaimedHostedMu.Unlock()
	now := time.Now()
	purgeExpiredUnclaimedHostedInstallsLocked(now)

	install, ok := unclaimedHostedInstalls[installationUUID]
	if !ok {
		return nil
	}
	if !install.ExpiresAt.IsZero() && !install.ExpiresAt.After(now) {
		delete(unclaimedHostedInstalls, installationUUID)
		return nil
	}
	if strings.TrimSpace(install.Code) != code {
		return nil
	}
	delete(unclaimedHostedInstalls, installationUUID)
	copied := install
	return &copied
}

func purgeExpiredUnclaimedHostedInstallsLocked(now time.Time) {
	for uuid, install := range unclaimedHostedInstalls {
		if !install.ExpiresAt.IsZero() && !install.ExpiresAt.After(now) {
			delete(unclaimedHostedInstalls, uuid)
		}
	}
}

func resetUnclaimedHostedInstalls() {
	unclaimedHostedMu.Lock()
	defer unclaimedHostedMu.Unlock()
	unclaimedHostedInstalls = map[string]hostedSentryInstall{}
}

func lookupReadyHostedSentryInstall(organizationID, excludeID string) (*hostedSentryInstall, error) {
	if hostedInstallEncryptor == nil {
		return nil, nil
	}

	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, nil
	}

	exclude := uuid.Nil
	if excludeID != "" {
		parsed, parseErr := uuid.Parse(excludeID)
		if parseErr == nil {
			exclude = parsed
		}
	}

	tx := database.Conn()
	source, err := models.FindReadyHostedSentryIntegration(tx, orgID, exclude)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return hostedSentryInstallFromIntegration(tx, hostedInstallEncryptor, source)
}

func hostedSentryInstallFromIntegration(
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	source *models.Integration,
) (*hostedSentryInstall, error) {
	if source == nil {
		return nil, nil
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(source.Metadata.Data(), &metadata); err != nil {
		return nil, err
	}
	if !metadata.HostedApp || strings.TrimSpace(metadata.InstallationUUID) == "" {
		return nil, nil
	}

	secrets, err := models.ListIntegrationSecretsInTransaction(tx, source.ID)
	if err != nil {
		return nil, err
	}

	install := &hostedSentryInstall{
		InstallationUUID: metadata.InstallationUUID,
		TokenExpiresAt:   metadata.TokenExpiresAt,
		Organization:     metadata.Organization,
		Projects:         metadata.Projects,
		Teams:            metadata.Teams,
	}
	for _, secret := range secrets {
		plain, err := encryptor.Decrypt(context.Background(), secret.Value, []byte(source.ID.String()))
		if err != nil {
			return nil, err
		}
		switch secret.Name {
		case SecretAccessToken:
			install.AccessToken = strings.TrimSpace(string(plain))
		case SecretRefreshToken:
			install.RefreshToken = strings.TrimSpace(string(plain))
		}
	}
	if install.AccessToken == "" {
		return nil, nil
	}
	return install, nil
}
