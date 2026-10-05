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

type State string

const (
	StateNone        State = "none"
	StateActive      State = "active"
	StateExpired     State = "expired"
	StateNotYetValid State = "not_yet_valid"
	StateInvalid     State = "invalid"
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

// Entitlements answers whether an Enterprise feature is available now.
type Entitlements interface {
	IsEntitled(feature Feature) bool
}

type communityEntitlements struct{}

func (communityEntitlements) IsEntitled(Feature) bool {
	return false
}

// Community grants no Enterprise features. Use it when no license service is
// configured so that Enterprise checks fail closed.
var Community Entitlements = communityEntitlements{}

// Allows reports whether entitlements grant a feature. A nil value grants
// nothing.
func Allows(entitlements Entitlements, feature Feature) bool {
	if entitlements == nil {
		return false
	}

	return entitlements.IsEntitled(feature)
}

type snapshot struct {
	source  SourceKind
	license *License
	reason  Reason
}

type Service struct {
	verifier  *Verifier
	source    Source
	now       func() time.Time
	current   atomic.Pointer[snapshot]
	refreshMu sync.Mutex
}

func NewService(verifier *Verifier, source Source) *Service {
	service := &Service{
		verifier: verifier,
		source:   source,
		now:      time.Now,
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

// Refresh reloads the license from its source. A temporary database error
// keeps the last known state instead of removing Enterprise access.
func (s *Service) Refresh(ctx context.Context) error {
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

func (s *Service) IsEntitled(feature Feature) bool {
	return s.Status().IsEntitled(feature)
}

// Install verifies a license and stores it. It accepts only a license that is
// valid now, so an installation never replaces a working license with one that
// grants nothing.
func (s *Service) Install(ctx context.Context, raw []byte, installedBy uuid.UUID) (Status, error) {
	writable := s.writableSource()
	if writable == nil {
		return s.Status(), ErrManagedByFile
	}

	license, err := s.verifier.Verify(raw)
	if err != nil {
		return s.Status(), err
	}

	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	switch license.ValidityAt(s.now()) {
	case ValidityExpired:
		return s.Status(), invalid(ReasonExpired)
	case ValidityNotYetValid:
		return s.Status(), invalid(ReasonNotYetValid)
	}

	if err := writable.Write(ctx, bytes.TrimSpace(raw), installedBy); err != nil {
		return s.Status(), err
	}

	log.WithFields(log.Fields{
		"license_id":   license.ID.String(),
		"kid":          license.KeyID,
		"installed_by": installedBy.String(),
	}).Info("Licensing: license installed")

	if err := s.refreshLocked(ctx); err != nil {
		return s.Status(), err
	}

	return s.Status(), nil
}

func (s *Service) Remove(ctx context.Context, removedBy uuid.UUID) (Status, error) {
	writable := s.writableSource()
	if writable == nil {
		return s.Status(), ErrManagedByFile
	}

	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	if err := writable.Clear(ctx); err != nil {
		return s.Status(), err
	}

	log.WithField("removed_by", removedBy.String()).Info("Licensing: license removed")

	// A failed reload must not keep the removed license active. Community mode
	// applies until the next refresh reads the shared state.
	if err := s.refreshLocked(ctx); err != nil {
		log.WithError(err).Warn("Licensing: license reload after removal failed; using Community mode")
		s.publish(&snapshot{source: SourceNone})
	}

	return s.Status(), nil
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
