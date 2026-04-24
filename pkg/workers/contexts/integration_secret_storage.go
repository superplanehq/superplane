package contexts

import (
	"context"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type IntegrationSecretStorage struct {
	tx          *gorm.DB
	encryptor   crypto.Encryptor
	integration *models.Integration
	secrets     []models.IntegrationSecret
}

func NewIntegrationSecretStorage(tx *gorm.DB, encryptor crypto.Encryptor, integration *models.Integration) (*IntegrationSecretStorage, error) {
	var secrets []models.IntegrationSecret
	err := tx.Where("installation_id = ?", integration.ID).Find(&secrets).Error
	if err != nil {
		return nil, err
	}

	return &IntegrationSecretStorage{
		tx:          tx,
		encryptor:   encryptor,
		integration: integration,
		secrets:     secrets,
	}, nil
}

func (s *IntegrationSecretStorage) Get(name string) (string, error) {
	for _, secret := range s.secrets {
		if secret.Name == name {
			decryptedValue, err := s.encryptor.Decrypt(
				context.Background(),
				secret.Value,
				[]byte(s.integration.ID.String()),
			)

			if err != nil {
				return "", err
			}

			return string(decryptedValue), nil
		}
	}

	return "", fmt.Errorf("secret %s not found", name)
}

func (s *IntegrationSecretStorage) Delete(name string) error {
	err := s.tx.
		Where("installation_id = ? AND name = ?", s.integration.ID, name).
		Delete(&models.IntegrationSecret{}).
		Error

	if err != nil {
		return err
	}

	for i, secret := range s.secrets {
		if secret.Name == name {
			s.secrets = append(s.secrets[:i], s.secrets[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("secret %s not found", name)
}

func (s *IntegrationSecretStorage) Create(name string, def core.IntegrationSecretDefinition) error {
	_, err := s.Get(name)
	if err == nil {
		return fmt.Errorf("secret %s already exists", name)
	}

	encryptedValue, err := s.encryptor.Encrypt(
		context.Background(),
		def.Value,
		[]byte(s.integration.ID.String()),
	)
	if err != nil {
		return err
	}

	now := time.Now()
	secret := models.IntegrationSecret{
		OrganizationID: s.integration.OrganizationID,
		InstallationID: s.integration.ID,
		Name:           name,
		Value:          encryptedValue,
		CreatedAt:      &now,
		UpdatedAt:      &now,
		Editable:       def.Editable,
	}

	err = s.tx.Create(&secret).Error
	if err != nil {
		return err
	}

	s.secrets = append(s.secrets, secret)
	return nil
}
