package contexts

import (
	"encoding/json"

	"github.com/superplanehq/superplane/pkg/applications"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

type AppContext struct {
	appInstallation *models.AppInstallation
}

func NewAppContext(appInstallation *models.AppInstallation) applications.AppContext {
	return &AppContext{appInstallation: appInstallation}
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

func (m *AppContext) NewBrowserAction(action applications.BrowserAction) {
	d := datatypes.NewJSONType(action)
	m.appInstallation.BrowserAction = &d
}

func (m *AppContext) RemoveBrowserAction() {
	m.appInstallation.BrowserAction = nil
}
