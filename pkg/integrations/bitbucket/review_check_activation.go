package bitbucket

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func reviewActivation(node *core.PublicationNode) (int64, bool, error) {
	if node == nil || node.Component != "bitbucket.setPullRequestReviewCheck" {
		return 0, false, nil
	}
	var config struct {
		Enabled    *bool `mapstructure:"enabled"`
		Generation int64 `mapstructure:"activationGeneration"`
	}
	if err := mapstructure.Decode(node.Configuration, &config); err != nil {
		return 0, false, err
	}
	if config.Generation == 0 {
		config.Generation = 1
	}
	if config.Generation < 1 {
		return 0, false, fmt.Errorf("activation generation must be positive")
	}
	return config.Generation, config.Enabled == nil || *config.Enabled, nil
}

func reviewCheckActivation(store core.PublicationStore, canvas uuid.UUID, node string, attempt uuid.UUID) (int64, bool, error) {
	admitted, err := store.NodeSnapshot(canvas, node, attempt)
	if err != nil {
		return 0, false, err
	}
	before, enabled, err := reviewActivation(admitted)
	if err != nil || !enabled {
		return before, false, err
	}
	current, err := store.NodeSnapshot(canvas, node, uuid.Nil)
	if err != nil {
		return 0, false, err
	}
	now, enabled, err := reviewActivation(current)
	return before, enabled && before == now, err
}

func reviewCheckStillActive(store core.PublicationStore, record *core.OrderedPublication) (bool, error) {
	current, err := store.NodeSnapshot(record.OwnerCanvasID, record.OwnerNodeID, uuid.Nil)
	if err != nil {
		return false, err
	}
	generation, enabled, err := reviewActivation(current)
	return enabled && generation == record.ActivationGeneration && record.OwnerDeletedAt == nil, err
}

func disableReviewCheck(record *core.OrderedPublication) error {
	if record.Disabled {
		return core.ErrPublicationUnchanged
	}
	var state reviewCheckState
	if err := json.Unmarshal(record.Payload, &state); err != nil {
		return err
	}
	record.Disabled = true
	record.Generation++
	state.Configuration.Phase = "disabled"
	state.Delivered, state.Error, state.Failures = "", "", 0
	record.WakeAt = reviewWake(time.Second)
	var err error
	record.Payload, err = json.Marshal(state)
	return err
}

func bitbucketReviewStore(integration core.IntegrationContext) (core.PublicationStore, error) {
	owner, ok := integration.(core.PublicationIntegration)
	if !ok {
		return nil, fmt.Errorf("integration does not support durable publications")
	}
	return owner.Publications(), nil
}
