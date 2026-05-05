package organizations

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func NextIntegrationSetupStep(
	ctx context.Context,
	registry *registry.Registry,
	baseURL, webhooksBaseURL,
	orgID, id string,
	inputs *structpb.Struct,
	capabilities []string,
) (*pb.NextIntegrationSetupStepResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	integrationID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid integration ID")
	}

	integration, err := models.FindIntegration(org, integrationID)
	if err != nil {
		logrus.WithError(err).Error("failed to find integration")

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "integration not found")
		}

		return nil, status.Error(codes.Internal, "failed to find integration")
	}

	if integration.SetupState == nil {
		return nil, status.Error(codes.InvalidArgument, "current step is not set, cannot submit")
	}

	setupState := integration.SetupState.Data()
	if setupState.CurrentStep == nil {
		return nil, status.Error(codes.InvalidArgument, "current step is not set, cannot submit")
	}

	//
	// If we submitting a "done" step, we just clear the setup state and return.
	//
	if setupState.CurrentStep.Type == core.SetupStepTypeDone {
		return clearIntegrationSetupState(registry, integration)
	}

	return submitStep(registry, integration, baseURL, webhooksBaseURL, &setupState, inputs, capabilities)
}

func getStepInputs(inputs *structpb.Struct) map[string]any {
	if inputs == nil {
		return map[string]any{}
	}

	return inputs.AsMap()
}

// postSetupSyncDescription is shown while the integration request worker runs the first Sync()
// (required to persist OAuth tokens, webhooks, metadata, etc.).
const postSetupSyncDescription = "Running initial sync..."

const postSetupSyncDescriptionGCPWIFDelayed = "Running initial sync shortly (brief wait for IAM propagation)..."

// GCP workload identity: IAM bindings can lag creation; defer first sync so workers do not hit STS before propagation.
const (
	gcpIntegrationAppName               = "gcp"
	gcpPropertyConnectionMethod         = "connectionMethod"
	gcpConnectionMethodWorkloadIdentity = "workloadIdentityFederation"
	gcpWIFInitialSyncDelay              = 90 * time.Second
)

func gcpWorkloadIdentityFederationDeferredSync(integration *models.Integration) bool {
	if integration.AppName != gcpIntegrationAppName {
		return false
	}
	for _, p := range integration.Properties {
		if p.Name != gcpPropertyConnectionMethod {
			continue
		}
		v, ok := p.Value.(string)
		if !ok {
			v = fmt.Sprint(p.Value)
		}
		return strings.TrimSpace(v) == gcpConnectionMethodWorkloadIdentity
	}
	return false
}

func initialSyncRunAt(integration *models.Integration) time.Time {
	t := time.Now()
	if gcpWorkloadIdentityFederationDeferredSync(integration) {
		return t.Add(gcpWIFInitialSyncDelay)
	}
	return t
}

func initialSyncStateDescription(integration *models.Integration) string {
	if gcpWorkloadIdentityFederationDeferredSync(integration) {
		return postSetupSyncDescriptionGCPWIFDelayed
	}
	return postSetupSyncDescription
}

func clearIntegrationSetupState(registry *registry.Registry, integration *models.Integration) (*pb.NextIntegrationSetupStepResponse, error) {
	integration.SetupState = nil
	integration.State = models.IntegrationStatePending
	integration.StateDescription = initialSyncStateDescription(integration)
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(integration).Error; err != nil {
			return err
		}
		runAt := initialSyncRunAt(integration)
		return integration.CreateSyncRequest(tx, &runAt)
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to clear integration setup state")
	}

	proto, err := serializeIntegration(registry, integration, []models.CanvasNodeReference{})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize integration")
	}

	return &pb.NextIntegrationSetupStepResponse{
		Integration: proto,
	}, nil
}

func submitStep(
	registry *registry.Registry,
	integration *models.Integration,
	baseURL,
	webhooksBaseURL string,
	setupState *models.SetupState,
	inputs *structpb.Struct,
	capabilities []string,
) (*pb.NextIntegrationSetupStepResponse, error) {
	setupProvider, err := registry.GetSetupProvider(integration.AppName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get setup provider")
	}

	allCapabilities := allCapabilities(setupProvider)

	//
	// Verify that the requested capabilities are valid.
	//
	if len(capabilities) > 0 {
		for _, capability := range capabilities {
			if !slices.ContainsFunc(allCapabilities, func(c core.Capability) bool { return c.Name == capability }) {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid capability: %s", capability))
			}
		}
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		capabilityCtx := contexts.NewCapabilityContext(allCapabilities, integration.Capabilities)
		nextStep, err := setupProvider.OnStepSubmit(core.SetupStepContext{
			Logger:          logging.ForIntegration(*integration),
			Step:            core.StepInfo{Name: setupState.CurrentStep.Name, Inputs: getStepInputs(inputs), Capabilities: capabilities},
			BaseURL:         baseURL,
			WebhooksBaseURL: webhooksBaseURL,
			IntegrationID:   integration.ID,
			OrganizationID:  integration.OrganizationID.String(),
			HTTP:            registry.HTTPContext(),
			Properties:      contexts.NewIntegrationPropertyStorage(integration),
			Secrets:         contexts.NewIntegrationSecretStorage(tx, registry.Encryptor, integration),
			Capabilities:    capabilityCtx,
		})

		if err != nil {
			return err
		}

		//
		// If no next step, clear the setup state and return.
		//
		if nextStep == nil {
			now := time.Now()
			integration.UpdatedAt = &now
			integration.Capabilities = capabilityCtx.States()
			integration.SetupState = nil
			integration.State = models.IntegrationStatePending
			integration.StateDescription = initialSyncStateDescription(integration)
			if err := tx.Save(integration).Error; err != nil {
				return err
			}
			runAt := initialSyncRunAt(integration)
			return integration.CreateSyncRequest(tx, &runAt)
		}

		//
		// Calculate the next setup state
		//
		newState := models.SetupState{
			CurrentStep:   nextStep,
			PreviousSteps: []core.SetupStep{},
		}

		if len(setupState.PreviousSteps) > 0 {
			newState.PreviousSteps = append(newState.PreviousSteps, setupState.PreviousSteps...)
		}

		newState.PreviousSteps = append(newState.PreviousSteps, *setupState.CurrentStep)
		nextSetupState := datatypes.NewJSONType(newState)
		integration.SetupState = &nextSetupState
		integration.Capabilities = capabilityCtx.States()
		return tx.Save(integration).Error
	})

	if err != nil {
		logrus.WithError(err).Error("failed to submit integration setup step")
		return nil, status.Error(codes.Internal, "failed to submit integration setup step")
	}

	proto, err := serializeIntegration(registry, integration, []models.CanvasNodeReference{})
	if err != nil {
		logrus.WithError(err).Error("failed to serialize integration")
		return nil, err
	}

	return &pb.NextIntegrationSetupStepResponse{
		Integration: proto,
	}, nil
}
