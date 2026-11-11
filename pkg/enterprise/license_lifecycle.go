package enterprise

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var errActiveLicenseMissing = errors.New("active license status has no license")

// LicenseLifecycle keeps organization grants inside the active license.
type LicenseLifecycle struct {
	db  func() *gorm.DB
	now func() time.Time
}

func NewLicenseLifecycle(db func() *gorm.DB) *LicenseLifecycle {
	return &LicenseLifecycle{db: db, now: time.Now}
}

type grantChange struct {
	OrganizationID uuid.UUID
	Feature        string
	Source         string
}

type lifecycleResult struct {
	Event             string
	Reason            string
	LicenseID         uuid.UUID
	OrganizationCount int64
	Pruned            int
	AutoGranted       bool
	Changes           []grantChange
}

func (l *LicenseLifecycle) HandleStatus(ctx context.Context, status licensing.Status) error {
	if status.IsDevelopment() {
		return nil
	}

	db := l.db()
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var result *lifecycleResult
	err := db.Transaction(func(tx *gorm.DB) error {
		var handleErr error
		result, handleErr = l.handle(tx, status)
		return handleErr
	})
	if err != nil {
		return lifecycleFailure(result, status, err)
	}

	logLifecycle(result)
	return nil
}

func (l *LicenseLifecycle) handle(tx *gorm.DB, status licensing.Status) (*lifecycleResult, error) {
	open, err := models.FindOpenLicenseActivation(tx)
	if err != nil {
		return nil, err
	}

	if open == nil {
		if status.State != licensing.StateActive {
			return nil, nil
		}
		return l.activate(tx, status)
	}

	switch status.State {
	case licensing.StateActive:
		if status.License == nil || status.License.ID == open.LicenseID {
			return nil, nil
		}
		return l.replace(tx, open, status)
	case licensing.StateExpired:
		return l.deactivate(tx, open, models.LicenseDeactivationReasonExpired)
	case licensing.StateRevoked:
		return l.deactivate(tx, open, models.LicenseDeactivationReasonRevoked)
	case licensing.StateNone:
		return l.deactivate(tx, open, models.LicenseDeactivationReasonRemoved)
	default:
		return nil, nil
	}
}

// activate inserts the open row first. A conflict means another replica
// already owns this activation, so this transaction does not change grants.
func (l *LicenseLifecycle) activate(tx *gorm.DB, status licensing.Status) (*lifecycleResult, error) {
	if status.License == nil {
		return nil, &licensing.LifecycleFailure{Event: "activation", Err: errActiveLicenseMissing}
	}

	features := sortedFeatureKeys(status.License.Features)
	activation := &models.InstallationLicenseActivation{
		LicenseID:       status.License.ID,
		LicenseFeatures: features,
	}
	inserted, err := models.InsertOpenLicenseActivation(tx, activation)
	if err != nil || !inserted {
		return nil, err
	}

	licenseID := status.License.ID
	removed, err := models.PruneEnterpriseFeaturesNotIn(tx, features)
	if err != nil {
		return nil, activationFailure(licenseID, err)
	}

	removedAt := l.now()
	if err := models.AppendRemovedGrants(tx, activation.ID, removedGrantRecords(removed, models.EnterpriseFeatureRemovedByActivation, removedAt)); err != nil {
		return nil, activationFailure(licenseID, err)
	}

	count, err := models.CountOrganizations(tx)
	if err != nil {
		return nil, activationFailure(licenseID, err)
	}

	changes := make([]grantChange, 0, len(removed))
	for _, row := range removed {
		changes = append(changes, grantChange{OrganizationID: row.OrganizationID, Feature: row.FeatureKey, Source: row.Source})
	}

	autoGranted := false
	if count == 1 {
		granted, grantChanges, grantErr := grantSingleOrganization(tx, features)
		if grantErr != nil {
			return nil, activationFailure(licenseID, grantErr)
		}
		autoGranted = granted
		changes = append(changes, grantChanges...)
	}

	if err := models.SetLicenseActivationResult(tx, activation.ID, int(count), autoGranted); err != nil {
		return nil, activationFailure(licenseID, err)
	}

	return &lifecycleResult{
		Event:             "activation",
		LicenseID:         licenseID,
		OrganizationCount: count,
		Pruned:            len(removed),
		AutoGranted:       autoGranted,
		Changes:           changes,
	}, nil
}

func grantSingleOrganization(tx *gorm.DB, features []string) (bool, []grantChange, error) {
	org, err := models.FindOnlyOrganization(tx)
	if err != nil {
		return false, nil, err
	}
	if org == nil {
		return false, nil, errors.New("organization count is 1 but no organization was found")
	}

	existing, err := models.ListOrganizationEnterpriseFeatures(tx, org.ID)
	if err != nil {
		return false, nil, err
	}
	before := make(map[string]struct{}, len(existing))
	for _, row := range existing {
		before[row.FeatureKey] = struct{}{}
	}

	inserted, err := models.GrantEnterpriseFeatures(tx, org.ID, features, models.EnterpriseFeatureSourceAutoSingleOrg, nil)
	if err != nil || inserted == 0 {
		return false, nil, err
	}

	latest, err := models.ListOrganizationEnterpriseFeatures(tx, org.ID)
	if err != nil {
		return false, nil, err
	}

	changes := make([]grantChange, 0, inserted)
	for _, row := range latest {
		if _, ok := before[row.FeatureKey]; ok {
			continue
		}
		changes = append(changes, grantChange{
			OrganizationID: org.ID,
			Feature:        row.FeatureKey,
			Source:         row.Source,
		})
	}

	return true, changes, nil
}

func (l *LicenseLifecycle) replace(tx *gorm.DB, open *models.InstallationLicenseActivation, status licensing.Status) (*lifecycleResult, error) {
	closed, err := models.CloseLicenseActivation(tx, open.ID, models.LicenseDeactivationReasonReplaced)
	if err != nil || !closed {
		return nil, err
	}

	result, err := l.activate(tx, status)
	if err != nil || result == nil {
		return result, err
	}

	result.Event = "replacement"
	result.Reason = models.LicenseDeactivationReasonReplaced
	return result, nil
}

func (l *LicenseLifecycle) deactivate(tx *gorm.DB, open *models.InstallationLicenseActivation, reason string) (*lifecycleResult, error) {
	closed, err := models.CloseLicenseActivation(tx, open.ID, reason)
	if err != nil || !closed {
		return nil, err
	}

	removed, err := models.DeleteAllEnterpriseFeatures(tx)
	if err != nil {
		return nil, &licensing.LifecycleFailure{Event: "deactivation", LicenseID: open.LicenseID, Err: err}
	}
	if err := models.AppendRemovedGrants(tx, open.ID, removedGrantRecords(removed, models.EnterpriseFeatureRemovedByDeactivation, l.now())); err != nil {
		return nil, &licensing.LifecycleFailure{Event: "deactivation", LicenseID: open.LicenseID, Err: err}
	}

	changes := make([]grantChange, 0, len(removed))
	for _, row := range removed {
		changes = append(changes, grantChange{OrganizationID: row.OrganizationID, Feature: row.FeatureKey, Source: row.Source})
	}

	return &lifecycleResult{
		Event:             "deactivation",
		Reason:            reason,
		LicenseID:         open.LicenseID,
		OrganizationCount: int64(open.OrganizationCount),
		Pruned:            len(removed),
		Changes:           changes,
	}, nil
}

func activationFailure(licenseID uuid.UUID, err error) error {
	return &licensing.LifecycleFailure{Event: "activation", LicenseID: licenseID, Err: err}
}

func removedGrantRecords(rows []models.OrganizationEnterpriseFeature, removedBy string, removedAt time.Time) []models.RemovedEnterpriseGrant {
	records := make([]models.RemovedEnterpriseGrant, 0, len(rows))
	for _, row := range rows {
		records = append(records, models.RemovedEnterpriseGrant{
			OrganizationID: row.OrganizationID,
			FeatureKey:     row.FeatureKey,
			Source:         row.Source,
			GrantedBy:      row.GrantedBy,
			CreatedAt:      row.CreatedAt,
			RemovedAt:      removedAt,
			RemovedBy:      removedBy,
		})
	}

	return records
}

func sortedFeatureKeys(features []licensing.Feature) []string {
	keys := make([]string, 0, len(features))
	for _, feature := range features {
		keys = append(keys, string(feature))
	}
	slices.Sort(keys)
	return keys
}

func lifecycleFailure(result *lifecycleResult, status licensing.Status, err error) error {
	var failure *licensing.LifecycleFailure
	if errors.As(err, &failure) {
		return err
	}

	wrapped := &licensing.LifecycleFailure{Event: "lifecycle", Err: err}
	if result != nil {
		wrapped.Event = result.Event
		wrapped.LicenseID = result.LicenseID
		return wrapped
	}
	if status.License != nil {
		wrapped.LicenseID = status.License.ID
	}

	return wrapped
}

func logLifecycle(result *lifecycleResult) {
	if result == nil {
		return
	}

	log.WithFields(log.Fields{
		"event":              result.Event,
		"license_id":         result.LicenseID.String(),
		"reason":             result.Reason,
		"organization_count": result.OrganizationCount,
		"pruned":             result.Pruned,
		"auto_granted":       result.AutoGranted,
	}).Info("Enterprise license lifecycle event")

	for _, change := range result.Changes {
		log.WithFields(log.Fields{
			"organization_id": change.OrganizationID.String(),
			"feature":         change.Feature,
			"source":          change.Source,
			"actor":           "system",
		}).Info("Enterprise feature grant changed")
	}
}
