package models

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CodexLoginTimeout = 15 * time.Minute

	CodexLeasePurposeLogin = "login"
	CodexLeasePurposeRun   = "run"
)

// FactoryCodexLease is one task using a factory ChatGPT connection.
// A login lease is exclusive and may checkpoint the real auth bundle.
// Run leases share one connection. Each run lease secret stands in for the
// OpenAI refresh token on that runner.
type FactoryCodexLease struct {
	ID               uuid.UUID
	ConnectionID     uuid.UUID
	TaskID           uuid.UUID
	Generation       int64
	Purpose          string
	SecretHash       string
	SecretCiphertext []byte
	StartedVersion   int64
	CreatedAt        time.Time
	RevokedAt        *time.Time
	Secret           string `gorm:"-"`
}

func (FactoryCodexLease) TableName() string {
	return "factory_codex_leases"
}

func (l *FactoryCodexLease) Login() bool {
	return l != nil && l.Purpose == CodexLeasePurposeLogin
}

func FindCodexLoginLease(tx *gorm.DB, connectionID uuid.UUID) (*FactoryCodexLease, error) {
	var lease FactoryCodexLease
	err := tx.Where("connection_id = ? AND purpose = ?", connectionID, CodexLeasePurposeLogin).First(&lease).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &lease, nil
}

// ListCodexLoginConnections captures connection state before lease cleanup.
// Callers use the snapshot to publish changes after their transaction commits.
func ListCodexLoginConnections(tx *gorm.DB) ([]FactoryCodexConnection, error) {
	var connections []FactoryCodexConnection
	err := tx.Select("id", "factory_id", "status", "reservation_generation").
		Where("id IN (?)", tx.Model(&FactoryCodexLease{}).Select("connection_id").Where("purpose = ?", CodexLeasePurposeLogin)).
		Find(&connections).Error
	return connections, err
}

func FindCodexLeaseForTask(tx *gorm.DB, connectionID, taskID uuid.UUID, generation int64) (*FactoryCodexLease, error) {
	var lease FactoryCodexLease
	err := tx.Where(
		"connection_id = ? AND task_id = ? AND generation = ? AND revoked_at IS NULL",
		connectionID, taskID, generation,
	).First(&lease).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCodexConnectionStale
	}
	if err != nil {
		return nil, err
	}
	return &lease, nil
}

func FindCodexLeaseByTask(tx *gorm.DB, taskID uuid.UUID) (*FactoryCodexLease, error) {
	var lease FactoryCodexLease
	err := tx.Where("task_id = ?", taskID).First(&lease).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &lease, nil
}

func FindCodexRunLeaseBySecret(tx *gorm.DB, secret string) (*FactoryCodexLease, error) {
	if secret == "" {
		return nil, ErrCodexConnectionStale
	}
	var lease FactoryCodexLease
	err := tx.Where(
		"purpose = ? AND secret_hash = ? AND revoked_at IS NULL",
		CodexLeasePurposeRun, codexLeaseSecretHash(secret),
	).First(&lease).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCodexConnectionStale
	}
	if err != nil {
		return nil, err
	}
	return &lease, nil
}

func CountCodexLeases(tx *gorm.DB, connectionID uuid.UUID) (int64, error) {
	var count int64
	err := tx.Model(&FactoryCodexLease{}).Where("connection_id = ?", connectionID).Count(&count).Error
	return count, err
}

func (c *FactoryCodexConnection) AttachLoginLease(tx *gorm.DB) error {
	if c == nil {
		return nil
	}
	lease, err := FindCodexLoginLease(tx, c.ID)
	if err != nil {
		return err
	}
	c.LoginLease = lease
	c.LoginCancellationRequested = false
	if lease == nil {
		return nil
	}
	task, err := FindRunnerTask(tx, lease.TaskID)
	if errors.Is(err, ErrRunnerTaskNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	c.LoginCancellationRequested = task.CancelRequestedAt != nil || task.State == RunnerTaskStateCanceled
	return nil
}

// RequireSubscriptionLeaseAdmission rechecks revocation and membership before
// allocating a runner. It does not serialize the execution of separate leases.
func (t *RunnerTask) RequireSubscriptionLeaseAdmission(tx *gorm.DB) error {
	lease, err := FindCodexLeaseByTask(tx, t.ID)
	if err != nil {
		return err
	}
	if lease == nil {
		// Deleting a connection also deletes its leases. The execution retains
		// the credential source, so this task must not run with a stale token.
		execution, err := t.FindExecution(tx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		credentials, _ := execution.Configuration.Data()["credentials"].(map[string]any)
		source, _ := credentials["source"].(string)
		if strings.TrimSpace(source) == "subscription" {
			return ErrCodexConnectionNotFound
		}
		return nil
	}
	if lease.RevokedAt != nil {
		return ErrCodexConnectionStale
	}
	connection, err := FindFactoryCodexConnectionByID(tx, lease.ConnectionID)
	if err != nil {
		return err
	}
	if _, err := ActiveCodexMember(tx, connection.OrganizationID, connection.OwnerUserID); err != nil {
		return err
	}
	// Re-read after locking membership. Removal can revoke the snapshot above
	// before this transaction acquires its membership lock.
	connection, err = FindFactoryCodexConnectionByID(tx.Clauses(clause.Locking{Strength: "SHARE"}), lease.ConnectionID)
	if err != nil {
		return err
	}
	lease, err = FindCodexLeaseForTask(tx, connection.ID, t.ID, lease.Generation)
	if err != nil {
		return err
	}
	if connection.RevocationPendingAt != nil {
		return ErrCodexConnectionStale
	}
	if lease.Login() {
		return nil
	}
	if connection.Status != CodexConnectionStatusReady || len(connection.BundleCiphertext) == 0 {
		return ErrCodexConnectionNotReady
	}
	execution, err := t.FindExecution(tx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrSubscriptionTaskAssociation
	}
	if err != nil {
		return err
	}
	order, err := FindSubscriptionWorkOrderForRun(tx, connection.OrganizationID, connection.FactoryID, execution.RunID)
	if err != nil {
		return err
	}
	ownerID, err := order.RequireSubscriptionOwner(tx, true)
	if err != nil {
		return err
	}
	if ownerID != connection.OwnerUserID {
		return ErrSubscriptionOwnerInvalid
	}
	model, _ := execution.Configuration.Data()["model"].(string)
	if model == "" {
		return nil
	}
	allowed, err := RunnerAcceptsModel(tx, connection.OrganizationID, &connection.FactoryID, "runnerCodex", map[string]any{"credentials": map[string]any{"source": "subscription"}}, model, ownerID)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrSubscriptionModelUnavailable
	}
	return nil
}

func (c *FactoryCodexConnection) createLease(
	tx *gorm.DB,
	taskID uuid.UUID,
	purpose string,
	encryptor crypto.Encryptor,
	now time.Time,
) (*FactoryCodexLease, error) {
	generation := c.ReservationGeneration + 1
	lease := &FactoryCodexLease{
		ID:             uuid.New(),
		ConnectionID:   c.ID,
		TaskID:         taskID,
		Generation:     generation,
		Purpose:        purpose,
		StartedVersion: c.CredentialVersion,
		CreatedAt:      now,
	}
	if purpose == CodexLeasePurposeRun {
		plain, hash, ciphertext, err := newCodexLeaseSecret(encryptor, c.ID)
		if err != nil {
			return nil, err
		}
		lease.Secret = plain
		lease.SecretHash = hash
		lease.SecretCiphertext = ciphertext
	}
	if err := tx.Create(lease).Error; err != nil {
		return nil, err
	}
	result := tx.Model(c).
		Where("id = ? AND reservation_generation = ?", c.ID, c.ReservationGeneration).
		Updates(map[string]any{
			"reservation_generation": generation,
			"updated_at":             now,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, ErrCodexConnectionStale
	}
	c.ReservationGeneration = generation
	if lease.Login() {
		c.LoginLease = lease
	}
	return lease, nil
}

func (c *FactoryCodexConnection) relock(tx *gorm.DB) error {
	if _, err := activeHumanCodexMember(tx, c.OrganizationID, c.OwnerUserID); err != nil {
		return err
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", c.ID).First(c).Error; err != nil {
		return err
	}
	if c.RevocationPendingAt != nil {
		return ErrCodexConnectionNotReady
	}
	return nil
}

func (c *FactoryCodexConnection) busyLease(tx *gorm.DB) (*FactoryCodexLease, error) {
	login, err := FindCodexLoginLease(tx, c.ID)
	if err != nil || login != nil {
		return login, err
	}
	var run FactoryCodexLease
	err = tx.Where("connection_id = ? AND purpose = ?", c.ID, CodexLeasePurposeRun).
		Order("created_at, id").
		First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func newCodexLeaseSecret(encryptor crypto.Encryptor, connectionID uuid.UUID) (string, string, []byte, error) {
	if encryptor == nil {
		return "", "", nil, errors.New("encryptor is required")
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", nil, err
	}
	plain := base64.RawURLEncoding.EncodeToString(buf)
	ciphertext, err := encryptor.Encrypt(context.Background(), []byte(plain), []byte(connectionID.String()))
	if err != nil {
		return "", "", nil, err
	}
	return plain, codexLeaseSecretHash(plain), ciphertext, nil
}

func codexLeaseSecretHash(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
