package registry

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/extensions"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type ExtensionComponent struct {
	tx       *gorm.DB
	manifest extensions.ComponentManifest
}

func NewExtensionComponent(tx *gorm.DB, manifest extensions.ComponentManifest) *ExtensionComponent {
	return &ExtensionComponent{
		tx:       tx,
		manifest: manifest,
	}
}

func (s *ExtensionComponent) Name() string {
	return s.manifest.Name
}

func (s *ExtensionComponent) Label() string {
	return s.manifest.Label
}

func (s *ExtensionComponent) Description() string {
	return s.manifest.Description
}

func (s *ExtensionComponent) Documentation() string {
	return ""
}

func (s *ExtensionComponent) Icon() string {
	return s.manifest.Icon
}

func (s *ExtensionComponent) Color() string {
	return s.manifest.Color
}

func (s *ExtensionComponent) ExampleOutput() map[string]any {
	return nil
}

func (s *ExtensionComponent) Configuration() []configuration.Field {
	return s.manifest.Configuration
}

func (s *ExtensionComponent) Actions() []core.Action {
	return s.manifest.Actions
}

func (s *ExtensionComponent) OutputChannels(config any) []core.OutputChannel {
	return s.manifest.OutputChannels
}

func (s *ExtensionComponent) Setup(ctx core.SetupContext) error {
	return nil
}

func (s *ExtensionComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (s *ExtensionComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (s *ExtensionComponent) Execute(ctx core.ExecutionContext) error {
	//
	// TODO: not sure about pkg/registry importing pkg/models for this.
	// TODO 2: we are using the latest version of the extension,
	// but ideally we'd use the version specified in the "installation metadata".
	//
	extension, err := models.FindExtensionInTransaction(s.tx, uuid.MustParse(ctx.OrganizationID), s.manifest.ExtensionID)
	if err != nil {
		return fmt.Errorf("error finding extension: %w", err)
	}

	latestVersion, err := extension.FindLatestVersion(s.tx)
	if err != nil {
		return fmt.Errorf("error finding latest version: %w", err)
	}

	_, err = models.CreateInvokeExtensionJob(
		s.tx,
		uuid.MustParse(ctx.OrganizationID),
		latestVersion,
		ctx.ID,
		&extensions.InvocationTarget{
			BlockType: extensions.InvocationBlockTypeComponent,
			BlockName: s.Name(),
			Operation: extensions.InvocationOperationExecute,
		},
	)

	return err
}

//
// TODO: implement the rest of the methods
//

func (s *ExtensionComponent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("not implemented")
}

func (s *ExtensionComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusInternalServerError, nil, fmt.Errorf("not implemented")
}

func (s *ExtensionComponent) Cancel(ctx core.ExecutionContext) error {
	return fmt.Errorf("not implemented")
}

func (s *ExtensionComponent) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	return fmt.Errorf("not implemented")
}
