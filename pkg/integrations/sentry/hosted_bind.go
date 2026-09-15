package sentry

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// hostedSentryInstall is a SuperPlane-side copy of a public Sentry app
// install. Sentry does not mint tokens from the app credentials, so a new
// connection reuses tokens that SuperPlane already stored.
type hostedSentryInstall struct {
	InstallationUUID string
	AccessToken      string
	RefreshToken     string
	TokenExpiresAt   string
	Organization     *OrganizationSummary
	Projects         []ProjectSummary
	Teams            []TeamSummary
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
	if ctx.OrganizationID == "" {
		return false, nil
	}

	install, err := findReadyHostedSentryInstall(ctx.OrganizationID, ctx.Integration.ID().String())
	if err != nil {
		return false, err
	}
	if install == nil || strings.TrimSpace(install.InstallationUUID) == "" || strings.TrimSpace(install.AccessToken) == "" {
		return false, nil
	}

	if err := s.bindHostedInstallation(ctx, pending, *install); err != nil {
		return false, err
	}
	return true, nil
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

func rememberUnclaimedHostedInstall(install hostedSentryInstall) {
	if strings.TrimSpace(install.InstallationUUID) == "" || strings.TrimSpace(install.AccessToken) == "" {
		return
	}

	unclaimedHostedMu.Lock()
	defer unclaimedHostedMu.Unlock()
	unclaimedHostedInstalls[install.InstallationUUID] = install
}

func takeUnclaimedHostedInstall(installationUUID string) *hostedSentryInstall {
	installationUUID = strings.TrimSpace(installationUUID)
	if installationUUID == "" {
		return nil
	}

	unclaimedHostedMu.Lock()
	defer unclaimedHostedMu.Unlock()
	install, ok := unclaimedHostedInstalls[installationUUID]
	if !ok {
		return nil
	}
	delete(unclaimedHostedInstalls, installationUUID)
	copied := install
	return &copied
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
