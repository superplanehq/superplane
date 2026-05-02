package contexts

import (
	"context"
	"fmt"
	"slices"
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
	loaded      bool
}

func NewIntegrationSecretStorage(tx *gorm.DB, encryptor crypto.Encryptor, integration *models.Integration) *IntegrationSecretStorage {
	return &IntegrationSecretStorage{
		tx:          tx,
		encryptor:   encryptor,
		integration: integration,
		secrets:     []models.IntegrationSecret{},
	}
}

func (s *IntegrationSecretStorage) loadSecrets() error {
	if s.loaded {
		return nil
	}

	var secrets []models.IntegrationSecret
	err := s.tx.Where("installation_id = ?", s.integration.ID).Find(&secrets).Error
	if err != nil {
		return err
	}

	s.secrets = secrets
	s.loaded = true
	return nil
}

func (s *IntegrationSecretStorage) findSecret(name string) (*models.IntegrationSecret, error) {
	for i := range s.secrets {
		if s.secrets[i].Name == name {
			return &s.secrets[i], nil
		}
	}

	return nil, fmt.Errorf("secret %s not found", name)
}

func (s *IntegrationSecretStorage) Get(name string) (string, error) {
	err := s.loadSecrets()
	if err != nil {
		return "", err
	}

	secret, err := s.findSecret(name)
	if err != nil {
		return "", err
	}

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

func (s *IntegrationSecretStorage) Delete(name string) error {
	err := s.loadSecrets()
	if err != nil {
		return err
	}

	_, err = s.findSecret(name)
	if err != nil {
		return err
	}

	err = s.tx.
		Where("installation_id = ? AND name = ?", s.integration.ID, name).
		Delete(&models.IntegrationSecret{}).
		Error

	if err != nil {
		return err
	}

	s.secrets = slices.DeleteFunc(s.secrets, func(secret models.IntegrationSecret) bool {
		return secret.Name == name
	})

	return nil
}

func (s *IntegrationSecretStorage) Create(def core.IntegrationSecretDefinition) error {
	if def.Name == "" {
		return fmt.Errorf("secret name is required")
	}

	err := s.loadSecrets()
	if err != nil {
		return err
	}

	_, err = s.Get(def.Name)
	if err == nil {
		return fmt.Errorf("secret %s already exists", def.Name)
	}

	encryptedValue, err := s.encryptor.Encrypt(
		context.Background(),
		[]byte(def.Value),
		[]byte(s.integration.ID.String()),
	)
	if err != nil {
		return err
	}

	now := time.Now()
	secret := models.IntegrationSecret{
		OrganizationID: s.integration.OrganizationID,
		InstallationID: s.integration.ID,
		Name:           def.Name,
		Label:          def.Label,
		Description:    def.Description,
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

func (s *IntegrationSecretStorage) CreateMany(defs []core.IntegrationSecretDefinition) error {
	for _, def := range defs {
		err := s.Create(def)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *IntegrationSecretStorage) Update(name string, value string) error {
	err := s.loadSecrets()
	if err != nil {
		return err
	}

	secret, err := s.findSecret(name)
	if err != nil {
		return err
	}

	encryptedValue, err := s.encryptor.Encrypt(
		context.Background(),
		[]byte(value),
		[]byte(s.integration.ID.String()),
	)
	if err != nil {
		return err
	}

	now := time.Now()
	secret.Value = encryptedValue
	secret.UpdatedAt = &now
	return s.tx.Save(secret).Error
}
