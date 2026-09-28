package github

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const hostedInstallationClaimWindow = 15 * time.Minute

var (
	errHostedInstallationNotAllowed = errors.New("installation is not allowed")

	findHostedInstallation          = findHostedInstallationFromDB
	findHostedInstallationByAccount = findHostedInstallationByAccountFromDB
	installationUsedByOtherOrg      = installationUsedByOtherOrgFromDB
	fetchAppInstallation            = fetchAppInstallationFromGitHub
	hostedClaimNow                  = time.Now
	hostedClaimDB                   = hostedClaimDBFromContext
)

type hostedInstallationSnapshot struct {
	ID           string
	AccountLogin string
	AccountType  string
	LastEventAt  time.Time
	CreatedAt    time.Time
	Deleted      bool
}

func (g *GitHub) offerHostedInstallation(ctx core.HTTPRequestContext, metadata common.Metadata, installationID string) error {
	if metadata.AllowsPendingInstallation(installationID) {
		ctx.Integration.SetMetadata(metadata)
		return nil
	}

	snapshot, err := g.resolveHostedInstallationClaim(ctx, metadata, installationID)
	if err != nil {
		return err
	}

	pending := metadata.PendingInstallations
	pending = append(pending, common.PendingInstallation{
		ID:           snapshot.ID,
		AccountLogin: snapshot.AccountLogin,
		AccountType:  snapshot.AccountType,
	})
	metadata.SetPendingInstallations(pending)

	remainingRequests := metadata.CurrentInstallRequests()
	remainingRequests = filterResolvedInstallRequests(remainingRequests, snapshot.AccountLogin)
	metadata.SetInstallRequests(remainingRequests)

	ctx.Integration.SetMetadata(metadata)
	ctx.Integration.RemoveBrowserAction()
	return nil
}

func (g *GitHub) resolveHostedInstallationClaim(ctx core.HTTPRequestContext, metadata common.Metadata, installationID string) (hostedInstallationSnapshot, error) {
	now := hostedClaimNow().UTC()
	record, err := findHostedInstallation(ctx, installationID)
	if err != nil {
		return hostedInstallationSnapshot{}, err
	}

	var snapshot hostedInstallationSnapshot
	if record != nil {
		snapshot = *record
	} else {
		fetched, err := fetchAppInstallation(ctx, metadata.GitHubApp.ID, installationID)
		if err != nil {
			return hostedInstallationSnapshot{}, err
		}
		if fetched.ID == "" {
			return hostedInstallationSnapshot{}, errHostedInstallationNotAllowed
		}
		snapshot = fetched
	}

	// Identity first: when GitHub vouches that the starter's login can use
	// this installation, the freshness window and the cross-organization
	// exclusivity do not apply. Identity is additive-only; on a miss or an
	// error the first-claim rules below run unchanged.
	if g.claimVerifiedByIdentity(ctx, metadata, snapshot) {
		return snapshot, nil
	}

	if snapshot.Deleted {
		return hostedInstallationSnapshot{}, errHostedInstallationNotAllowed
	}
	freshFrom := snapshot.LastEventAt
	if record == nil {
		freshFrom = snapshot.CreatedAt
		if freshFrom.IsZero() {
			freshFrom = snapshot.LastEventAt
		}
	}
	if now.Sub(freshFrom) > hostedInstallationClaimWindow {
		return hostedInstallationSnapshot{}, errHostedInstallationNotAllowed
	}
	if err := rejectCrossOrganizationInstallation(ctx, installationID); err != nil {
		return hostedInstallationSnapshot{}, err
	}
	return snapshot, nil
}

func (g *GitHub) claimVerifiedByIdentity(ctx core.HTTPRequestContext, metadata common.Metadata, snapshot hostedInstallationSnapshot) bool {
	login, err := findGitHubLoginForUser(metadata.StartedByUserID)
	if err != nil || login == "" {
		return false
	}

	// A claim is privileged, so the check is live: a cached answer from
	// discovery must not authorize a claim after GitHub revoked the access.
	allowed, err := g.userCanAccessInstallationLive(ctx.Integration, metadata.GitHubApp.ID, login, snapshot)
	return err == nil && allowed
}

// ensureHostedBindAllowed re-checks exclusivity when the member binds a
// picker entry. The entry was validated when it was offered, but another
// organization can bind the same installation between offer and bind (or in
// a racing claim), so an identity-unverified bind must not proceed when the
// installation is already in use elsewhere.
func (g *GitHub) ensureHostedBindAllowed(ctx core.HTTPRequestContext, metadata common.Metadata, installationID string) error {
	snapshot := hostedInstallationSnapshot{ID: installationID}
	if record, err := findHostedInstallation(ctx, installationID); err == nil && record != nil {
		snapshot = *record
	} else {
		for _, pending := range metadata.PendingInstallations {
			if pending.ID == installationID {
				snapshot.AccountLogin = pending.AccountLogin
				snapshot.AccountType = pending.AccountType
				break
			}
		}
	}

	if g.claimVerifiedByIdentity(ctx, metadata, snapshot) {
		return nil
	}
	return rejectCrossOrganizationInstallation(ctx, installationID)
}

func rejectCrossOrganizationInstallation(ctx core.HTTPRequestContext, installationID string) error {
	used, err := installationUsedByOtherOrg(ctx.OrganizationID, installationID)
	if err != nil {
		return err
	}
	if used {
		return errHostedInstallationNotAllowed
	}
	return nil
}

func installationUsedByOtherOrgFromDB(organizationID, installationID string) (bool, error) {
	users, err := models.ListGitHubIntegrationsUsingInstallation(database.Conn(), installationID)
	if err != nil {
		return false, err
	}
	return otherOrganizationUsesInstallation(organizationID, users), nil
}

func otherOrganizationUsesInstallation(organizationID string, users []models.Integration) bool {
	if organizationID == "" {
		return len(users) > 0
	}
	for _, user := range users {
		if user.OrganizationID.String() != organizationID {
			return true
		}
	}
	return false
}

func filterResolvedInstallRequests(requests []common.InstallRequest, accountLogin string) []common.InstallRequest {
	if accountLogin == "" {
		return requests
	}
	remaining := make([]common.InstallRequest, 0, len(requests))
	for _, request := range requests {
		if request.AccountLogin != "" && strings.EqualFold(request.AccountLogin, accountLogin) {
			continue
		}
		remaining = append(remaining, request)
	}
	return remaining
}

func findHostedInstallationFromDB(ctx core.HTTPRequestContext, installationID string) (*hostedInstallationSnapshot, error) {
	row, err := models.FindHostedAppInstallation(hostedClaimDB(ctx), models.HostedAppProviderGitHub, installationID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return &hostedInstallationSnapshot{
		ID:           row.InstallationID,
		AccountLogin: row.AccountLogin,
		AccountType:  row.AccountType,
		LastEventAt:  row.LastEventAt,
		CreatedAt:    row.CreatedAt,
		Deleted:      row.DeletedAt.Valid,
	}, nil
}

func findHostedInstallationByAccountFromDB(ctx context.Context, accountLogin string) (*hostedInstallationSnapshot, error) {
	row, err := models.FindHostedAppInstallationByAccountLogin(database.DB(ctx), models.HostedAppProviderGitHub, accountLogin)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return &hostedInstallationSnapshot{
		ID:           row.InstallationID,
		AccountLogin: row.AccountLogin,
		AccountType:  row.AccountType,
		LastEventAt:  row.LastEventAt,
		CreatedAt:    row.CreatedAt,
		Deleted:      row.DeletedAt.Valid,
	}, nil
}

func fetchAppInstallationFromGitHub(ctx core.HTTPRequestContext, appID int64, installationID string) (hostedInstallationSnapshot, error) {
	id, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		return hostedInstallationSnapshot{}, errHostedInstallationNotAllowed
	}

	client, err := newAppJWTClient(ctx.Integration, appID)
	if err != nil {
		return hostedInstallationSnapshot{}, fmt.Errorf("failed to create app client: %w", err)
	}

	installation, _, err := client.Apps.GetInstallation(context.Background(), id)
	if err != nil {
		return hostedInstallationSnapshot{}, errHostedInstallationNotAllowed
	}
	if installation == nil || installation.GetAccount() == nil {
		return hostedInstallationSnapshot{}, errHostedInstallationNotAllowed
	}

	createdAt := time.Time{}
	if installation.CreatedAt != nil {
		createdAt = installation.CreatedAt.Time
	}

	return hostedInstallationSnapshot{
		ID:           strconv.FormatInt(installation.GetID(), 10),
		AccountLogin: installation.GetAccount().GetLogin(),
		AccountType:  installation.GetAccount().GetType(),
		CreatedAt:    createdAt,
		LastEventAt:  createdAt,
	}, nil
}

func hostedClaimDBFromContext(ctx core.HTTPRequestContext) *gorm.DB {
	if ctx.Request != nil {
		return database.DB(ctx.Request.Context())
	}
	return database.Conn()
}

func resetHostedClaimHooks() {
	findHostedInstallation = findHostedInstallationFromDB
	findHostedInstallationByAccount = findHostedInstallationByAccountFromDB
	installationUsedByOtherOrg = installationUsedByOtherOrgFromDB
	fetchAppInstallation = fetchAppInstallationFromGitHub
	hostedClaimNow = time.Now
	hostedClaimDB = hostedClaimDBFromContext
}
