package organizations

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		return nil, err
	}

	newCapabilities, err := updateCapabilities(integration.Capabilities, capabilities)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	integration.UpdatedAt = &now
	integration.Capabilities = newCapabilities
	err = database.Conn().Save(integration).Error
	if err != nil {
		return nil, err
	}

	proto, err := serializeIntegration(registry, integration, []models.CanvasNodeReference{})
	if err != nil {
		return nil, err
	}

	return &pb.UpdateIntegrationCapabilitiesResponse{
		Integration: proto,
	}, nil
}

func updateCapabilities(current []models.CapabilityState, updatedCapabilities []*pb.Integration_CapabilityState) ([]models.CapabilityState, error) {
	m := map[string]models.CapabilityState{}
	for _, capability := range current {
		m[capability.Name] = capability
	}

	for _, updatedCapability := range updatedCapabilities {
		_, ok := m[updatedCapability.Name]
		if !ok {
			return nil, status.Errorf(codes.NotFound, "capability %s not found", updatedCapability.Name)
		}

		switch updatedCapability.State {
		case pb.Integration_CapabilityState_STATE_ENABLED:
			m[updatedCapability.Name] = models.CapabilityState{
				Name:  updatedCapability.Name,
				State: models.IntegrationCapabilityStateEnabled,
			}
		case pb.Integration_CapabilityState_STATE_DISABLED:
			m[updatedCapability.Name] = models.CapabilityState{
				Name:  updatedCapability.Name,
				State: models.IntegrationCapabilityStateDisabled,
			}
		default:
			// TODO: handle requesting new capabilities - UNAVAILABLE -> X
			// TODO: handle requesting less capabilities - X -> UNAVAILABLE
			return nil, status.Errorf(codes.InvalidArgument, "invalid capability state %s", updatedCapability.State)
		}
	}

	newCapabilities := make([]models.CapabilityState, 0, len(m))
	for _, capability := range m {
		newCapabilities = append(newCapabilities, capability)
	}

	return newCapabilities, nil
}
