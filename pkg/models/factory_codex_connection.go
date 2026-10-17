package models

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/features"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CodexConnectionStatusPending           = "pending"
	CodexConnectionStatusFinishing         = "finishing"
	CodexConnectionStatusReady             = "ready"
	CodexConnectionStatusReconnectRequired = "reconnect_required"
	CodexConnectionStatusDisconnected      = "disconnected"
)

var (
	ErrCodexConnectionNotFound    = errors.New("chatgpt subscription connection not found")
	ErrCodexConnectionNotEligible = errors.New("this organization must have one member to use a ChatGPT subscription")
	ErrCodexConnectionNotReady    = errors.New("connect ChatGPT before this task can run")
	ErrCodexConnectionStale       = errors.New("the ChatGPT sign-in update is out of date")
	ErrCodexConnectionActive      = errors.New("stop the active task before you disconnect ChatGPT")
	ErrCodexConnectionUnavailable = errors.New("ChatGPT subscription is not available for this organization")
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

// FactoryCodexConnection is one factory-bound ChatGPT subscription.
// The ciphertext is the native Codex auth bundle, bound to this row ID.
type FactoryCodexConnection struct {
	ID                        uuid.UUID
	OrganizationID            uuid.UUID
	FactoryID                 uuid.UUID
	OwnerUserID               uuid.UUID
	Status                    string
	ResumeStatus              string
	VerificationURL           string
	VerificationCode          string
	AccountLabel              string
	UsageLimits               datatypes.JSONSlice[CodexUsageLimit]
	UsageObservedAt           *time.Time
	BundleCiphertext          []byte
	CredentialVersion         int64
	FinalCheckpointVersion    int64
	ReservationStartedVersion int64
	ReservationTaskID         *uuid.UUID
	ReservationGeneration     int64
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
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

func (FactoryCodexConnection) TableName() string {
	return "factory_codex_connections"
}

func (c *FactoryCodexConnection) Reserved() bool {
	return c != nil && c.ReservationTaskID != nil
}

func FindFactoryCodexConnection(tx *gorm.DB, organizationID, factoryID uuid.UUID) (*FactoryCodexConnection, error) {
	var connection FactoryCodexConnection
	err := tx.
		Where("organization_id = ? AND factory_id = ?", organizationID, factoryID).
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

func LockFactoryCodexConnection(tx *gorm.DB, organizationID, factoryID uuid.UUID) (*FactoryCodexConnection, error) {
	var connection FactoryCodexConnection
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ? AND factory_id = ?", organizationID, factoryID).
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

func CreateFactoryCodexConnection(
	tx *gorm.DB,
	organizationID, factoryID, ownerUserID uuid.UUID,
	now time.Time,
) (*FactoryCodexConnection, error) {
	connection := &FactoryCodexConnection{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		FactoryID:      factoryID,
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
// The reservation stays until the login runner VM is confirmed deleted.
func (c *FactoryCodexConnection) BeginLogin(tx *gorm.DB, taskID uuid.UUID, now time.Time) error {
	if c.Reserved() {
		return &CodexConnectionBusyError{TaskID: *c.ReservationTaskID}
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

	generation := c.ReservationGeneration + 1
	result := tx.Model(c).
		Where("id = ? AND reservation_task_id IS NULL AND status = ?", c.ID, c.Status).
		Updates(map[string]any{
			"status":                      CodexConnectionStatusPending,
			"usage_limits":                datatypes.NewJSONSlice([]CodexUsageLimit{}),
			"usage_observed_at":           nil,
			"resume_status":               c.Status,
			"verification_url":            "",
			"verification_code":           "",
			"reservation_task_id":         taskID,
			"reservation_generation":      generation,
			"reservation_started_version": c.CredentialVersion,
			"updated_at":                  now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrCodexConnectionStale
	}
	c.ResumeStatus = c.Status
	c.UsageLimits = nil
	c.UsageObservedAt = nil
	c.Status = CodexConnectionStatusPending
	c.VerificationURL = ""
	c.VerificationCode = ""
	c.ReservationTaskID = &taskID
	c.ReservationGeneration = generation
	c.ReservationStartedVersion = c.CredentialVersion
	c.UpdatedAt = now
	return nil
}

func (c *FactoryCodexConnection) RecordDeviceCode(
	tx *gorm.DB,
	taskID uuid.UUID,
	generation int64,
	verificationURL, verificationCode string,
	now time.Time,
) error {
	if err := c.requireReservation(taskID, generation); err != nil {
		return err
	}
	if c.Status != CodexConnectionStatusPending {
		return ErrCodexConnectionStale
	}
	result := tx.Model(c).
		Where("id = ? AND reservation_task_id = ? AND reservation_generation = ? AND status = ?",
			c.ID, taskID, generation, CodexConnectionStatusPending).
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
// Login stays in finishing until the runner VM is confirmed deleted.
func (c *FactoryCodexConnection) StoreBundle(
	tx *gorm.DB,
	taskID uuid.UUID,
	generation, version int64,
	ciphertext []byte,
	accountLabel string,
	final bool,
	now time.Time,
) error {
	if err := c.requireReservation(taskID, generation); err != nil {
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
		Where("id = ? AND reservation_task_id = ? AND reservation_generation = ? AND credential_version = ?",
			c.ID, taskID, generation, c.CredentialVersion).
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

// ReserveForTask claims a ready connection for one factory task.
func ReserveFactoryCodexConnection(
	tx *gorm.DB,
	organizationID, factoryID, taskID uuid.UUID,
	now time.Time,
) (*FactoryCodexConnection, error) {
	if err := requireCodexSubscriptionEligible(tx, organizationID); err != nil {
		return nil, err
	}
	connection, err := LockFactoryCodexConnection(tx, organizationID, factoryID)
	if err != nil {
		if errors.Is(err, ErrCodexConnectionNotFound) {
			return nil, ErrCodexConnectionNotReady
		}
		return nil, err
	}
	if connection.Status != CodexConnectionStatusReady || connection.CredentialVersion < 1 || len(connection.BundleCiphertext) == 0 {
		if connection.Reserved() {
			return nil, &CodexConnectionBusyError{TaskID: *connection.ReservationTaskID}
		}
		return nil, ErrCodexConnectionNotReady
	}
	if connection.Reserved() {
		return nil, &CodexConnectionBusyError{TaskID: *connection.ReservationTaskID}
	}
	generation := connection.ReservationGeneration + 1
	result := tx.Model(connection).
		Where("id = ? AND status = ? AND reservation_task_id IS NULL", connection.ID, CodexConnectionStatusReady).
		Updates(map[string]any{
			"reservation_task_id":         taskID,
			"reservation_generation":      generation,
			"reservation_started_version": connection.CredentialVersion,
			"updated_at":                  now,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, ErrCodexConnectionStale
	}
	connection.ReservationTaskID = &taskID
	connection.ReservationGeneration = generation
	connection.ReservationStartedVersion = connection.CredentialVersion
	connection.UpdatedAt = now
	return connection, nil
}

func (c *FactoryCodexConnection) Disconnect(tx *gorm.DB, now time.Time) error {
	if c.Reserved() {
		return ErrCodexConnectionActive
	}
	generation := c.ReservationGeneration + 1
	result := tx.Model(c).
		Where("id = ? AND reservation_task_id IS NULL", c.ID).
		Updates(map[string]any{
			"status":                   CodexConnectionStatusDisconnected,
			"resume_status":            "",
			"verification_url":         "",
			"verification_code":        "",
			"account_label":            "",
			"usage_limits":             datatypes.NewJSONSlice([]CodexUsageLimit{}),
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
	c.UsageObservedAt = nil
	c.BundleCiphertext = nil
	c.CredentialVersion = 0
	c.FinalCheckpointVersion = 0
	c.ReservationGeneration = generation
	c.UpdatedAt = now
	return nil
}

// MarkCodexConnectionUncertain keeps the reservation and requires a new sign-in.
// A missed heartbeat is not proof that the runner process has stopped.
func MarkCodexConnectionUncertain(tx *gorm.DB, taskID uuid.UUID, now time.Time) error {
	return tx.Model(&FactoryCodexConnection{}).
		Where("reservation_task_id = ?", taskID).
		Updates(map[string]any{
			"status":     CodexConnectionStatusReconnectRequired,
			"updated_at": now,
		}).Error
}

// ReleaseCodexReservationForTask releases one reservation after its runner
// task is terminal and, when a VM was assigned, after that VM is confirmed deleted.
func ReleaseCodexReservationForTask(tx *gorm.DB, taskID uuid.UUID, now time.Time) error {
	var ids []uuid.UUID
	err := tx.Model(&FactoryCodexConnection{}).
		Where("reservation_task_id = ?", taskID).
		Limit(1).
		Pluck("id", &ids).
		Error
	if err != nil || len(ids) == 0 {
		return err
	}
	return releaseConfirmedCodexReservation(tx, ids[0], now)
}

func ReleaseConfirmedCodexReservations(tx *gorm.DB, now time.Time) error {
	var ids []uuid.UUID
	err := tx.Model(&FactoryCodexConnection{}).
		Where("reservation_task_id IS NOT NULL").
		Pluck("id", &ids).
		Error
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := releaseConfirmedCodexReservation(tx, id, now); err != nil {
			return err
		}
	}
	return nil
}

func releaseConfirmedCodexReservation(tx *gorm.DB, id uuid.UUID, now time.Time) error {
	var connection FactoryCodexConnection
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&connection).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if connection.ReservationTaskID == nil {
		return nil
	}

	task, err := FindRunnerTask(tx, *connection.ReservationTaskID)
	if err != nil {
		if errors.Is(err, ErrRunnerTaskNotFound) {
			return connection.finishReservation(tx, CodexConnectionStatusReconnectRequired, now)
		}
		return err
	}
	if !task.IsTerminal() {
		return nil
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

	return connection.finishReservation(tx, codexReservationOutcome(&connection, task), now)
}

func codexReservationOutcome(connection *FactoryCodexConnection, task *RunnerTask) string {
	checkpointSaved := connection.FinalCheckpointVersion > connection.ReservationStartedVersion
	switch task.State {
	case RunnerTaskStateSucceeded:
		if checkpointSaved {
			return CodexConnectionStatusReady
		}
		return CodexConnectionStatusReconnectRequired
	case RunnerTaskStateLost:
		return CodexConnectionStatusReconnectRequired
	case RunnerTaskStateCanceled:
		if checkpointSaved {
			return CodexConnectionStatusReady
		}
		if connection.ResumeStatus != "" {
			return connection.ResumeStatus
		}
		if connection.CredentialVersion > 0 {
			return CodexConnectionStatusReconnectRequired
		}
		return CodexConnectionStatusDisconnected
	case RunnerTaskStateFailed:
		if !checkpointSaved && connection.ResumeStatus != "" {
			return connection.ResumeStatus
		}
		return CodexConnectionStatusReconnectRequired
	default:
		return CodexConnectionStatusReconnectRequired
	}
}

func (c *FactoryCodexConnection) finishReservation(tx *gorm.DB, status string, now time.Time) error {
	if c.ReservationTaskID == nil {
		return nil
	}
	generation := c.ReservationGeneration + 1
	updates := map[string]any{
		"status":                 status,
		"resume_status":          "",
		"reservation_task_id":    nil,
		"reservation_generation": generation,
		"verification_url":       "",
		"verification_code":      "",
		"updated_at":             now,
	}
	if status != CodexConnectionStatusReady {
		updates["verification_url"] = ""
		updates["verification_code"] = ""
	}
	result := tx.Model(c).
		Where("id = ? AND reservation_task_id = ?", c.ID, *c.ReservationTaskID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return nil
	}
	c.Status = status
	c.ResumeStatus = ""
	c.ReservationTaskID = nil
	c.ReservationGeneration = generation
	c.VerificationURL = ""
	c.VerificationCode = ""
	c.UpdatedAt = now
	return nil
}

func (c *FactoryCodexConnection) requireReservation(taskID uuid.UUID, generation int64) error {
	if c.ReservationTaskID == nil || *c.ReservationTaskID != taskID || c.ReservationGeneration != generation {
		return ErrCodexConnectionStale
	}
	return nil
}

func requireCodexSubscriptionEligible(tx *gorm.DB, organizationID uuid.UUID) error {
	organization, err := FindOrganizationByIDInTransaction(tx, organizationID.String())
	if err != nil {
		return err
	}
	if !organization.HasExperimentalFeature(features.FeatureCodexSubscriptions) {
		return ErrCodexConnectionUnavailable
	}
	count, err := CountActiveHumanUsersByOrganizationInTransaction(tx, organizationID.String())
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrCodexConnectionNotEligible
	}
	return nil
}

// SoleHumanOrganizationMember returns the only active human member.
func SoleHumanOrganizationMember(tx *gorm.DB, organizationID uuid.UUID) (*User, error) {
	if err := requireCodexSubscriptionEligible(tx, organizationID); err != nil {
		return nil, err
	}
	var users []User
	err := tx.
		Where("organization_id = ? AND type = ?", organizationID, UserTypeHuman).
		Find(&users).
		Error
	if err != nil {
		return nil, err
	}
	if len(users) != 1 {
		return nil, ErrCodexConnectionNotEligible
	}
	return &users[0], nil
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
	err := tx.Select("id", "factory_id", "status", "account_label", "usage_limits", "usage_observed_at").
		Where("organization_id = ? AND owner_user_id = ? AND status <> ?", organizationID, ownerUserID, CodexConnectionStatusDisconnected).
		Order("created_at, id").Find(&connections).Error
	return connections, err
}

func (c *FactoryCodexConnection) StoreUsage(tx *gorm.DB, limits []CodexUsageLimit, now time.Time) error {
	if err := ValidateCodexUsageLimits(limits); err != nil {
		return err
	}
	return tx.Model(c).Updates(map[string]any{
		"usage_limits":      datatypes.NewJSONSlice(limits),
		"usage_observed_at": now,
	}).Error
}
