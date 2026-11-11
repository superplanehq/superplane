package licensing

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
)

// RefreshInterval is how often every replica reloads the license, so that
// install, replace, and remove operations reach all replicas.
const RefreshInterval = 30 * time.Second

// ErrManagedByFile is returned when an administrator tries to change a license
// that comes from SUPERPLANE_LICENSE_PATH.
var ErrManagedByFile = errors.New("the license is managed by the installation configuration")

// ErrKeySyncUnavailable is returned when the service does not manage keys.
var ErrKeySyncUnavailable = errors.New("license key updates are not available")

type State string

const (
	StateNone        State = "none"
	StateActive      State = "active"
	StateExpired     State = "expired"
	StateNotYetValid State = "not_yet_valid"
	StateInvalid     State = "invalid"
	StateRevoked     State = "revoked"
)

// Status is the safe license state. It never contains the raw license.
type Status struct {
	Edition  Edition
	Source   SourceKind
	State    State
	Reason   Reason
	License  *License
	Writable bool
}

func (s Status) IsEntitled(feature Feature) bool {
	return s.State == StateActive && s.License.HasFeature(feature)
}

// StatusProvider reports the current installation license.
type StatusProvider interface {
	Status() Status
}

// IsDevelopment reports the ephemeral license used by the development server.
// That license bypasses organization grants and the lifecycle handler.
func (s Status) IsDevelopment() bool {
	return s.State == StateActive && s.License != nil && s.License.KeyID == DevelopmentKeyID
}

// GrantLookup answers whether an organization has a grant for a feature.
type GrantLookup interface {
	HasGrant(ctx context.Context, orgID uuid.UUID, feature Feature) (bool, error)
}

// LifecycleHandler reacts to the status after each successful license load.
type LifecycleHandler interface {
	HandleStatus(ctx context.Context, status Status) error
}

// LifecycleFailure is a lifecycle handler error with the fields the service logs.
type LifecycleFailure struct {
	Event     string
	LicenseID uuid.UUID
	Err       error
}

func (e *LifecycleFailure) Error() string {
	if e == nil || e.Err == nil {
		return "enterprise license lifecycle failed"
	}

	return e.Err.Error()
}

func (e *LifecycleFailure) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

// Entitlements answers whether an Enterprise feature is available to an
// organization now.
type Entitlements interface {
	IsEntitled(ctx context.Context, orgID uuid.UUID, feature Feature) bool
}

type communityEntitlements struct{}

func (communityEntitlements) IsEntitled(context.Context, uuid.UUID, Feature) bool {
	return false
}

// Community grants no Enterprise features. Use it when no license service is
// configured so that Enterprise checks fail closed.
var Community Entitlements = communityEntitlements{}

// Allows reports whether entitlements grant a feature to an organization.
// A nil value grants nothing.
func Allows(ctx context.Context, entitlements Entitlements, orgID uuid.UUID, feature Feature) bool {
	if entitlements == nil {
		return false
	}

	return entitlements.IsEntitled(ctx, orgID, feature)
}

// ErrNotLicensed reports that an Enterprise feature is not available.
// A missing organization grant uses this same error.
var ErrNotLicensed = errors.New("this action requires a SuperPlane Enterprise license")

// Require returns ErrNotLicensed unless the entitlements grant the feature
// to the organization.
func Require(ctx context.Context, entitlements Entitlements, orgID uuid.UUID, feature Feature) error {
	if !Allows(ctx, entitlements, orgID, feature) {
		return ErrNotLicensed
	}

	return nil
}

type snapshot struct {
	source  SourceKind
	license *License
	reason  Reason
}

type Service struct {
	verifier    *Verifier
	source      Source
	keySync     *KeySync
	revocations *RevocationSync
	grantLookup GrantLookup
	lifecycle   LifecycleHandler
	now         func() time.Time
	current     atomic.Pointer[snapshot]
	refreshMu   sync.Mutex
}

type ServiceOption func(*Service)

// WithKeySync keeps the trusted signing keys current while the service runs.
func WithKeySync(keySync *KeySync) ServiceOption {
	return func(s *Service) {
		s.keySync = keySync
	}
}

// WithRevocationSync keeps the revoked-license list current while the service runs.
func WithRevocationSync(revocations *RevocationSync) ServiceOption {
	return func(s *Service) {
		s.revocations = revocations
	}
}

// WithGrantLookup stores organization grant checks.
func WithGrantLookup(grants GrantLookup) ServiceOption {
	return func(s *Service) {
		s.grantLookup = grants
	}
}

// WithLifecycleHandler runs grant maintenance after each successful load.
func WithLifecycleHandler(handler LifecycleHandler) ServiceOption {
	return func(s *Service) {
		s.lifecycle = handler
	}
}

func NewService(verifier *Verifier, source Source, options ...ServiceOption) *Service {
	service := &Service{
		verifier: verifier,
		source:   source,
		now:      time.Now,
	}

	for _, option := range options {
		option(service)
	}

	service.current.Store(&snapshot{source: SourceNone})
	return service
}

// SourceFromEnvironment selects the single authoritative license source. A
// configured license file wins over the database. A configured but invalid
// file does not fall back to the database.
func SourceFromEnvironment(encryptor crypto.Encryptor) Source {
	if path := os.Getenv(LicensePathEnv); path != "" {
		return NewFileSource(path)
	}

	return NewDatabaseSource(encryptor)
}

// Start loads the license once and then refreshes it in the background until
// the context ends.
func (s *Service) Start(ctx context.Context) {
	if err := s.Refresh(ctx); err != nil {
		log.WithError(err).Warn("Licensing: initial license load failed; using Community mode")
	}

	if s.keySync != nil {
		s.keySync.Start(ctx, func() {
			if err := s.Refresh(ctx); err != nil {
				log.WithError(err).Warn("Licensing: license refresh after a key update failed")
			}
		})
	}

	if s.revocations != nil {
		s.revocations.Start(ctx, func() {
			if err := s.Refresh(ctx); err != nil {
				log.WithError(err).Warn("Licensing: license refresh after a revocation update failed")
			}
		})
	}

	go func() {
		ticker := time.NewTicker(RefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.Refresh(ctx); err != nil {
					log.WithError(err).Warn("Licensing: license refresh failed; keeping the last known state")
				}
			}
		}
	}()
}

// Refresh reloads the trusted keys and the license. A temporary database
// error keeps the last known state instead of removing Enterprise access.
func (s *Service) Refresh(ctx context.Context) error {
	s.reloadKeys(ctx)
	s.reloadRevocations(ctx)

	if err := s.reloadLicense(ctx); err != nil {
		return err
	}

	if s.current.Load().reason == ReasonUnknownKey && s.syncKeysOnDemand(ctx) {
		if err := s.reloadLicense(ctx); err != nil {
			return err
		}
	}

	s.notifyLifecycle(ctx, s.Status())
	return nil
}

// notifyLifecycle runs after the license lock is released. A handler error
// is logged and does not fail the load. The next refresh retries it.
func (s *Service) notifyLifecycle(ctx context.Context, status Status) {
	if s.lifecycle == nil || status.IsDevelopment() {
		return
	}

	err := s.lifecycle.HandleStatus(ctx, status)
	if err == nil {
		return
	}

	fields := log.Fields{"event": "lifecycle", "error": err}
	var failure *LifecycleFailure
	if errors.As(err, &failure) {
		if failure.Event != "" {
			fields["event"] = failure.Event
		}
		if failure.LicenseID != uuid.Nil {
			fields["license_id"] = failure.LicenseID.String()
		}
	} else if status.License != nil {
		fields["license_id"] = status.License.ID.String()
	}

	log.WithFields(fields).Warn("Licensing: enterprise license lifecycle failed")
}

// reloadKeys trusts a newer key list that another replica saved. It reports
// whether the trusted keys changed.
func (s *Service) reloadKeys(ctx context.Context) bool {
	if s.keySync == nil {
		return false
	}

	changed, err := s.keySync.Reload(ctx)
	if err != nil {
		log.WithError(err).Warn("Licensing: cached license keys were not loaded")
	}

	return changed
}

func (s *Service) reloadRevocations(ctx context.Context) {
	if s.revocations == nil {
		return
	}

	if _, err := s.revocations.Reload(ctx); err != nil {
		log.WithError(err).Warn("Licensing: cached revocation list was not loaded")
	}
}

func (s *Service) licenseRevoked(id uuid.UUID) bool {
	if s.revocations == nil {
		return false
	}

	return s.revocations.Contains(id)
}

// syncKeysOnDemand downloads the key list when a license names an unknown
// key. It reports whether the trusted keys changed.
func (s *Service) syncKeysOnDemand(ctx context.Context) bool {
	if s.keySync == nil {
		return false
	}

	changed, err := s.keySync.SyncOnDemand(ctx)
	if err != nil {
		log.WithError(err).Warn("Licensing: license key sync failed; keeping the trusted keys")
	}

	return changed
}

func (s *Service) reloadLicense(ctx context.Context) error {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	return s.refreshLocked(ctx)
}

func (s *Service) refreshLocked(ctx context.Context) error {
	next, err := s.load(ctx)
	if err != nil {
		return err
	}

	s.publish(next)
	return nil
}

func (s *Service) publish(next *snapshot) {
	previous := s.current.Swap(next)
	logTransition(previous, next)
}

func (s *Service) Status() Status {
	current := s.current.Load()
	status := Status{
		Edition:  EditionCommunity,
		Source:   current.source,
		State:    StateNone,
		Reason:   current.reason,
		License:  current.license,
		Writable: s.writableSource() != nil,
	}

	if current.reason != "" {
		status.State = StateInvalid
		return status
	}

	if current.license == nil {
		return status
	}

	if s.licenseRevoked(current.license.ID) {
		status.State = StateRevoked
		return status
	}

	switch current.license.ValidityAt(s.now()) {
	case ValidityActive:
		status.State = StateActive
		status.Edition = EditionEnterprise
	case ValidityNotYetValid:
		status.State = StateNotYetValid
	case ValidityExpired:
		status.State = StateExpired
	}

	return status
}

// IsEntitled reports whether the organization can use the feature.
// The license must be active and must contain the feature. A development
// license then allows every organization. Any other license also requires
// a grant. A missing organization, a missing lookup, or a lookup error
// denies access.
func (s *Service) IsEntitled(ctx context.Context, orgID uuid.UUID, feature Feature) bool {
	status := s.Status()
	if !status.IsEntitled(feature) {
		return false
	}
	if status.IsDevelopment() {
		return true
	}
	if orgID == uuid.Nil || s.grantLookup == nil {
		return false
	}

	granted, err := s.grantLookup.HasGrant(ctx, orgID, feature)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"organization_id": orgID.String(),
			"feature":         string(feature),
		}).Warn("Enterprise feature grant lookup failed")
		return false
	}

	return granted
}

// TrustedKeys reports the trusted key list. It returns false when the service
// does not sync keys.
func (s *Service) TrustedKeys() (KeySyncStatus, bool) {
	if s.keySync == nil {
		return KeySyncStatus{}, false
	}

	return s.keySync.Status(), true
}

// InstallKeyList trusts a key list that an administrator uploads and then
// reloads the license, which may use a key from the list.
func (s *Service) InstallKeyList(ctx context.Context, raw []byte) error {
	if s.keySync == nil {
		return ErrKeySyncUnavailable
	}

	if _, err := s.keySync.Install(ctx, raw); err != nil {
		return err
	}

	if err := s.Refresh(ctx); err != nil {
		log.WithError(err).Warn("Licensing: license refresh after a key list upload failed")
	}

	return nil
}

// DryRunInstall verifies a license the same way Install does, and does not store it.
func (s *Service) DryRunInstall(ctx context.Context, raw []byte) (*License, error) {
	if s.writableSource() == nil {
		return nil, ErrManagedByFile
	}

	return s.licenseAcceptedForInstall(ctx, raw)
}

// licenseAcceptedForInstall is the verification Install and DryRunInstall share.
func (s *Service) licenseAcceptedForInstall(ctx context.Context, raw []byte) (*License, error) {
	license, err := s.verifier.Verify(raw)
	if ReasonOf(err) == ReasonUnknownKey && s.reloadKeys(ctx) {
		license, err = s.verifier.Verify(raw)
	}

	if ReasonOf(err) == ReasonUnknownKey && s.syncKeysOnDemand(ctx) {
		license, err = s.verifier.Verify(raw)
	}

	if err != nil {
		return nil, err
	}

	switch license.ValidityAt(s.now()) {
	case ValidityExpired:
		return nil, invalid(ReasonExpired)
	case ValidityNotYetValid:
		return nil, invalid(ReasonNotYetValid)
	}

	if s.licenseRevoked(license.ID) {
		return nil, invalid(ReasonRevoked)
	}

	return license, nil
}

// Install verifies a license and stores it. It accepts only a license that is
// valid now, so an installation never replaces a working license with one that
// grants nothing.
func (s *Service) Install(ctx context.Context, raw []byte, installedBy uuid.UUID) (Status, error) {
	writable := s.writableSource()
	if writable == nil {
		return s.Status(), ErrManagedByFile
	}

	license, err := s.licenseAcceptedForInstall(ctx, raw)
	if err != nil {
		return s.Status(), err
	}

	s.refreshMu.Lock()

	switch license.ValidityAt(s.now()) {
	case ValidityExpired:
		status := s.Status()
		s.refreshMu.Unlock()
		return status, invalid(ReasonExpired)
	case ValidityNotYetValid:
		status := s.Status()
		s.refreshMu.Unlock()
		return status, invalid(ReasonNotYetValid)
	}

	if s.licenseRevoked(license.ID) {
		status := s.Status()
		s.refreshMu.Unlock()
		return status, invalid(ReasonRevoked)
	}

	if err := writable.Write(ctx, bytes.TrimSpace(raw), installedBy); err != nil {
		status := s.Status()
		s.refreshMu.Unlock()
		return status, err
	}

	log.WithFields(log.Fields{
		"license_id":   license.ID.String(),
		"kid":          license.KeyID,
		"installed_by": installedBy.String(),
	}).Info("Licensing: license installed")

	reloadErr := s.refreshAfterChange(ctx)
	status := s.Status()
	s.refreshMu.Unlock()
	if reloadErr != nil {
		return status, reloadErr
	}

	s.notifyLifecycle(ctx, status)
	return status, nil
}

func (s *Service) Remove(ctx context.Context, removedBy uuid.UUID) (Status, error) {
	writable := s.writableSource()
	if writable == nil {
		return s.Status(), ErrManagedByFile
	}

	s.refreshMu.Lock()

	if err := writable.Clear(ctx); err != nil {
		status := s.Status()
		s.refreshMu.Unlock()
		return status, err
	}

	log.WithField("removed_by", removedBy.String()).Info("Licensing: license removed")

	reloadErr := s.refreshAfterChange(ctx)
	status := s.Status()
	s.refreshMu.Unlock()
	if reloadErr == nil {
		s.notifyLifecycle(ctx, status)
	}

	return status, nil
}

// refreshAfterChange reads the shared state after a change. When the read
// fails, neither the previous nor the written license is confirmed, so
// Community mode applies until the next refresh succeeds.
func (s *Service) refreshAfterChange(ctx context.Context) error {
	err := s.refreshLocked(ctx)
	if err != nil {
		log.WithError(err).Warn("Licensing: license reload after change failed; using Community mode")
		s.publish(&snapshot{source: SourceNone})
	}

	return err
}

func (s *Service) writableSource() WritableSource {
	writable, ok := s.source.(WritableSource)
	if !ok {
		return nil
	}

	return writable
}

func (s *Service) load(ctx context.Context) (*snapshot, error) {
	kind := s.source.Kind()
	raw, err := s.source.Read(ctx)
	if errors.Is(err, ErrNotInstalled) {
		return &snapshot{source: SourceNone}, nil
	}

	if err != nil {
		var verificationErr *VerificationError
		if errors.As(err, &verificationErr) {
			return &snapshot{source: kind, reason: verificationErr.Reason}, nil
		}

		if kind == SourceFile {
			return &snapshot{source: kind, reason: ReasonUnreadable}, nil
		}

		return nil, err
	}

	license, err := s.verifier.Verify(raw)
	if err != nil {
		return &snapshot{source: kind, reason: ReasonOf(err)}, nil
	}

	return &snapshot{source: kind, license: license}, nil
}

func logTransition(previous, next *snapshot) {
	if previous != nil && sameSnapshot(previous, next) {
		return
	}

	fields := log.Fields{"source": string(next.source)}
	if next.reason != "" {
		fields["reason"] = string(next.reason)
		log.WithFields(fields).Warn("Licensing: license is not valid; using Community mode")
		return
	}

	if next.license == nil {
		log.WithFields(fields).Info("Licensing: no license installed; using Community mode")
		return
	}

	fields["license_id"] = next.license.ID.String()
	fields["kid"] = next.license.KeyID
	fields["expires_at"] = next.license.ExpiresAt.Format(time.RFC3339)
	log.WithFields(fields).Info("Licensing: license loaded")
}

func sameSnapshot(a, b *snapshot) bool {
	if a.source != b.source || a.reason != b.reason {
		return false
	}

	if a.license == nil || b.license == nil {
		return a.license == b.license
	}

	return a.license.ID == b.license.ID
}
