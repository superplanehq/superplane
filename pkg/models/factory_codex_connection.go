package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/features"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CodexConnectionStatusPending = "pending"
	// CodexConnectionStatusCanceling is derived for API views and is not persisted.
	CodexConnectionStatusCanceling         = "canceling"
	CodexConnectionStatusFinishing         = "finishing"
	CodexConnectionStatusReady             = "ready"
	CodexConnectionStatusReconnectRequired = "reconnect_required"
	CodexConnectionStatusDisconnected      = "disconnected"
)

var (
	ErrCodexConnectionNotFound    = newSubscriptionAdmissionError("chatgpt subscription connection not found")
	ErrCodexConnectionNotEligible = newSubscriptionAdmissionError("Connect a ChatGPT subscription as an active workspace member")
	ErrCodexConnectionNotReady    = newSubscriptionAdmissionError("connect ChatGPT before this task can run")
	ErrCodexConnectionStale       = newSubscriptionAdmissionError("the ChatGPT sign-in update is out of date")
	ErrCodexConnectionActive      = errors.New("stop the running tasks before you disconnect ChatGPT")
	ErrCodexConnectionUnavailable = newSubscriptionAdmissionError("ChatGPT subscription requires factories access for this organization")
)

// CodexConnectionBusyError names the task that holds the subscription.
type CodexConnectionBusyError struct {
	TaskID uuid.UUID
}

func (e *CodexConnectionBusyError) Error() string {
	if e == nil || e.TaskID == uuid.Nil {
		return "ChatGPT subscription is in use by another task."
	}
	return fmt.Sprintf("ChatGPT subscription is in use by task %s.", e.TaskID)
}

// FactoryCodexConnection is one member’s ChatGPT subscription in an organization.
// The ciphertext is the native Codex auth bundle, bound to this row ID.
type FactoryCodexConnection struct {
	ID                         uuid.UUID
	OrganizationID             uuid.UUID
	OwnerUserID                uuid.UUID
	Status                     string
	ResumeStatus               string
	VerificationURL            string
	VerificationCode           string
	AccountLabel               string
	UsageLimits                datatypes.JSONSlice[CodexUsageLimit]
	AvailableModels            datatypes.JSONSlice[CodexModel]
	UsageObservedAt            *time.Time
	BundleCiphertext           []byte
	CredentialVersion          int64
	FinalCheckpointVersion     int64
	ReservationGeneration      int64
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
	RevocationPendingAt        *time.Time
	LoginLease                 *FactoryCodexLease `gorm:"-"`
	LoginCancellationRequested bool               `gorm:"-"`
}

type CodexUsageLimit struct {
	LimitID   string            `json:"limitId"`
	LimitName string            `json:"limitName"`
	PlanType  string            `json:"planType"`
	Primary   *CodexUsageWindow `json:"primary"`
	Secondary *CodexUsageWindow `json:"secondary"`
}

type CodexUsageWindow struct {
	UsedPercent        *float64 `json:"usedPercent"`
	WindowDurationMins *int64   `json:"windowDurationMins"`
	ResetsAt           *int64   `json:"resetsAt"`
}

type CodexModel struct {
	ID                        string                       `json:"id"`
	Model                     string                       `json:"model"`
	DisplayName               string                       `json:"displayName"`
	DefaultReasoningEffort    string                       `json:"defaultReasoningEffort"`
	SupportedReasoningEfforts []CodexReasoningEffortOption `json:"supportedReasoningEfforts"`
	IsDefault                 bool                         `json:"isDefault"`
}

type CodexReasoningEffortOption struct {
	ReasoningEffort string `json:"reasoningEffort"`
	Description     string `json:"description"`
}

func (FactoryCodexConnection) TableName() string {
	return "factory_codex_connections"
}

// Reserved reports an exclusive sign-in or catalog refresh.
// Run leases do not mark the connection reserved.
func (c *FactoryCodexConnection) Reserved() bool {
	return c != nil && c.LoginLease != nil
}

func FindCodexConnection(tx *gorm.DB, organizationID, ownerID uuid.UUID) (*FactoryCodexConnection, error) {
	var connection FactoryCodexConnection
	err := tx.
		Where("organization_id = ? AND owner_user_id = ?", organizationID, ownerID).
		First(&connection).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCodexConnectionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &connection, nil
}

func LockCodexConnection(tx *gorm.DB, organizationID, ownerID uuid.UUID) (*FactoryCodexConnection, error) {
	var connection FactoryCodexConnection
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ? AND owner_user_id = ?", organizationID, ownerID).
		First(&connection).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCodexConnectionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &connection, nil
}

func FindFactoryCodexConnectionByID(tx *gorm.DB, id uuid.UUID) (*FactoryCodexConnection, error) {
	var connection FactoryCodexConnection
	err := tx.Where("id = ?", id).First(&connection).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCodexConnectionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &connection, nil
}

// LockScopedFactoryCodexConnection locks the exact connection in a signed scope.
func LockScopedFactoryCodexConnection(tx *gorm.DB, organizationID, connectionID uuid.UUID) (*FactoryCodexConnection, error) {
	snapshot, err := FindFactoryCodexConnectionByID(tx, connectionID)
	if err != nil {
		return nil, err
	}
	if snapshot.OrganizationID != organizationID {
		return nil, ErrCodexConnectionStale
	}
	if _, err := ActiveCodexMember(tx, organizationID, snapshot.OwnerUserID); err != nil {
		return nil, err
	}
	c, err := FindFactoryCodexConnectionByID(tx.Clauses(clause.Locking{Strength: "UPDATE"}), connectionID)
	if err != nil {
		return nil, err
	}
	if c.OrganizationID != organizationID || c.RevocationPendingAt != nil {
		return nil, ErrCodexConnectionStale
	}
	return c, nil
}

func CreateCodexConnection(
	tx *gorm.DB,
	organizationID, ownerUserID uuid.UUID,
	now time.Time,
) (*FactoryCodexConnection, error) {
	connection := &FactoryCodexConnection{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		OwnerUserID:    ownerUserID,
		Status:         CodexConnectionStatusDisconnected,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := tx.Create(connection).Error; err != nil {
		return nil, err
	}
	return connection, nil
}

// BeginLogin holds the connection for a new device-login task.
// The login lease stays until the login runner VM is confirmed deleted.
func (c *FactoryCodexConnection) BeginLogin(tx *gorm.DB, taskID uuid.UUID, now time.Time) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		if err := requireCodexFactoriesAccess(tx, c.OrganizationID); err != nil {
			return err
		}
		if err := c.relock(tx); err != nil {
			return err
		}
		if busy, err := c.busyLease(tx); err != nil {
			return err
		} else if busy != nil {
			return &CodexConnectionBusyError{TaskID: busy.TaskID}
		}
		switch c.Status {
		case CodexConnectionStatusPending, CodexConnectionStatusFinishing:
			return &CodexConnectionBusyError{}
		case CodexConnectionStatusReady,
			CodexConnectionStatusReconnectRequired,
			CodexConnectionStatusDisconnected:
		default:
			return fmt.Errorf("cannot start ChatGPT sign-in from status %s", c.Status)
		}

		resume := c.Status
		if _, err := c.createLease(tx, taskID, CodexLeasePurposeLogin, nil, nil, now); err != nil {
			return err
		}
		result := tx.Model(c).
			Where("id = ? AND status = ?", c.ID, resume).
			Updates(map[string]any{
				"status":            CodexConnectionStatusPending,
				"usage_limits":      datatypes.NewJSONSlice([]CodexUsageLimit{}),
				"available_models":  datatypes.NewJSONSlice([]CodexModel{}),
				"usage_observed_at": nil,
				"resume_status":     resume,
				"verification_url":  "",
				"verification_code": "",
				"updated_at":        now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrCodexConnectionStale
		}
		c.ResumeStatus = resume
		c.UsageLimits = nil
		c.AvailableModels = nil
		c.UsageObservedAt = nil
		c.Status = CodexConnectionStatusPending
		c.VerificationURL = ""
		c.VerificationCode = ""
		c.UpdatedAt = now
		return nil
	})
}

// BeginCatalogRefresh holds the connection for one model-catalog task.
// The task checkpoints the real auth bundle, so it stays exclusive.
func (c *FactoryCodexConnection) BeginCatalogRefresh(tx *gorm.DB, taskID uuid.UUID, now time.Time) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		if err := requireCodexFactoriesAccess(tx, c.OrganizationID); err != nil {
			return err
		}
		if err := c.relock(tx); err != nil {
			return err
		}
		if c.Status != CodexConnectionStatusReady || c.CredentialVersion < 1 || len(c.BundleCiphertext) == 0 {
			if busy, err := c.busyLease(tx); err != nil {
				return err
			} else if busy != nil {
				return &CodexConnectionBusyError{TaskID: busy.TaskID}
			}
			return ErrCodexConnectionNotReady
		}
		if busy, err := c.busyLease(tx); err != nil {
			return err
		} else if busy != nil {
			return &CodexConnectionBusyError{TaskID: busy.TaskID}
		}
		if _, err := c.createLease(tx, taskID, CodexLeasePurposeLogin, nil, nil, now); err != nil {
			return err
		}
		if err := tx.Model(c).Updates(map[string]any{"resume_status": c.Status, "updated_at": now}).Error; err != nil {
			return err
		}
		c.ResumeStatus = c.Status
		c.UpdatedAt = now
		return nil
	})
}

func (c *FactoryCodexConnection) RecordDeviceCode(
	tx *gorm.DB,
	taskID uuid.UUID,
	generation int64,
	verificationURL, verificationCode string,
	now time.Time,
) error {
	if _, err := c.requireLoginLease(tx, taskID, generation); err != nil {
		return err
	}
	if c.Status != CodexConnectionStatusPending {
		return ErrCodexConnectionStale
	}
	result := tx.Model(c).
		Where("id = ? AND status = ?", c.ID, CodexConnectionStatusPending).
		Updates(map[string]any{
			"verification_url":  verificationURL,
			"verification_code": verificationCode,
			"updated_at":        now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrCodexConnectionStale
	}
	c.VerificationURL = verificationURL
	c.VerificationCode = verificationCode
	c.UpdatedAt = now
	return nil
}

// StoreBundle saves a newer auth bundle for the active reservation.
// Login stays exclusive until setup confirms local cleanup or the VM is deleted.
func (c *FactoryCodexConnection) StoreBundle(
	tx *gorm.DB,
	taskID uuid.UUID,
	generation, version int64,
	ciphertext []byte,
	accountLabel string,
	final bool,
	now time.Time,
) error {
	if _, err := c.requireLoginLease(tx, taskID, generation); err != nil {
		return err
	}
	if version != c.CredentialVersion+1 {
		return ErrCodexConnectionStale
	}
	status := c.Status
	if status == CodexConnectionStatusPending {
		status = CodexConnectionStatusFinishing
	}
	finalVersion := c.FinalCheckpointVersion
	if final {
		finalVersion = version
	}
	result := tx.Model(c).
		Where("id = ? AND credential_version = ? AND revocation_pending_at IS NULL", c.ID, c.CredentialVersion).
		Updates(map[string]any{
			"status":                   status,
			"bundle_ciphertext":        ciphertext,
			"account_label":            accountLabel,
			"credential_version":       version,
			"final_checkpoint_version": finalVersion,
			"verification_url":         "",
			"verification_code":        "",
			"updated_at":               now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrCodexConnectionStale
	}
	c.Status = status
	c.BundleCiphertext = ciphertext
	c.AccountLabel = accountLabel
	c.CredentialVersion = version
	c.FinalCheckpointVersion = finalVersion
	c.VerificationURL = ""
	c.VerificationCode = ""
	c.UpdatedAt = now
	return nil
}

// ReserveFactoryCodexConnection adds one run lease.
// Any number of run leases may exist. A login lease blocks new runs.
func ReserveFactoryCodexConnection(
	tx *gorm.DB,
	organizationID, factoryID, runID, taskID uuid.UUID,
	now time.Time,
	encryptor crypto.Encryptor,
	selectedModels ...string,
) (*FactoryCodexConnection, error) {
	var connection *FactoryCodexConnection
	err := tx.Transaction(func(tx *gorm.DB) error {
		if err := requireCodexFactoriesAccess(tx, organizationID); err != nil {
			return err
		}
		ownerID, err := RequireSubscriptionRunOwner(tx, organizationID, factoryID, runID)
		if err != nil {
			return err
		}
		locked, err := LockCodexConnection(tx, organizationID, ownerID)
		if err != nil {
			return err
		}
		if locked.RevocationPendingAt != nil {
			return ErrCodexConnectionNotReady
		}
		login, err := FindCodexLoginLease(tx, locked.ID)
		if err != nil {
			return err
		}
		if locked.Status != CodexConnectionStatusReady || locked.CredentialVersion < 1 || len(locked.BundleCiphertext) == 0 {
			if login != nil {
				return &CodexConnectionBusyError{TaskID: login.TaskID}
			}
			return ErrCodexConnectionNotReady
		}
		if login != nil {
			return &CodexConnectionBusyError{TaskID: login.TaskID}
		}
		if len(selectedModels) == 0 {
			selectedModels = []string{""}
		}
		for _, selected := range selectedModels {
			resolved, err := locked.resolvePermittedModel(tx, factoryID, selected)
			if err != nil {
				return err
			}
			// The model is fixed in the execution snapshot. A blank model must
			// not delegate a restricted choice to the native default.
			if strings.TrimSpace(selected) == "" && resolved != "" {
				return ErrSubscriptionModelUnavailable
			}
		}
		if _, err := locked.createLease(tx, taskID, CodexLeasePurposeRun, &factoryID, encryptor, now); err != nil {
			return err
		}
		locked.UpdatedAt = now
		connection = locked
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connection, nil
}

func (c *FactoryCodexConnection) Disconnect(tx *gorm.DB, now time.Time) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		if err := c.relock(tx); err != nil {
			return err
		}
		count, err := CountCodexLeases(tx, c.ID)
		if err != nil {
			return err
		}
		if count > 0 {
			return ErrCodexConnectionActive
		}
		generation := c.ReservationGeneration + 1
		result := tx.Model(c).
			Where("id = ? AND reservation_generation = ?", c.ID, c.ReservationGeneration).
			Updates(map[string]any{
				"status":                   CodexConnectionStatusDisconnected,
				"resume_status":            "",
				"verification_url":         "",
				"verification_code":        "",
				"account_label":            "",
				"usage_limits":             datatypes.NewJSONSlice([]CodexUsageLimit{}),
				"available_models":         datatypes.NewJSONSlice([]CodexModel{}),
				"usage_observed_at":        nil,
				"bundle_ciphertext":        nil,
				"credential_version":       0,
				"final_checkpoint_version": 0,
				"reservation_generation":   generation,
				"updated_at":               now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrCodexConnectionActive
		}
		c.Status = CodexConnectionStatusDisconnected
		c.ResumeStatus = ""
		c.VerificationURL = ""
		c.VerificationCode = ""
		c.AccountLabel = ""
		c.UsageLimits = nil
		c.AvailableModels = nil
		c.UsageObservedAt = nil
		c.BundleCiphertext = nil
		c.CredentialVersion = 0
		c.FinalCheckpointVersion = 0
		c.ReservationGeneration = generation
		c.LoginLease = nil
		c.UpdatedAt = now
		return nil
	})
}

// MarkCodexConnectionUncertain keeps a login lease and requires a new sign-in.
// A lost run does not change the shared subscription.
func MarkCodexConnectionUncertain(tx *gorm.DB, taskID uuid.UUID, now time.Time) error {
	lease, err := FindCodexLeaseByTask(tx, taskID)
	if err != nil || lease == nil || !lease.Login() {
		return err
	}
	return tx.Model(&FactoryCodexConnection{}).
		Where("id = ? AND revocation_pending_at IS NULL", lease.ConnectionID).
		Updates(map[string]any{
			"status":     CodexConnectionStatusReconnectRequired,
			"updated_at": now,
		}).Error
}

// ReleaseCodexReservationForTask releases the lease for one runner task.
// A run lease ends when the task is terminal. A successful login can also end
// after confirmed local cleanup; other logins wait for VM deletion.
func ReleaseCodexReservationForTask(tx *gorm.DB, taskID uuid.UUID, now time.Time) error {
	lease, err := FindCodexLeaseByTask(tx, taskID)
	if err != nil || lease == nil {
		return err
	}
	return releaseConfirmedCodexLease(tx, lease.ID, now)
}

func ReleaseConfirmedCodexReservations(tx *gorm.DB, now time.Time, connectionIDs ...uuid.UUID) error {
	var ids []uuid.UUID
	query := tx.Model(&FactoryCodexLease{})
	if len(connectionIDs) > 0 {
		query = query.Where("connection_id IN ?", connectionIDs)
	}
	err := query.Pluck("id", &ids).Error
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := releaseConfirmedCodexLease(tx, id, now); err != nil {
			return err
		}
	}
	return nil
}

func releaseConfirmedCodexLease(tx *gorm.DB, id uuid.UUID, now time.Time) error {
	var lease FactoryCodexLease
	err := tx.Where("id = ?", id).First(&lease).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	// Completion locks the task before the connection. Do the same here, and
	// skip tasks held by another transaction so batched cleanup never waits
	// for a task while it holds a connection from an earlier lease.
	task, taskErr := FindRunnerTask(tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}), lease.TaskID)
	if errors.Is(taskErr, ErrRunnerTaskNotFound) {
		if _, err := FindRunnerTask(tx, lease.TaskID); err == nil {
			return nil
		} else if !errors.Is(err, ErrRunnerTaskNotFound) {
			return err
		}
	} else if taskErr != nil {
		return taskErr
	}
	var connection FactoryCodexConnection
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", lease.ConnectionID).
		First(&connection).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", lease.ID).
		First(&lease).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	if taskErr != nil {
		if errors.Is(taskErr, ErrRunnerTaskNotFound) {
			if lease.Login() && lease.RevokedAt == nil && connection.RevocationPendingAt == nil {
				return connection.finishLoginLease(tx, &lease, CodexConnectionStatusReconnectRequired, now)
			}
			return deleteCodexLease(tx, &lease)
		}
		return taskErr
	}
	if !task.IsTerminal() {
		if !lease.Login() || now.Before(lease.CreatedAt.Add(CodexLoginTimeout)) {
			return nil
		}
		if err := task.RequestCancel(tx, now); err != nil {
			return err
		}
		if !task.IsTerminal() {
			return nil
		}
	}
	if !lease.Login() && lease.RevokedAt == nil {
		return deleteCodexLease(tx, &lease)
	}
	if lease.Login() && lease.RevokedAt == nil && connection.RevocationPendingAt == nil &&
		(connection.Status == CodexConnectionStatusFinishing || connection.Status == CodexConnectionStatusReady) &&
		task.State == RunnerTaskStateSucceeded && task.CancelRequestedAt == nil &&
		codexSetupCleaned(task) && codexFinalCheckpointSaved(&connection, &lease) {
		return connection.finishLoginLease(tx, &lease, CodexConnectionStatusReady, now)
	}
	if task.RunnerID != nil {
		runner, runnerErr := FindRunner(tx, *task.RunnerID)
		if runnerErr != nil && !errors.Is(runnerErr, ErrRunnerNotFound) {
			return runnerErr
		}
		if runnerErr == nil && runner.ProviderDeletedAt == nil {
			if task.State == RunnerTaskStateLost && connection.Status != CodexConnectionStatusReconnectRequired {
				return MarkCodexConnectionUncertain(tx, task.ID, now)
			}
			return nil
		}
	}
	if lease.RevokedAt != nil || connection.RevocationPendingAt != nil {
		return deleteCodexLease(tx, &lease)
	}
	return connection.finishLoginLease(tx, &lease, codexLoginLeaseOutcome(&connection, &lease, task), now)
}

func codexSetupCleaned(task *RunnerTask) bool {
	var result struct {
		Cleaned bool `json:"codexSubscriptionSetupCleaned"`
	}
	return json.Unmarshal(task.Result, &result) == nil && result.Cleaned
}

func codexFinalCheckpointSaved(connection *FactoryCodexConnection, lease *FactoryCodexLease) bool {
	return connection.FinalCheckpointVersion > lease.StartedVersion &&
		connection.FinalCheckpointVersion == connection.CredentialVersion
}

func codexLoginLeaseOutcome(connection *FactoryCodexConnection, lease *FactoryCodexLease, task *RunnerTask) string {
	if codexFinalCheckpointSaved(connection, lease) {
		return CodexConnectionStatusReady
	}
	if connection.CredentialVersion > lease.StartedVersion {
		return CodexConnectionStatusReconnectRequired
	}
	switch task.State {
	case RunnerTaskStateSucceeded:
		return CodexConnectionStatusReconnectRequired
	case RunnerTaskStateLost:
		return CodexConnectionStatusReconnectRequired
	case RunnerTaskStateCanceled:
		if connection.ResumeStatus != "" {
			return connection.ResumeStatus
		}
		if connection.CredentialVersion > 0 {
			return CodexConnectionStatusReconnectRequired
		}
		return CodexConnectionStatusDisconnected
	case RunnerTaskStateFailed:
		if connection.ResumeStatus != "" {
			return connection.ResumeStatus
		}
		return CodexConnectionStatusReconnectRequired
	default:
		return CodexConnectionStatusReconnectRequired
	}
}

func (c *FactoryCodexConnection) finishLoginLease(tx *gorm.DB, lease *FactoryCodexLease, status string, now time.Time) error {
	if lease == nil {
		return nil
	}
	if err := deleteCodexLease(tx, lease); err != nil {
		return err
	}
	generation := c.ReservationGeneration + 1
	result := tx.Model(c).
		Where("id = ?", c.ID).
		Updates(map[string]any{
			"status":                 status,
			"resume_status":          "",
			"reservation_generation": generation,
			"verification_url":       "",
			"verification_code":      "",
			"updated_at":             now,
		})
	if result.Error != nil {
		return result.Error
	}
	c.Status = status
	c.ResumeStatus = ""
	c.LoginLease = nil
	c.ReservationGeneration = generation
	c.VerificationURL = ""
	c.VerificationCode = ""
	c.UpdatedAt = now
	return nil
}

func deleteCodexLease(tx *gorm.DB, lease *FactoryCodexLease) error {
	result := tx.Where("id = ?", lease.ID).Delete(&FactoryCodexLease{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (c *FactoryCodexConnection) requireLoginLease(tx *gorm.DB, taskID uuid.UUID, generation int64) (*FactoryCodexLease, error) {
	lease, err := FindCodexLeaseForTask(tx, c.ID, taskID, generation)
	if err != nil {
		return nil, err
	}
	if !lease.Login() {
		return nil, ErrCodexConnectionStale
	}
	return lease, nil
}

// StoreRefreshedBundle saves tokens refreshed by SuperPlane for every run.
func (c *FactoryCodexConnection) StoreRefreshedBundle(tx *gorm.DB, ciphertext []byte, accountLabel string, now time.Time) error {
	version := c.CredentialVersion + 1
	result := tx.Model(c).
		Where("id = ? AND credential_version = ? AND revocation_pending_at IS NULL", c.ID, c.CredentialVersion).
		Updates(map[string]any{
			"bundle_ciphertext":  ciphertext,
			"account_label":      accountLabel,
			"credential_version": version,
			"updated_at":         now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrCodexConnectionStale
	}
	c.BundleCiphertext = ciphertext
	c.AccountLabel = accountLabel
	c.CredentialVersion = version
	c.UpdatedAt = now
	return nil
}

// MarkReconnectRequired records that the stored refresh token can no longer be used.
func (c *FactoryCodexConnection) MarkReconnectRequired(tx *gorm.DB, now time.Time) error {
	result := tx.Model(c).
		Where("id = ?", c.ID).
		Updates(map[string]any{
			"status":     CodexConnectionStatusReconnectRequired,
			"updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	c.Status = CodexConnectionStatusReconnectRequired
	c.UpdatedAt = now
	return nil
}

// ActiveCodexMember requires an active organization and locks membership before
// connection mutation. Member removal takes the same lock before revocation.
func ActiveCodexMember(tx *gorm.DB, organizationID, userID uuid.UUID) (*User, error) {
	_, err := FindOrganizationByIDInTransaction(tx, organizationID.String())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCodexConnectionNotEligible
	}
	if err != nil {
		return nil, err
	}
	var user User
	err = tx.Clauses(clause.Locking{Strength: "SHARE"}).
		Where("organization_id = ? AND id = ? AND type = ?", organizationID, userID, UserTypeHuman).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCodexConnectionNotEligible
	}
	return &user, err
}

func ValidateCodexUsageLimits(limits []CodexUsageLimit) error {
	if len(limits) == 0 || len(limits) > 32 {
		return errors.New("invalid Codex usage limits")
	}
	seen := map[string]bool{}
	for _, limit := range limits {
		if limit.LimitID == "" || len(limit.LimitID) > 256 || len(limit.LimitName) > 256 || len(limit.PlanType) > 64 || seen[limit.LimitID] {
			return errors.New("invalid Codex usage bucket")
		}
		seen[limit.LimitID] = true
		for _, window := range []*CodexUsageWindow{limit.Primary, limit.Secondary} {
			if window == nil {
				continue
			}
			if window.UsedPercent == nil || math.IsNaN(*window.UsedPercent) || math.IsInf(*window.UsedPercent, 0) || *window.UsedPercent < 0 {
				return errors.New("invalid Codex usage percentage")
			}
			if window.WindowDurationMins != nil && *window.WindowDurationMins <= 0 || window.ResetsAt != nil && *window.ResetsAt <= 0 {
				return errors.New("invalid Codex usage window")
			}
			if window.ResetsAt != nil && time.Unix(*window.ResetsAt, 0).UTC().Year() > 9999 {
				return errors.New("Codex usage reset exceeds the API timestamp range")
			}
		}
	}
	return nil
}

func ListOwnedFactoryCodexConnections(tx *gorm.DB, organizationID, ownerUserID uuid.UUID) ([]FactoryCodexConnection, error) {
	var connections []FactoryCodexConnection
	err := tx.Select("id", "status", "account_label", "usage_limits", "usage_observed_at").
		Where("organization_id = ? AND owner_user_id = ? AND status <> ?", organizationID, ownerUserID, CodexConnectionStatusDisconnected).
		Order("created_at, id").Find(&connections).Error
	return connections, err
}

func ValidateCodexModels(availableModels []CodexModel) error {
	for _, model := range availableModels {
		if strings.TrimSpace(model.Model) == "" {
			return errors.New("invalid Codex model")
		}
	}
	return nil
}

func (c *FactoryCodexConnection) StoreUsage(tx *gorm.DB, limits []CodexUsageLimit, availableModels []CodexModel, now time.Time) error {
	updates := map[string]any{}
	if len(limits) > 0 {
		if err := ValidateCodexUsageLimits(limits); err != nil {
			return err
		}
		updates["usage_limits"] = datatypes.NewJSONSlice(limits)
		updates["usage_observed_at"] = now
	}
	if availableModels != nil {
		if err := ValidateCodexModels(availableModels); err != nil {
			return err
		}
		updates["available_models"] = datatypes.NewJSONSlice(availableModels)
	}
	if len(updates) == 0 {
		return nil
	}
	return tx.Model(c).Updates(updates).Error
}

func FactoryCodexSubscriptionModels(tx *gorm.DB, organizationID, factoryID, ownerID uuid.UUID) ([]string, error) {
	available, err := FactoryCodexAvailableSubscriptionModels(tx, organizationID, factoryID, ownerID)
	if err != nil {
		return nil, err
	}
	return ApplyFactoryLLMModelPolicy(tx, factoryID, UsageProviderOpenAI, UsageFundingSourceSubscription, available)
}

// FactoryCodexAvailableSubscriptionModels returns the owner's native catalog
// before the workspace policy is applied.
func FactoryCodexAvailableSubscriptionModels(tx *gorm.DB, organizationID, factoryID, ownerID uuid.UUID) ([]string, error) {
	connection, err := FindCodexConnection(tx, organizationID, ownerID)
	if errors.Is(err, ErrCodexConnectionNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if connection.Status != CodexConnectionStatusReady || connection.RevocationPendingAt != nil || connection.CredentialVersion < 1 || len(connection.BundleCiphertext) == 0 {
		return nil, nil
	}
	models := make([]string, 0, len(connection.AvailableModels))
	for _, model := range connection.AvailableModels {
		models = append(models, model.Model)
	}
	return CompactModelIDs(models), nil
}

// ResolveFactoryCodexSubscriptionModel preserves the native default while
// inheriting. Restricted workspaces always dispatch a permitted native ID.
func ResolveFactoryCodexSubscriptionModel(tx *gorm.DB, organizationID, factoryID, ownerID uuid.UUID, requested string) (string, error) {
	connection, err := FindCodexConnection(tx, organizationID, ownerID)
	if err != nil {
		return "", err
	}
	return connection.resolvePermittedModel(tx, factoryID, requested)
}

func (c *FactoryCodexConnection) resolvePermittedModel(tx *gorm.DB, factoryID uuid.UUID, requested string) (string, error) {
	model := strings.TrimSpace(requested)
	policy, err := FindFactoryLLMModelAllowlist(tx, factoryID, UsageProviderOpenAI, UsageFundingSourceSubscription)
	if err != nil {
		return "", err
	}
	if model == "" && policy.InheritsParent() {
		return "", nil
	}
	available := make([]string, 0, len(c.AvailableModels))
	for _, candidate := range c.AvailableModels {
		available = append(available, candidate.Model)
	}
	permitted, err := ApplyFactoryLLMModelPolicy(tx, factoryID, UsageProviderOpenAI, UsageFundingSourceSubscription, CompactModelIDs(available))
	if err != nil {
		return "", err
	}
	if model != "" {
		if !slices.Contains(permitted, model) {
			return "", ErrSubscriptionModelUnavailable
		}
		return model, nil
	}
	if len(permitted) == 0 {
		return "", fmt.Errorf("No ChatGPT models are enabled for this task owner. Update Workspace Models: %w", ErrSubscriptionModelUnavailable)
	}
	for _, candidate := range c.AvailableModels {
		if candidate.IsDefault && slices.Contains(permitted, candidate.Model) {
			return candidate.Model, nil
		}
	}
	slices.Sort(permitted)
	return permitted[0], nil
}

func requireCodexFactoriesAccess(tx *gorm.DB, organizationID uuid.UUID) error {
	enabled, err := OrganizationHasExperimentalFeatures(tx, organizationID, features.FeatureFactories)
	if err != nil {
		return err
	}
	if !enabled {
		return ErrCodexConnectionUnavailable
	}
	return nil
}

// ListEligibleCodexConnections supplies the combined workspace policy catalog.
// Task admission still checks the actual owner's connection and native catalog.
func ListEligibleCodexConnections(tx *gorm.DB, organizationID uuid.UUID) ([]FactoryCodexConnection, error) {
	var connections []FactoryCodexConnection
	err := tx.Model(&FactoryCodexConnection{}).Select("factory_codex_connections.*").
		Joins("JOIN users ON users.id = factory_codex_connections.owner_user_id AND users.organization_id = factory_codex_connections.organization_id").
		Where("factory_codex_connections.organization_id = ? AND users.deleted_at IS NULL AND users.type = ?", organizationID, UserTypeHuman).
		Where("factory_codex_connections.status = ? AND revocation_pending_at IS NULL AND credential_version > 0 AND octet_length(bundle_ciphertext) > 0", CodexConnectionStatusReady).
		Order("factory_codex_connections.id").Find(&connections).Error
	return connections, err
}

// DeleteDrainedOrganizationCodexConnections must run before organization users
// are removed. Leases retain evidence until their runners finish cleanup.
func DeleteDrainedOrganizationCodexConnections(tx *gorm.DB, organizationID uuid.UUID) (bool, error) {
	var connections []FactoryCodexConnection
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id = ?", organizationID).Order("id").Find(&connections).Error; err != nil {
		return false, err
	}
	for _, connection := range connections {
		count, err := CountCodexLeases(tx, connection.ID)
		if err != nil {
			return false, err
		}
		if count > 0 {
			return false, nil
		}
	}
	return true, tx.Where("organization_id = ?", organizationID).Delete(&FactoryCodexConnection{}).Error
}
