package enterprise

import (
	"cmp"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	LicenseEventActivation  = "activation"
	LicenseEventReplacement = "replacement"
	LicenseEventNone        = "none"
)

// PlannedGrantRemoval is one grant an install or removal would delete.
type PlannedGrantRemoval struct {
	OrganizationID uuid.UUID
	FeatureKey     string
}

// LicenseInstallPlan is the grant change a license install would make.
type LicenseInstallPlan struct {
	Event                   string
	Removed                 []PlannedGrantRemoval
	AutoGrantOrganizationID uuid.UUID
}

// AppliedLicenseInstall is the grant change a completed install wrote.
type AppliedLicenseInstall struct {
	Event       string
	Removed     []models.RemovedEnterpriseGrant
	AutoGranted bool
}

// PlanLicenseInstall predicts the lifecycle event for a verified license.
// organizationCount is the number of organizations that are not deleted.
// onlyOrganizationID is that organization when the count is one.
func PlanLicenseInstall(open *models.InstallationLicenseActivation, license *licensing.License, grants []models.OrganizationEnterpriseFeature, organizationCount int64, onlyOrganizationID uuid.UUID) LicenseInstallPlan {
	plan := LicenseInstallPlan{Event: LicenseEventActivation}
	if open != nil && license != nil && open.LicenseID == license.ID {
		plan.Event = LicenseEventNone
		return plan
	}
	if open != nil {
		plan.Event = LicenseEventReplacement
	}
	if license == nil {
		return plan
	}

	allowed := map[string]struct{}{}
	for _, feature := range license.Features {
		allowed[string(feature)] = struct{}{}
	}
	for _, grant := range grants {
		if _, ok := allowed[grant.FeatureKey]; ok {
			continue
		}
		plan.Removed = append(plan.Removed, PlannedGrantRemoval{
			OrganizationID: grant.OrganizationID,
			FeatureKey:     grant.FeatureKey,
		})
	}
	slices.SortFunc(plan.Removed, func(a, b PlannedGrantRemoval) int {
		if order := cmp.Compare(a.OrganizationID.String(), b.OrganizationID.String()); order != 0 {
			return order
		}
		return cmp.Compare(a.FeatureKey, b.FeatureKey)
	})

	if organizationCount != 1 || onlyOrganizationID == uuid.Nil {
		return plan
	}

	granted := map[string]struct{}{}
	for _, grant := range grants {
		if grant.OrganizationID == onlyOrganizationID {
			granted[grant.FeatureKey] = struct{}{}
		}
	}
	for _, feature := range license.Features {
		if _, ok := granted[string(feature)]; ok {
			continue
		}
		plan.AutoGrantOrganizationID = onlyOrganizationID
		break
	}

	return plan
}

// PlanLicenseRemoval lists every current grant. A removal deletes them all.
func PlanLicenseRemoval(grants []models.OrganizationEnterpriseFeature) []PlannedGrantRemoval {
	removed := make([]PlannedGrantRemoval, 0, len(grants))
	for _, grant := range grants {
		removed = append(removed, PlannedGrantRemoval{
			OrganizationID: grant.OrganizationID,
			FeatureKey:     grant.FeatureKey,
		})
	}
	slices.SortFunc(removed, func(a, b PlannedGrantRemoval) int {
		if order := cmp.Compare(a.OrganizationID.String(), b.OrganizationID.String()); order != 0 {
			return order
		}
		return cmp.Compare(a.FeatureKey, b.FeatureKey)
	})

	return removed
}

// DescribeAppliedInstall reads the activation rows from before and after Install.
func DescribeAppliedInstall(before, after *models.InstallationLicenseActivation) AppliedLicenseInstall {
	if after == nil || (before != nil && before.ID == after.ID) {
		return AppliedLicenseInstall{Event: LicenseEventNone}
	}

	event := LicenseEventActivation
	if before != nil {
		event = LicenseEventReplacement
	}

	return AppliedLicenseInstall{
		Event:       event,
		Removed:     after.RemovedGrants,
		AutoGranted: after.AutoGranted,
	}
}

// GrantsRemovedByDeactivation returns grants the removal event deleted.
func GrantsRemovedByDeactivation(row *models.InstallationLicenseActivation) []models.RemovedEnterpriseGrant {
	if row == nil || row.DeactivatedAt == nil || row.DeactivationReason == nil || *row.DeactivationReason != models.LicenseDeactivationReasonRemoved {
		return nil
	}

	removed := make([]models.RemovedEnterpriseGrant, 0, len(row.RemovedGrants))
	for _, grant := range row.RemovedGrants {
		if grant.RemovedBy == models.EnterpriseFeatureRemovedByDeactivation {
			removed = append(removed, grant)
		}
	}

	return removed
}
