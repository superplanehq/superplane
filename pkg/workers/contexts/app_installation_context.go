package contexts

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type AppInstallationContext struct {
	tx              *gorm.DB
	appInstallation *models.AppInstallation
	encryptor       crypto.Encryptor
	registry        *registry.Registry
}

func NewAppInstallationContext(tx *gorm.DB, appInstallation *models.AppInstallation, encryptor crypto.Encryptor, registry *registry.Registry) *AppInstallationContext {
	return &AppInstallationContext{
		tx:              tx,
		appInstallation: appInstallation,
		encryptor:       encryptor,
		registry:        registry,
	}
}

func (c *AppInstallationContext) GetConfig(name string) ([]byte, error) {
	config := c.appInstallation.Configuration.Data()
	v, ok := config[name]
	if !ok {
		return nil, fmt.Errorf("config %s not found", name)
	}

	app, err := c.registry.GetApplication(c.appInstallation.AppName)
	if err != nil {
		return nil, fmt.Errorf("failed to get app %s: %w", c.appInstallation.AppName, err)
	}

	configDef, err := findConfigDef(app.Configuration(), name)
	if err != nil {
		return nil, fmt.Errorf("failed to find config %s: %w", name, err)
	}

	if configDef.Type != configuration.FieldTypeString {
		return nil, fmt.Errorf("config %s is not a string", name)
	}

	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("config %s is not a string", name)
	}

	if !configDef.Sensitive {
		return []byte(s), nil
	}

	b64, err := c.encryptor.Decrypt(context.Background(), []byte(s), []byte(c.appInstallation.ID.String()))
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(string(b64))
}

func findConfigDef(configs []configuration.Field, name string) (configuration.Field, error) {
	for _, config := range configs {
		if config.Name == name {
			return config, nil
		}
	}

	return configuration.Field{}, fmt.Errorf("config %s not found", name)
}
