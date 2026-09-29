package github

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	// hostedMemberCheckTTL is how long a cached membership check stays valid
	// before the next sync re-asks GitHub.
	hostedMemberCheckTTL = 15 * time.Minute

	// hostedMemberErrorRetryTTL is how long an errored membership check
	// blocks a retry. A short window lets a fixed app permission take effect
	// on the next syncs instead of after the full check TTL.
	hostedMemberErrorRetryTTL = time.Minute

	// hostedDiscoveryAPIBudget caps uncached membership lookups per discovery
	// run. The cache fills across the 5-second onboarding polls, so a large
	// installation table never turns one sync into an API storm.
	hostedDiscoveryAPIBudget = 25

	// hostedReconcileTTL is the in-process guard between reconciliations of
	// the installations table against the GitHub list API.
	hostedReconcileTTL = 5 * time.Minute
)

var (
	findGitHubLoginForUser       = findGitHubLoginForUserFromDB
	findCachedInstallationMember = findCachedInstallationMemberFromDB
	saveCachedInstallationMember = saveCachedInstallationMemberToDB
	listHostedInstallationRows   = listHostedInstallationRowsFromDB
	checkInstallationMembership  = checkInstallationMembershipOnGitHub
	listAppInstallationsDetailed = listAppInstallationsDetailedFromGitHub
	saveReconciledInstallation   = saveReconciledInstallationToDB
	pruneReconciledInstallations = pruneReconciledInstallationsFromDB
	hostedIdentityNow            = time.Now

	hostedReconcileMu   sync.Mutex
	hostedReconcileLast time.Time
)

// applyHostedIdentityDiscovery resolves the starter's GitHub identity and
// merges the installations that identity can use into the account picker.
// It is additive-only: it never removes picker entries, and any failure
// leaves the metadata as it was, so Sync proceeds exactly as without
// discovery.
func (g *GitHub) applyHostedIdentityDiscovery(ctx core.SyncContext, app common.HostedApp, metadata *common.Metadata) {
	userID := metadata.StartedByUserID
	if userID == "" {
		userID = ctx.ActorUserID
	}

	login, err := findGitHubLoginForUser(userID)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Errorf("failed to resolve GitHub login for user %s: %v", userID, err)
		}
		return
	}
	if login == "" {
		return
	}
	metadata.StartedByGitHubLogin = login

	// A bound connection keeps its picker as-is; discovery only serves the
	// account picker before bind.
	if metadata.InstallationID != "" {
		return
	}

	g.reconcileHostedInstallations(ctx, app)

	accessible, err := g.accessibleInstallations(ctx, app.ID, login)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Errorf("failed to discover accessible GitHub App installations: %v", err)
		}
		return
	}
	if len(accessible) == 0 {
		return
	}

	metadata.SetPendingInstallations(append(metadata.PendingInstallations, accessible...))

	requests := metadata.CurrentInstallRequests()
	for _, installation := range accessible {
		requests = filterResolvedInstallRequests(requests, installation.AccountLogin)
	}
	metadata.SetInstallRequests(requests)
}

// userCanAccessInstallation reports whether the GitHub login may use the
// installation: the installation account is the login itself, or the login is
// an active member of the installation organization. Results of the GitHub
// lookups are cached with a TTL; GitHub stays the source of truth. Discovery
// uses this cached variant.
func (g *GitHub) userCanAccessInstallation(
	integration core.IntegrationContext,
	appID int64,
	login string,
	snapshot hostedInstallationSnapshot,
) (bool, error) {
	return g.verifyInstallationAccess(integration, appID, login, snapshot, true)
}

// userCanAccessInstallationLive skips the cached answer. Privileged steps
// (claim, adopt, bind) always ask GitHub, so a membership revoked after a
// positive check cannot ride the cache into a bind.
func (g *GitHub) userCanAccessInstallationLive(
	integration core.IntegrationContext,
	appID int64,
	login string,
	snapshot hostedInstallationSnapshot,
) (bool, error) {
	return g.verifyInstallationAccess(integration, appID, login, snapshot, false)
}

func (g *GitHub) verifyInstallationAccess(
	integration core.IntegrationContext,
	appID int64,
	login string,
	snapshot hostedInstallationSnapshot,
	useCache bool,
) (bool, error) {
	login = strings.TrimSpace(login)
	if login == "" || snapshot.ID == "" || snapshot.Deleted {
		return false, nil
	}
	if strings.EqualFold(snapshot.AccountLogin, login) {
		return true, nil
	}
	if !strings.EqualFold(snapshot.AccountType, "Organization") {
		return false, nil
	}

	now := hostedIdentityNow().UTC()
	if useCache {
		cached, err := findCachedInstallationMember(snapshot.ID, login)
		if err == nil && cached != nil && now.Sub(cached.CheckedAt) <= cachedMemberTTL(cached) {
			return cached.Allowed, nil
		}
	}

	allowed, err := checkInstallationMembership(integration, appID, snapshot.ID, snapshot.AccountLogin, login)
	if err != nil {
		// A failed lookup is cached as not-allowed so repeated errors do not
		// burn the discovery budget on the same rows every sync. The errored
		// mark keeps the block short. The caller falls back to the
		// no-identity rules either way.
		_ = saveCachedInstallationMember(snapshot.ID, login, false, true, now)
		return false, err
	}

	// The cache only avoids repeated lookups; a write failure must not turn
	// a verified answer into an error.
	_ = saveCachedInstallationMember(snapshot.ID, login, allowed, false, now)
	return allowed, nil
}

// cachedMemberTTL returns how long a cached membership row stays valid. An
// errored check retries much sooner than a clean answer, so a fixed app
// permission shows in the picker without a long wait.
func cachedMemberTTL(cached *models.HostedAppInstallationMember) time.Duration {
	if cached.Errored {
		return hostedMemberErrorRetryTTL
	}
	return hostedMemberCheckTTL
}

// accessibleInstallations returns the live installations the login can use,
// as picker entries. Errors on single rows are logged and skipped: discovery
// is additive-only and must never break a sync.
func (g *GitHub) accessibleInstallations(
	ctx core.SyncContext,
	appID int64,
	login string,
) ([]common.PendingInstallation, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return nil, nil
	}

	rows, err := listHostedInstallationRows()
	if err != nil {
		return nil, err
	}

	now := hostedIdentityNow().UTC()
	apiBudget := hostedDiscoveryAPIBudget
	result := []common.PendingInstallation{}
	for _, row := range rows {
		if row.Deleted {
			continue
		}
		if g.needsMembershipLookup(row, login, now) {
			if apiBudget <= 0 {
				continue
			}
			apiBudget--
		}

		allowed, err := g.userCanAccessInstallation(ctx.Integration, appID, login, row)
		if err != nil && ctx.Logger != nil {
			// A misconfigured app (for example a missing organization
			// members read permission) must be visible in the logs, not a
			// silently empty picker.
			ctx.Logger.Errorf(
				"membership check for GitHub App installation %s (%s) failed: %v",
				row.ID, row.AccountLogin, err,
			)
		}
		if err != nil || !allowed {
			continue
		}
		result = append(result, common.PendingInstallation{
			ID:           row.ID,
			AccountLogin: row.AccountLogin,
			AccountType:  row.AccountType,
		})
	}
	return result, nil
}

func (g *GitHub) needsMembershipLookup(row hostedInstallationSnapshot, login string, now time.Time) bool {
	if strings.EqualFold(row.AccountLogin, login) {
		return false
	}
	if !strings.EqualFold(row.AccountType, "Organization") {
		return false
	}
	cached, err := findCachedInstallationMember(row.ID, login)
	return err != nil || cached == nil || now.Sub(cached.CheckedAt) > cachedMemberTTL(cached)
}

// reconcileHostedInstallations upserts every installation GitHub lists for
// the app, so identity discovery also sees installations that predate the
// webhook table. It never freshens last_event_at on existing rows: only
// signed webhooks may move an installation into the no-identity first-claim
// window. Rows GitHub no longer lists are soft-deleted: a dead installation
// (for example after a lost uninstall webhook) must not waste discovery
// lookups on every sync.
func (g *GitHub) reconcileHostedInstallations(ctx core.SyncContext, app common.HostedApp) {
	if !takeHostedReconcileSlot() {
		return
	}

	installations, err := listAppInstallationsDetailed(ctx.Integration, app.ID)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Errorf("failed to list GitHub App installations for reconcile: %v", err)
		}
		return
	}

	liveIDs := make([]string, 0, len(installations))
	for _, installation := range installations {
		liveIDs = append(liveIDs, installation.ID)
		if err := saveReconciledInstallation(installation); err != nil && ctx.Logger != nil {
			ctx.Logger.Errorf("failed to reconcile GitHub App installation %s: %v", installation.ID, err)
		}
	}

	if err := pruneReconciledInstallations(liveIDs); err != nil && ctx.Logger != nil {
		ctx.Logger.Errorf("failed to prune uninstalled GitHub App installations: %v", err)
	}
}

func takeHostedReconcileSlot() bool {
	hostedReconcileMu.Lock()
	defer hostedReconcileMu.Unlock()

	now := hostedIdentityNow().UTC()
	if !hostedReconcileLast.IsZero() && now.Sub(hostedReconcileLast) < hostedReconcileTTL {
		return false
	}
	hostedReconcileLast = now
	return true
}

func findGitHubLoginForUserFromDB(userID string) (string, error) {
	return models.FindGitHubLoginForUser(database.Conn(), userID)
}

func findCachedInstallationMemberFromDB(installationID, login string) (*models.HostedAppInstallationMember, error) {
	return models.FindHostedAppInstallationMember(database.Conn(), models.HostedAppProviderGitHub, installationID, login)
}

func saveCachedInstallationMemberToDB(installationID, login string, allowed, errored bool, checkedAt time.Time) error {
	return models.UpsertHostedAppInstallationMember(database.Conn(), models.HostedAppInstallationMember{
		Provider:       models.HostedAppProviderGitHub,
		InstallationID: installationID,
		MemberLogin:    login,
		Allowed:        allowed,
		Errored:        errored,
		CheckedAt:      checkedAt,
	})
}

func listHostedInstallationRowsFromDB() ([]hostedInstallationSnapshot, error) {
	rows, err := models.ListHostedAppInstallations(database.Conn(), models.HostedAppProviderGitHub)
	if err != nil {
		return nil, err
	}

	result := make([]hostedInstallationSnapshot, 0, len(rows))
	for _, row := range rows {
		result = append(result, hostedInstallationSnapshot{
			ID:           row.InstallationID,
			AccountLogin: row.AccountLogin,
			AccountType:  row.AccountType,
			LastEventAt:  row.LastEventAt,
			CreatedAt:    row.CreatedAt,
			Deleted:      row.DeletedAt.Valid,
		})
	}
	return result, nil
}

// checkInstallationMembershipOnGitHub asks GitHub whether the login is an
// active member of the installation organization. Only active membership
// verifies identity access: a collaborator on a single repository must not
// gain the whole installation. The membership endpoint needs the app's
// organization members read permission; without it the lookup errors and the
// caller falls back to the no-identity rules.
func checkInstallationMembershipOnGitHub(
	integration core.IntegrationContext,
	appID int64,
	installationID string,
	accountLogin string,
	login string,
) (bool, error) {
	client, err := newInstallationClient(integration, appID, installationID)
	if err != nil {
		return false, err
	}

	membership, response, err := client.Organizations.GetOrgMembership(context.Background(), login, accountLogin)
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}
	return membership.GetState() == "active", nil
}

func listAppInstallationsDetailedFromGitHub(integration core.IntegrationContext, appID int64) ([]hostedInstallationSnapshot, error) {
	client, err := newAppJWTClient(integration, appID)
	if err != nil {
		return nil, err
	}

	result := []hostedInstallationSnapshot{}
	opts := &github.ListOptions{PerPage: 100}
	for {
		installations, response, err := client.Apps.ListInstallations(context.Background(), opts)
		if err != nil {
			return nil, err
		}

		for _, installation := range installations {
			if installation == nil || installation.GetAccount() == nil {
				continue
			}
			createdAt := time.Time{}
			if installation.CreatedAt != nil {
				createdAt = installation.CreatedAt.Time
			}
			result = append(result, hostedInstallationSnapshot{
				ID:           strconv.FormatInt(installation.GetID(), 10),
				AccountLogin: installation.GetAccount().GetLogin(),
				AccountType:  installation.GetAccount().GetType(),
				CreatedAt:    createdAt,
				LastEventAt:  createdAt,
			})
		}

		if response == nil || response.NextPage == 0 {
			return result, nil
		}
		opts.Page = response.NextPage
	}
}

func pruneReconciledInstallationsFromDB(liveInstallationIDs []string) error {
	return models.SoftDeleteHostedAppInstallationsNotIn(database.Conn(), models.HostedAppProviderGitHub, liveInstallationIDs)
}

func saveReconciledInstallationToDB(snapshot hostedInstallationSnapshot) error {
	return models.ReconcileHostedAppInstallation(database.Conn(), models.HostedAppInstallation{
		Provider:       models.HostedAppProviderGitHub,
		InstallationID: snapshot.ID,
		AccountLogin:   snapshot.AccountLogin,
		AccountType:    snapshot.AccountType,
		LastEventAt:    snapshot.CreatedAt,
		CreatedAt:      snapshot.CreatedAt,
	})
}

func resetHostedIdentityHooks() {
	findGitHubLoginForUser = findGitHubLoginForUserFromDB
	findCachedInstallationMember = findCachedInstallationMemberFromDB
	saveCachedInstallationMember = saveCachedInstallationMemberToDB
	listHostedInstallationRows = listHostedInstallationRowsFromDB
	checkInstallationMembership = checkInstallationMembershipOnGitHub
	listAppInstallationsDetailed = listAppInstallationsDetailedFromGitHub
	saveReconciledInstallation = saveReconciledInstallationToDB
	pruneReconciledInstallations = pruneReconciledInstallationsFromDB
	hostedIdentityNow = time.Now

	hostedReconcileMu.Lock()
	hostedReconcileLast = time.Time{}
	hostedReconcileMu.Unlock()
}
