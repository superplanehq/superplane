package enterprise

import (
	"context"
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// ErrActiveEnterpriseLicenseRequired is returned when an enable needs an open
// license activation and none exists.
var ErrActiveEnterpriseLicenseRequired = errors.New("active enterprise license required")

// ActiveEnterpriseLicenseMessage is the administrator-facing refusal to enable a feature.
const ActiveEnterpriseLicenseMessage = "An active Enterprise license is required to enable Enterprise features."

// UnlicensedFeatureError is returned when a new grant is not in the open license.
type UnlicensedFeatureError struct {
	Key string
}

func (e UnlicensedFeatureError) Error() string {
	return "The license does not include " + e.Key + "."
}

// OrganizationFeatureView is one Enterprise feature for one organization.
type OrganizationFeatureView struct {
	Key           string
	Licensed      bool
	Granted       bool
	Effective     bool
	Source        string
	GrantedBy     *uuid.UUID
	GrantedByName string
	CreatedAt     time.Time
}

// OrganizationFeatures reads and replaces organization Enterprise grants.
type OrganizationFeatures struct {
	license licensing.StatusProvider
}

func NewOrganizationFeatures(license licensing.StatusProvider) *OrganizationFeatures {
	return &OrganizationFeatures{license: license}
}

func (s *OrganizationFeatures) status() licensing.Status {
	if s == nil || s.license == nil {
		return licensing.Status{Edition: licensing.EditionCommunity, State: licensing.StateNone, Source: licensing.SourceNone}
	}

	return s.license.Status()
}

// List returns the active license features and any grant the license does not contain.
func (s *OrganizationFeatures) List(ctx context.Context, orgID uuid.UUID) ([]OrganizationFeatureView, bool, error) {
	db := database.DB(ctx)
	if _, err := models.FindOrganizationByIDInTransaction(db, orgID.String()); err != nil {
		return nil, false, err
	}

	grants, err := models.ListOrganizationEnterpriseFeatures(db, orgID)
	if err != nil {
		return nil, false, err
	}

	names, err := accountNames(db, grantedByIDs(grants))
	if err != nil {
		return nil, false, err
	}

	status := s.status()
	return featureViews(status, orgID, grants, names), status.State == licensing.StateActive, nil
}

// Set replaces the organization's grants with keys. New rows record actor.
// Existing rows keep their source, granted_by, and created_at.
func (s *OrganizationFeatures) Set(ctx context.Context, orgID uuid.UUID, keys []string, actor uuid.UUID) (added, removed []string, err error) {
	var addedRows, removedRows []models.OrganizationEnterpriseFeature
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		open, lockErr := models.LockOpenLicenseActivationForShare(tx)
		if lockErr != nil {
			return lockErr
		}

		wanted := sortedUniqueKeys(keys)
		current, listErr := models.ListOrganizationEnterpriseFeatures(tx, orgID)
		if listErr != nil {
			return listErr
		}

		currentKeys := map[string]struct{}{}
		for _, row := range current {
			currentKeys[row.FeatureKey] = struct{}{}
		}

		var toAdd []string
		for _, key := range wanted {
			if _, ok := currentKeys[key]; !ok {
				toAdd = append(toAdd, key)
			}
		}

		if open == nil && len(toAdd) > 0 {
			return ErrActiveEnterpriseLicenseRequired
		}
		if open != nil {
			licensed := map[string]struct{}{}
			for _, key := range open.LicenseFeatures {
				licensed[key] = struct{}{}
			}
			for _, key := range toAdd {
				if _, ok := licensed[key]; !ok || !licensing.IsRecognizedFeature(key) {
					return UnlicensedFeatureError{Key: key}
				}
			}
		}

		if _, findErr := models.FindOrganizationByIDInTransaction(tx, orgID.String()); findErr != nil {
			return findErr
		}

		addedRows, removedRows, err = models.SetOrganizationEnterpriseFeatures(tx, orgID, wanted, actor)
		return err
	})
	if err != nil {
		return nil, nil, err
	}

	for _, row := range addedRows {
		logGrantChange(row, actor)
		added = append(added, row.FeatureKey)
	}
	for _, row := range removedRows {
		logGrantChange(row, actor)
		removed = append(removed, row.FeatureKey)
	}
	slices.Sort(added)
	slices.Sort(removed)
	return added, removed, nil
}

// Effective returns the features that pass the organization entitlement formula.
func (s *OrganizationFeatures) Effective(ctx context.Context, orgID uuid.UUID) ([]licensing.Feature, error) {
	if orgID == uuid.Nil {
		return nil, nil
	}

	grants, err := models.ListOrganizationEnterpriseFeatures(database.DB(ctx), orgID)
	if err != nil {
		return nil, err
	}

	views := featureViews(s.status(), orgID, grants, nil)
	features := make([]licensing.Feature, 0, len(views))
	for _, view := range views {
		if view.Effective {
			features = append(features, licensing.Feature(view.Key))
		}
	}

	return features, nil
}

func featureViews(status licensing.Status, orgID uuid.UUID, grants []models.OrganizationEnterpriseFeature, names map[uuid.UUID]string) []OrganizationFeatureView {
	byKey := map[string]models.OrganizationEnterpriseFeature{}
	keys := map[string]struct{}{}
	if status.State == licensing.StateActive && status.License != nil {
		for _, feature := range status.License.Features {
			keys[string(feature)] = struct{}{}
		}
	}
	for _, grant := range grants {
		byKey[grant.FeatureKey] = grant
		keys[grant.FeatureKey] = struct{}{}
	}

	views := make([]OrganizationFeatureView, 0, len(keys))
	for _, key := range slices.Sorted(maps.Keys(keys)) {
		feature := licensing.Feature(key)
		grant, granted := byKey[key]
		view := OrganizationFeatureView{
			Key:       key,
			Licensed:  status.IsEntitled(feature),
			Granted:   granted,
			Effective: featureEffective(status, orgID, feature, granted),
		}
		if granted {
			view.Source = grant.Source
			view.GrantedBy = grant.GrantedBy
			view.CreatedAt = grant.CreatedAt
			if grant.GrantedBy != nil && names != nil {
				view.GrantedByName = names[*grant.GrantedBy]
			}
		}
		views = append(views, view)
	}

	return views
}

func featureEffective(status licensing.Status, orgID uuid.UUID, feature licensing.Feature, granted bool) bool {
	if orgID == uuid.Nil || !status.IsEntitled(feature) {
		return false
	}
	if status.IsDevelopment() {
		return true
	}

	return granted
}

func grantedByIDs(grants []models.OrganizationEnterpriseFeature) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(grants))
	seen := map[uuid.UUID]struct{}{}
	for _, grant := range grants {
		if grant.GrantedBy == nil {
			continue
		}
		if _, ok := seen[*grant.GrantedBy]; ok {
			continue
		}
		seen[*grant.GrantedBy] = struct{}{}
		ids = append(ids, *grant.GrantedBy)
	}

	return ids
}

func accountNames(tx *gorm.DB, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	names := map[uuid.UUID]string{}
	if len(ids) == 0 {
		return names, nil
	}

	var accounts []models.Account
	if err := tx.Where("id IN ?", ids).Find(&accounts).Error; err != nil {
		return nil, err
	}
	for _, account := range accounts {
		names[account.ID] = account.Name
	}

	return names, nil
}

func sortedUniqueKeys(keys []string) []string {
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, key)
	}
	slices.Sort(unique)
	return unique
}

func logGrantChange(row models.OrganizationEnterpriseFeature, actor uuid.UUID) {
	log.WithFields(log.Fields{
		"organization_id": row.OrganizationID.String(),
		"feature":         row.FeatureKey,
		"source":          row.Source,
		"actor":           actor.String(),
	}).Info("Enterprise feature grant changed")
}
