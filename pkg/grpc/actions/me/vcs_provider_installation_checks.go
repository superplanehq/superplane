package me

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"golang.org/x/sync/errgroup"
)

const (
	vcsProviderInstallationCheckConcurrency = 4
	vcsProviderInstallationCheckTimeout     = 15 * time.Second
)

type VCSProviderInstallationVerifier interface {
	VerifyInstallation(ctx context.Context, installationID int64) error
}

// VCSProviderInstallationChecks keeps one provider check per installation in
// each interval, so many open onboarding screens do not multiply the
// provider API calls.
type VCSProviderInstallationChecks struct {
	interval  time.Duration
	mu        sync.Mutex
	checkedAt map[int64]time.Time
	prunedAt  time.Time
}

func NewVCSProviderInstallationChecks(interval time.Duration) *VCSProviderInstallationChecks {
	return &VCSProviderInstallationChecks{interval: interval, checkedAt: map[int64]time.Time{}}
}

// due returns the installations that were not checked in the interval and
// records them as checked at now.
func (c *VCSProviderInstallationChecks) due(installationIDs []int64, now time.Time) []int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneExpired(now)
	due := make([]int64, 0, len(installationIDs))
	for _, installationID := range installationIDs {
		if checkedAt, checked := c.checkedAt[installationID]; checked && now.Sub(checkedAt) < c.interval {
			continue
		}
		c.checkedAt[installationID] = now
		due = append(due, installationID)
	}
	return due
}

func (c *VCSProviderInstallationChecks) pruneExpired(now time.Time) {
	if now.Sub(c.prunedAt) < c.interval {
		return
	}
	c.prunedAt = now
	for installationID, checkedAt := range c.checkedAt {
		if now.Sub(checkedAt) >= c.interval {
			delete(c.checkedAt, installationID)
		}
	}
}

// VerifyVCSProviderInstallations checks that the installations the user can
// see still exist. GitHub does not always send a webhook when an App is
// uninstalled. Without this check, an uninstalled organization stays on the
// onboarding screens until the next full catalog reconciliation.
func VerifyVCSProviderInstallations(
	ctx context.Context,
	provider string,
	checks *VCSProviderInstallationChecks,
	verifier VCSProviderInstallationVerifier,
) (*pb.VerifyVCSProviderInstallationsResponse, error) {
	provider, err := supportedVCSProvider(provider)
	if err != nil {
		return nil, err
	}
	if !vcsProviderConfigured(provider) {
		return nil, grpcerrors.FailedPrecondition(nil, "public GitHub App is not configured")
	}
	identity, err := currentVCSProviderIdentity(ctx, provider)
	if err != nil {
		return nil, vcsProviderIdentityError(err)
	}
	repositories, err := models.ListAccessibleVCSProviderRepositories(database.DB(ctx), provider, identity.userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list accessible repositories")
	}

	installationIDs := make([]int64, 0)
	seen := map[int64]struct{}{}
	for _, repository := range repositories {
		if _, ok := seen[repository.InstallationID]; ok {
			continue
		}
		seen[repository.InstallationID] = struct{}{}
		installationIDs = append(installationIDs, repository.InstallationID)
	}

	verifyInstallations(ctx, verifier, checks.due(installationIDs, time.Now()))
	return &pb.VerifyVCSProviderInstallationsResponse{}, nil
}

// The installations are already recorded as checked, so the checks must
// finish even when the page closes the request early.
func verifyInstallations(ctx context.Context, verifier VCSProviderInstallationVerifier, installationIDs []int64) {
	checkCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), vcsProviderInstallationCheckTimeout)
	defer cancel()
	var checks errgroup.Group
	checks.SetLimit(vcsProviderInstallationCheckConcurrency)
	for _, installationID := range installationIDs {
		checks.Go(func() error {
			if err := verifier.VerifyInstallation(checkCtx, installationID); err != nil {
				log.WithError(err).
					WithField("installation_id", installationID).
					Warn("failed to verify VCS provider installation")
			}
			return nil
		})
	}
	_ = checks.Wait()
}
