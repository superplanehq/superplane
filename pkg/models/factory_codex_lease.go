package models

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
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

func FindCodexLeaseForTask(tx *gorm.DB, connectionID, taskID uuid.UUID, generation int64) (*FactoryCodexLease, error) {
	var lease FactoryCodexLease
	err := tx.Where(
		"connection_id = ? AND task_id = ? AND generation = ?",
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
		"purpose = ? AND secret_hash = ?",
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
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", c.ID).
		First(c).Error
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
