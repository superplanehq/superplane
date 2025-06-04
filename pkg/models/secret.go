package models

import (
	"strings"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm/clause"
)

type Secret struct {
	ID        uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	CanvasID  uuid.UUID
	Name      string
	CreatedAt *time.Time
	CreatedBy uuid.UUID
	UpdatedAt *time.Time
	Provider  string
	Data      []byte
}

type SecretData struct {
	Local map[string]string `json:"local"`
}

func (s *Secret) UpdateData(data []byte) (*Secret, error) {
	now := time.Now()

	err := database.Conn().
		Model(s).
		Clauses(clause.Returning{}).
		Where("id = ?", s.ID).
		Update("data", data).
		Update("updated_at", &now).
		Error

	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Secret) Delete() error {
	return database.Conn().Delete(s).Error
}

func FindSecretByName(canvasID, name string) (*Secret, error) {
	var secret Secret

	err := database.Conn().
		Where("canvas_id = ?", canvasID).
		Where("name = ?", name).
		First(&secret).
		Error

	if err != nil {
		return nil, err
	}

	return &secret, nil
}

func FindSecretByID(canvasID, id string) (*Secret, error) {
	var secret Secret

	err := database.Conn().
		Where("canvas_id = ?", canvasID).
		Where("id = ?", id).
		First(&secret).
		Error

	if err != nil {
		return nil, err
	}

	return &secret, nil
}

func CreateSecret(name, provider, requesterID string, canvasID uuid.UUID, data []byte) (*Secret, error) {
	now := time.Now()

	secret := Secret{
		Name:      name,
		CanvasID:  canvasID,
		CreatedAt: &now,
		CreatedBy: uuid.MustParse(requesterID),
		UpdatedAt: &now,
		Provider:  provider,
		Data:      data,
	}

	err := database.Conn().
		Clauses(clause.Returning{}).
		Create(&secret).
		Error

	if err == nil {
		return &secret, nil
	}

	if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return nil, ErrNameAlreadyUsed
	}

	return nil, err
}

func ListSecrets(canvasID string) ([]Secret, error) {
	var secrets []Secret

	err := database.Conn().
		Where("canvas_id = ?", canvasID).
		Find(&secrets).
		Error

	if err != nil {
		return nil, err
	}

	return secrets, nil
}
