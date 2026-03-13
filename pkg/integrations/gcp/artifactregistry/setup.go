package artifactregistry

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

const ensureArtifactRegistrySetupDelay = 2 * time.Second

func scheduleArtifactRegistrySetupIfNeeded(integration core.IntegrationContext) error {
	if integration == nil {
		return nil
	}

	var metadata gcpcommon.Metadata
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	if err := integration.ScheduleActionCall(gcpcommon.ActionNameEnsureArtifactRegistry, nil, ensureArtifactRegistrySetupDelay); err != nil {
		return fmt.Errorf("schedule Artifact Registry setup: %w", err)
	}

	return nil
}
