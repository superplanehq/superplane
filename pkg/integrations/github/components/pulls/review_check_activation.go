package pulls

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// Participation comes from the admitted graph. Live settings can revoke it,
// but enabling cannot attach a check to an attempt that already started.
func reviewCheckActivation(store core.PublicationStore, canvas uuid.UUID, nodeID string, attempt uuid.UUID) (int64, bool, error) {
	admitted, err := store.NodeSnapshot(canvas, nodeID, attempt)
	if err != nil {
		return 0, false, err
	}
	current, err := store.NodeSnapshot(canvas, nodeID, uuid.Nil)
	if err != nil {
		return 0, false, err
	}
	if admitted == nil || current == nil {
		return 0, false, nil
	}
	if !supportsCheckActivation(admitted.Component) || current.Component != admitted.Component {
		return 0, false, nil
	}
	var before, now reviewCheckConfiguration
	if err := mapstructure.Decode(admitted.Configuration, &before); err != nil {
		return 0, false, err
	}
	if err := mapstructure.Decode(current.Configuration, &now); err != nil {
		return 0, false, err
	}
	eligible := checkActivationEnabled(before, admitted.Component) && checkActivationEnabled(now, current.Component) && before.ActivationGeneration == now.ActivationGeneration
	return before.ActivationGeneration, eligible, nil
}

// Admission already persisted this generation in the publication. Refresh only
// needs the live fence, so tracking survives retention of the selection run.
func reviewCheckStillActive(store core.PublicationStore, record *core.OrderedPublication) (bool, error) {
	current, err := store.NodeSnapshot(record.OwnerCanvasID, record.OwnerNodeID, uuid.Nil)
	if err != nil {
		return false, err
	}
	state, err := decodeReviewCheckState(record.Payload)
	if err != nil {
		return false, err
	}
	component := "github.setPullRequestReviewCheck"
	if state.Configuration.CheckRun {
		component = "github.setPullRequestCheckRun"
	}
	if current == nil || current.Component != component {
		return false, nil
	}
	var config reviewCheckConfiguration
	if err := mapstructure.Decode(current.Configuration, &config); err != nil {
		return false, err
	}
	return checkActivationEnabled(config, component) && config.ActivationGeneration == record.ActivationGeneration, nil
}

func checkActivationEnabled(config reviewCheckConfiguration, component string) bool {
	if config.Enabled != nil {
		return *config.Enabled
	}
	return component == "github.setPullRequestCheckRun"
}

func supportsCheckActivation(component string) bool {
	return component == "github.setPullRequestReviewCheck" || component == "github.setPullRequestCheckRun"
}

func configurationWithActivation(raw any, generation int64) (map[string]any, error) {
	var config map[string]any
	if err := mapstructure.Decode(raw, &config); err != nil {
		return nil, err
	}
	if generation < 1 {
		return nil, fmt.Errorf("activation generation must be positive")
	}
	config["activationGeneration"] = generation
	return config, nil
}

func releaseReviewChecks(store core.PublicationStore, canvas uuid.UUID, nodeID string) error {
	records, err := store.List(reviewCheckPrefix)
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.OwnerCanvasID != canvas || record.OwnerNodeID != nodeID || record.Disabled {
			continue
		}
		if err := store.Update(record.Key, func(current *core.OrderedPublication) error {
			if current.OwnerCanvasID != canvas || current.OwnerNodeID != nodeID {
				return core.ErrPublicationUnchanged
			}
			return disableReviewCheck(current)
		}); err != nil {
			return err
		}
	}
	return nil
}

func disableReviewCheck(record *core.OrderedPublication) error {
	if record.Disabled {
		return core.ErrPublicationUnchanged
	}
	state, err := decodeReviewCheckState(record.Payload)
	if err != nil {
		return err
	}
	state.Configuration.Phase = "disabled"
	if state.Configuration.CheckRun {
		// Retain receipts and in-flight delivery so a late creation is released.
		state.Configuration.State, state.Configuration.Reason = "success", "PR check publication is disabled."
		state.State, state.Description, state.Delivered, state.Error = "", "", "", ""
		state.SetupFailures = 0
	}
	record.Disabled = true
	record.Generation++
	record.WakeAt = reviewCheckWake(time.Second)
	if state.Configuration.CheckRun {
		record.Payload, err = json.Marshal(state)
		return err
	}
	record.Payload, err = json.Marshal(reviewCheckState{Configuration: state.Configuration, Delivery: state.Delivery})
	return err
}
