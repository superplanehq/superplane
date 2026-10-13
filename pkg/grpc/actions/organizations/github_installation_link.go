package organizations

import (
	"errors"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

func appendHostedGitHubInstallationURL(instance *models.Integration, proto *pb.Integration) {
	if !hostedGitHubIntegration(instance) {
		return
	}
	for _, property := range proto.Status.Properties {
		if property.Name == common.PropertyAppInstallationURL && strings.TrimSpace(property.Value) != "" {
			return
		}
	}

	installationURL, err := common.HostedInstallationSettingsURL(database.Conn(), instance.ID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("failed to load GitHub App installation URL for integration %s: %v", instance.ID, err)
		}
		return
	}
	if installationURL == "" {
		return
	}

	proto.Status.Properties = append(proto.Status.Properties, &pb.Integration_Property{
		Name:        common.PropertyAppInstallationURL,
		Label:       "GitHub App Installation URL",
		Description: "Open this GitHub App installation in GitHub.",
		Type:        string(core.IntegrationPropertyTypeString),
		Value:       installationURL,
		Editable:    false,
	})
}

func hostedGitHubIntegration(instance *models.Integration) bool {
	if instance == nil || instance.AppName != models.ProviderGitHub {
		return false
	}
	metadata := instance.Metadata.Data()
	hosted, _ := metadata["hostedApp"].(bool)
	return hosted
}
