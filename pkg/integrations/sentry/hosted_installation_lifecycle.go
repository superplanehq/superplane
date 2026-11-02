package sentry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

// ErrHostedInstallReconnectRequired means SuperPlane cannot get credentials
// for an installation without a new install. The delivery stays blocked so
// an operator can see it, and a later retry finishes it after the user
// reconnects or Sentry deletes the installation.
var ErrHostedInstallReconnectRequired = errors.New("reconnect required: SuperPlane cannot get credentials for this Sentry installation; reinstall the SuperPlane Sentry app")

// HostedInstallationOutcome is the explicit result of one installation
// lifecycle event. Filtered is true when nothing needed to be stored.
type HostedInstallationOutcome struct {
	Filtered bool
	Note     string
}

// ProcessHostedInstallationCreated stores the grant of installation.created
// for the browser setup callback. The installation worker calls it outside
// any database transaction, and retries it until it returns no error.
//
// The order of the checks protects working credentials:
//   - A deleted installation stores nothing.
//   - A connection that already uses the installation needs nothing.
//   - A stored grant with tokens is reused. Sentry accepts a code only once.
//   - A delivery older than the claim window needs a reconnect.
//   - Otherwise exchange the code. When the browser callback already used
//     it, recover tokens through the trusted installation token grant.
func ProcessHostedInstallationCreated(
	ctx context.Context,
	httpCtx core.HTTPContext,
	app HostedApp,
	grant InstallationGrant,
	receivedAt time.Time,
) (HostedInstallationOutcome, error) {
	installationUUID := strings.TrimSpace(grant.UUID)
	code := strings.TrimSpace(grant.Code)

	deleted, err := hostedInstallGrants.Deleted(installationUUID)
	if err != nil {
		return HostedInstallationOutcome{}, fmt.Errorf("read Sentry installation deletion: %w", err)
	}
	if deleted {
		return HostedInstallationOutcome{Filtered: true, Note: "Sentry deleted this installation"}, nil
	}

	bound, err := hostedInstallationBound(ctx, installationUUID)
	if err != nil {
		return HostedInstallationOutcome{}, fmt.Errorf("find connections for Sentry installation: %w", err)
	}
	if bound {
		return HostedInstallationOutcome{Note: "a connection already uses this installation"}, nil
	}

	if code == "" {
		return HostedInstallationOutcome{Filtered: true, Note: "the installation event has no grant code"}, nil
	}

	stored, err := hostedInstallGrants.Find(installationUUID, code)
	if err != nil {
		return HostedInstallationOutcome{}, fmt.Errorf("read Sentry install grant: %w", err)
	}
	if stored != nil && strings.TrimSpace(stored.AccessToken) != "" {
		return HostedInstallationOutcome{Note: "stored tokens were reused"}, nil
	}

	expiresAt := receivedAt.Add(hostedInstallGrantTTL)
	if !time.Now().Before(expiresAt) {
		return HostedInstallationOutcome{}, ErrHostedInstallReconnectRequired
	}

	install := hostedSentryInstall{InstallationUUID: installationUUID, Code: code, ExpiresAt: expiresAt}
	if grant.OrgSlug != "" {
		install.Organization = &OrganizationSummary{Slug: grant.OrgSlug}
	}

	tokens, note, tokenErr := hostedInstallationTokens(httpCtx, app, installationUUID, code)
	if err := ctx.Err(); err != nil && tokens == nil {
		// The worker lost its lease or stops before it got tokens. The next
		// attempt decides. Tokens that Sentry already issued are stored below.
		return HostedInstallationOutcome{}, err
	}
	if tokens != nil {
		install.AccessToken = tokens.Token
		install.RefreshToken = tokens.RefreshToken
		install.TokenExpiresAt = tokens.ExpiresAt
	}

	// Store the grant also without tokens. The browser callback can then
	// still exchange a code that SuperPlane could not exchange.
	if err := hostedInstallGrants.Remember(install); err != nil {
		if errors.Is(err, models.ErrSentryAppInstallationDeleted) {
			return HostedInstallationOutcome{Filtered: true, Note: "Sentry deleted this installation"}, nil
		}
		return HostedInstallationOutcome{}, fmt.Errorf("store Sentry install grant: %w", err)
	}
	if tokenErr != nil {
		return HostedInstallationOutcome{}, tokenErr
	}
	return HostedInstallationOutcome{Note: note}, nil
}

// ProcessHostedInstallationDeleted records the deletion and drops the grant.
func ProcessHostedInstallationDeleted(installationUUID string) (HostedInstallationOutcome, error) {
	if strings.TrimSpace(installationUUID) == "" {
		return HostedInstallationOutcome{Filtered: true, Note: "the installation event has no installation"}, nil
	}
	if err := hostedInstallGrants.Uninstall(installationUUID); err != nil {
		return HostedInstallationOutcome{}, fmt.Errorf("record Sentry installation deletion: %w", err)
	}
	return HostedInstallationOutcome{Note: "the installation grant was removed"}, nil
}

func hostedInstallationTokens(httpCtx core.HTTPContext, app HostedApp, installationUUID, code string) (*sentryAppAuthorizationResponse, string, error) {
	tokens, exchangeErr := exchangeSentryAppCode(httpCtx, app, installationUUID, code)
	if exchangeErr == nil {
		return tokens, "the grant code was exchanged", nil
	}

	minted, mintErr := mintSentryAppToken(httpCtx, app, installationUUID)
	if mintErr == nil {
		return minted, "tokens were recovered for a used grant code", nil
	}

	var authorizationErr *sentryAppAuthorizationError
	if errors.As(mintErr, &authorizationErr) && authorizationErr.Permanent() {
		return nil, "", fmt.Errorf("%w (exchange: %v; recovery: %v)", ErrHostedInstallReconnectRequired, exchangeErr, mintErr)
	}
	return nil, "", fmt.Errorf("get Sentry installation tokens: exchange: %v; recovery: %w", exchangeErr, mintErr)
}

func hostedInstallationBound(ctx context.Context, installationUUID string) (bool, error) {
	integrations, err := models.ListSentryIntegrationsByInstallationUUID(database.DB(ctx), installationUUID)
	if err != nil {
		return false, err
	}
	return len(integrations) > 0, nil
}

// HostedInstallEnabled reports whether this process can store install
// grants. A worker-only process enables it at startup.
func HostedInstallEnabled() bool {
	return hostedInstallEncryptor != nil
}
