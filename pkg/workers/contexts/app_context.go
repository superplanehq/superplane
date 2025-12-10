package contexts

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/superplanehq/superplane/pkg/applications"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AppContext struct {
	tx              *gorm.DB
	appInstallation *models.AppInstallation
	encryptor       crypto.Encryptor
}

func NewAppContext(tx *gorm.DB, appInstallation *models.AppInstallation, encryptor crypto.Encryptor) applications.AppContext {
	return &AppContext{
		tx:              tx,
		appInstallation: appInstallation,
		encryptor:       encryptor,
	}
}

func (m *AppContext) GetMetadata() any {
	return m.appInstallation.Metadata.Data()
}

func (m *AppContext) SetMetadata(value any) {
	b, err := json.Marshal(value)
	if err != nil {
		return
	}

	var v map[string]any
	err = json.Unmarshal(b, &v)
	if err != nil {
		return
	}

	m.appInstallation.Metadata = datatypes.NewJSONType(v)
}

func (m *AppContext) GetState() string {
	return m.appInstallation.State
}

func (m *AppContext) SetState(value string) {
	m.appInstallation.State = value
}

func (m *AppContext) SetSecret(name string, value []byte) error {
	now := time.Now()

	// Encrypt the secret value using the installation ID as associated data
	encryptedValue, err := m.encryptor.Encrypt(
		context.Background(),
		value,
		[]byte(m.appInstallation.ID.String()),
	)
	if err != nil {
		return err
	}

	var secret models.AppInstallationSecret
	err = m.tx.
		Where("installation_id = ?", m.appInstallation.ID).
		Where("name = ?", name).
		First(&secret).
		Error

	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		secret = models.AppInstallationSecret{
			OrganizationID: m.appInstallation.OrganizationID,
			InstallationID: m.appInstallation.ID,
			Name:           name,
			Value:          encryptedValue,
			CreatedAt:      &now,
			UpdatedAt:      &now,
		}

		return m.tx.Create(&secret).Error
	}

	secret.Value = encryptedValue
	secret.UpdatedAt = &now

	return m.tx.Save(&secret).Error
}

func (m *AppContext) GetSecrets() ([]applications.InstallationSecret, error) {
	var fromDB []models.AppInstallationSecret
	err := m.tx.
		Where("installation_id = ?", m.appInstallation.ID).
		Find(&fromDB).
		Error

	if err != nil {
		return nil, err
	}

	var secrets []applications.InstallationSecret
	for _, secret := range fromDB {
		// Decrypt the secret value using the installation ID as associated data
		decryptedValue, err := m.encryptor.Decrypt(
			context.Background(),
			secret.Value,
			[]byte(m.appInstallation.ID.String()),
		)
		if err != nil {
			return nil, err
		}

		secrets = append(secrets, applications.InstallationSecret{
			Name:  secret.Name,
			Value: decryptedValue,
		})
	}

	return secrets, nil
}

func (m *AppContext) NewBrowserAction(action applications.BrowserAction) {
	//
	// TODO: we wouldn't need to this unnecessary conversion
	// if no circular dependency existed between pkg/components, pkg/models, and pkg/applications
	//
	d := datatypes.NewJSONType(models.BrowserAction{
		URL:         action.URL,
		Method:      action.Method,
		FormFields:  action.FormFields,
		Description: action.Description,
	})

	m.appInstallation.BrowserAction = &d
}

func (m *AppContext) RemoveBrowserAction() {
	m.appInstallation.BrowserAction = nil
}
