package contexts

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/superplanehq/superplane/pkg/applications"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AppContext struct {
	tx              *gorm.DB
	appInstallation *models.AppInstallation
}

func NewAppContext(tx *gorm.DB, appInstallation *models.AppInstallation) applications.AppContext {
	return &AppContext{
		tx:              tx,
		appInstallation: appInstallation,
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

	var secret models.AppInstallationSecret
	err := m.tx.
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
			Value:          value,
			CreatedAt:      &now,
			UpdatedAt:      &now,
		}

		return m.tx.Create(&secret).Error
	}

	secret.Value = value
	secret.UpdatedAt = &now

	return m.tx.Save(&secret).Error
}

func (m *AppContext) GetSecrets() ([]applications.InstallationSecret, error) {
	var fromDB []models.AppInstallationSecret
	err := m.tx.
		Where("app_installation_id = ?", m.appInstallation.ID).
		Find(&fromDB).
		Error

	if err != nil {
		return nil, err
	}

	var secrets []applications.InstallationSecret
	for _, secret := range fromDB {
		secrets = append(secrets, applications.InstallationSecret{
			Name:  secret.Name,
			Value: secret.Value,
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
		URL:        action.URL,
		Method:     action.Method,
		FormFields: action.FormFields,
	})

	m.appInstallation.BrowserAction = &d
}

func (m *AppContext) RemoveBrowserAction() {
	m.appInstallation.BrowserAction = nil
}
