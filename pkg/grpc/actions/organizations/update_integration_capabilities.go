package organizations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func UpdateIntegrationCapabilities(ctx context.Context, registry *registry.Registry, orgID string, integrationID string, capabilities []*pb.Integration_CapabilityState) (*pb.UpdateIntegrationCapabilitiesResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	id, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid integration ID")
	}

	integration, err := models.FindIntegration(org, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "integration not found")
		}

		log.WithError(err).Error("failed to find integration")
		return nil, status.Error(codes.Internal, "failed to find integration")
	}

	changes := findChanges(integration.Capabilities, capabilities)
	if len(changes) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no changes")
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {

		//
		// Handle the capability updates being requested.
		// Database updates or integration implementation calls are needed.
		//
		err := handleCapabilityUpdates(tx, registry, integration, changes)
		if err != nil {
			return err
		}

		//
		// All the downstream changes are done, so we can merge
		// the changes requested into the current states and update the integration.
		//
		now := time.Now()
		integration.UpdatedAt = &now
		return tx.Save(integration).Error
	})

	if err != nil {
		log.WithError(err).Error("failed to update integration capabilities")
		return nil, status.Error(codes.Internal, "failed to update integration capabilities")
	}

	proto, err := serializeIntegration(registry, integration, []models.CanvasNodeReference{})
	if err != nil {
		return nil, err
	}

	return &pb.UpdateIntegrationCapabilitiesResponse{
		Integration: proto,
	}, nil
}

func findChanges(current []models.CapabilityState, updatedCapabilities []*pb.Integration_CapabilityState) map[core.IntegrationCapabilityState][]string {
	changes := map[core.IntegrationCapabilityState][]string{}

	currentMap := capabilityMap(current)
	for _, c := range updatedCapabilities {
		current, ok := currentMap[c.Name]
		if !ok {
			// TODO: if we add new capabilities to an integration,
			// we have to deal with this?
			continue
		}

		newState := ProtoToCapabilityState(c.State)
		if current.State != newState {
			if _, ok := changes[newState]; !ok {
				changes[newState] = []string{c.Name}
			} else {
				changes[newState] = append(changes[newState], c.Name)
			}
		}
	}

	return changes
}

func capabilityMap(in []models.CapabilityState) map[string]models.CapabilityState {
	m := map[string]models.CapabilityState{}
	for _, capability := range in {
		m[capability.Name] = capability
	}
	return m
}

func handleCapabilityUpdates(tx *gorm.DB, registry *registry.Registry, integration *models.Integration, changes map[core.IntegrationCapabilityState][]string) error {
	enabled := changes[core.IntegrationCapabilityStateEnabled]
	err := handleEnablingCapabilties(tx, integration, enabled)
	if err != nil {
		return err
	}

	disabled := changes[core.IntegrationCapabilityStateDisabled]
	err = handleDisablingCapabilties(tx, integration, disabled)
	if err != nil {
		return err
	}

	requested := changes[core.IntegrationCapabilityStateRequested]
	err = handleRequestingCapabilties(tx, registry, integration, requested)
	if err != nil {
		return err
	}

	return nil
}

func handleEnablingCapabilties(tx *gorm.DB, integration *models.Integration, capabilities []string) error {
	//
	// TODO: move nodes back to ready state? What if the node has another non-capability related error?
	//
	if len(capabilities) == 0 {
		return nil
	}

	//
	// Update the current capabilities states.
	//
	updatedStates := map[string]models.CapabilityState{}
	for _, capability := range integration.Capabilities {
		updatedStates[capability.Name] = capability
	}

	for _, capability := range capabilities {
		updatedStates[capability] = models.CapabilityState{
			Name:  capability,
			State: core.IntegrationCapabilityStateEnabled,
		}
	}

	states := make([]models.CapabilityState, 0, len(updatedStates))
	for _, state := range updatedStates {
		states = append(states, state)
	}

	integration.Capabilities = datatypes.JSONSlice[models.CapabilityState](states)
	return nil
}

func handleDisablingCapabilties(tx *gorm.DB, integration *models.Integration, capabilities []string) error {
	//
	// TODO: move nodes to error state - only ready nodes, since nodes in error state are already non-functional
	//
	if len(capabilities) == 0 {
		return nil
	}

	//
	// Update the current capabilities states.
	//
	updatedStates := map[string]models.CapabilityState{}
	for _, capability := range integration.Capabilities {
		updatedStates[capability.Name] = capability
	}

	for _, capability := range capabilities {
		updatedStates[capability] = models.CapabilityState{
			Name:  capability,
			State: core.IntegrationCapabilityStateDisabled,
		}
	}

	states := make([]models.CapabilityState, 0, len(updatedStates))
	for _, state := range updatedStates {
		states = append(states, state)
	}

	integration.Capabilities = datatypes.JSONSlice[models.CapabilityState](states)
	return nil
}

/*
 * When new capabilities are requested, we need to give the integration implementation a chance to react.
 */
func handleRequestingCapabilties(tx *gorm.DB, registry *registry.Registry, integration *models.Integration, capabilities []string) error {
	if len(capabilities) == 0 {
		return nil
	}

	setupProvider, err := registry.GetSetupProvider(integration.AppName)
	if err != nil {
		return fmt.Errorf("failed to get setup provider: %v", err)
	}

	secretStorage, err := contexts.NewIntegrationSecretStorage(tx, registry.Encryptor, integration)
	if err != nil {
		return err
	}

	capabilityCtx := contexts.NewCapabilityContext(allCapabilities(setupProvider), integration.Capabilities)
	nextStep, err := setupProvider.OnCapabilityUpdate(core.CapabilityUpdateContext{
		Logger:       logging.ForIntegration(*integration),
		Changes:      map[core.IntegrationCapabilityState][]string{core.IntegrationCapabilityStateRequested: capabilities},
		HTTP:         registry.HTTPContext(),
		Properties:   contexts.NewIntegrationPropertyStorage(integration),
		Secrets:      secretStorage,
		Capabilities: capabilityCtx,
	})

	if err != nil {
		return err
	}

	//
	// If OnCapabilityUpdate returns no next step,
	// we don't need to re-enter a setup flow.
	// We still update the capabilities here,
	// because the integration implementation might have enabled / disabled them.
	//
	if nextStep == nil {
		newCapabilities := capabilityCtx.States()
		integration.Capabilities = newCapabilities
		return nil
	}

	//
	// It it does, we need to initiate a new setup flow.
	//
	newState := models.SetupState{
		CurrentStep:   nextStep,
		PreviousSteps: []core.SetupStep{},
	}

	nextSetupState := datatypes.NewJSONType(newState)
	integration.SetupState = &nextSetupState
	integration.Capabilities = capabilityCtx.States()
	return nil
}
