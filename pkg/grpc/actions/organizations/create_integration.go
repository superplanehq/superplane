package organizations

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/oidc"
	configpb "github.com/superplanehq/superplane/pkg/protos/configuration"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func CreateIntegration(ctx context.Context, registry *registry.Registry, oidcProvider oidc.Provider, baseURL string, webhooksBaseURL string, orgID string, integrationName, name string, appConfig *structpb.Struct) (*pb.CreateIntegrationResponse, error) {
	return CreateIntegrationWithUsage(ctx, nil, registry, oidcProvider, baseURL, webhooksBaseURL, orgID, integrationName, name, appConfig, nil)
}

func CreateIntegrationWithUsage(
	ctx context.Context,
	usageService usage.Service,
	registry *registry.Registry,
	oidcProvider oidc.Provider,
	baseURL string,
	webhooksBaseURL string,
	orgID string,
	integrationName, name string,
	appConfig *structpb.Struct,
	capabilities []string,
) (*pb.CreateIntegrationResponse, error) {
	integration, err := registry.GetIntegration(integrationName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "integration %s not found", integrationName)
	}

	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization")
	}

	//
	// Check if an integration with this name already exists in the organization
	//
	_, err = models.FindIntegrationByName(org, name)
	if err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "an integration with the name %s already exists in this organization", name)
	}

	integrationCount, err := models.CountIntegrationsByOrganization(orgID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count integrations: %v", err)
	}

	if err := usage.EnsureOrganizationWithinLimits(ctx, usageService, orgID, &usagepb.OrganizationState{
		Integrations: int32(integrationCount + 1),
	}, nil); err != nil {
		return nil, err
	}

	//
	// We must encrypt the sensitive configuration fields before storing
	//
	integrationID := uuid.New()
	integrationLogger := logging.ForIntegration(models.Integration{
		ID:      integrationID,
		AppName: integrationName,
	})

	//
	// If the integration implementation supports the new flow,
	// and the user has requested to use it, we use the new flow.
	//
	if registry.SupportsNewSetupFlow(integrationName) {
		newIntegration, err := models.CreateIntegration(integrationID, org, integrationName, name, nil)
		if err != nil {
			integrationLogger.WithError(err).Error("failed to create integration")
			return nil, status.Error(codes.Internal, "failed to create integration")
		}

		setupProvider, err := registry.GetSetupProvider(integrationName)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get setup provider: %v", err)
		}

		return setupIntegration(registry, setupProvider, newIntegration, capabilities)
	}

	//
	// Otherwise, use the old flow.
	//
	configuration, err := encryptConfigurationIfNeeded(ctx, registry, integration, appConfig.AsMap(), integrationID, nil)
	if err != nil {
		integrationLogger.WithError(err).Error("failed to encrypt sensitive configuration")
		return nil, status.Error(codes.Internal, "failed to encrypt sensitive configuration")
	}

	newIntegration, err := models.CreateIntegration(integrationID, org, integrationName, name, configuration)
	if err != nil {
		integrationLogger.WithError(err).Error("failed to create integration")
		return nil, status.Error(codes.Internal, "failed to create integration")
	}

	return syncIntegration(registry, baseURL, webhooksBaseURL, oidcProvider, orgID, newIntegration, integration)
}

func allCapabilities(setupProvider core.IntegrationSetupProvider) []core.Capability {
	capabilities := []core.Capability{}
	for _, group := range setupProvider.CapabilityGroups() {
		capabilities = append(capabilities, group.Capabilities...)
	}
	return capabilities
}

func setupIntegration(registry *registry.Registry, setupProvider core.IntegrationSetupProvider, newIntegration *models.Integration, capabilities []string) (*pb.CreateIntegrationResponse, error) {
	logrus.Infof("setting up integration %s", newIntegration.ID)

	initialCapabilities, err := initialCapabilityStates(allCapabilities(setupProvider), capabilities)
	if err != nil {
		return nil, err
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		secretStorage, err := contexts.NewIntegrationSecretStorage(tx, registry.Encryptor, newIntegration)
		if err != nil {
			return err
		}

		newIntegration.Capabilities = initialCapabilities
		capabilityCtx := contexts.NewCapabilityContext(allCapabilities(setupProvider), newIntegration.Capabilities)
		firstStep := setupProvider.FirstStep(core.SetupStepContext{
			IntegrationID:  newIntegration.ID,
			OrganizationID: newIntegration.OrganizationID.String(),
			HTTP:           registry.HTTPContext(),
			Properties:     contexts.NewIntegrationPropertyStorage(newIntegration),
			Capabilities:   capabilityCtx,
			Secrets:        secretStorage,
		})

		setupState := datatypes.NewJSONType(models.SetupState{
			CurrentStep:   &firstStep,
			PreviousSteps: []core.SetupStep{},
		})

		newIntegration.SetupState = &setupState
		newIntegration.Capabilities = capabilityCtx.States()
		return tx.Save(newIntegration).Error
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to setup integration next setup step: %v", err)
	}

	proto, err := serializeIntegration(registry, newIntegration, []models.CanvasNodeReference{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to serialize integration: %v", err)
	}

	return &pb.CreateIntegrationResponse{Integration: proto}, nil
}

func initialCapabilityStates(definitions []core.Capability, requestedCapabilities []string) ([]models.CapabilityState, error) {
	definitionsByName := map[string]core.Capability{}
	for _, definition := range definitions {
		definitionsByName[definition.Name] = definition
	}

	requested := map[string]bool{}
	for _, capability := range requestedCapabilities {
		if _, ok := definitionsByName[capability]; !ok {
			return nil, status.Errorf(codes.InvalidArgument, "capability %s not found", capability)
		}
		requested[capability] = true
	}

	states := make([]models.CapabilityState, 0, len(definitions))
	for _, definition := range definitions {
		state := core.IntegrationCapabilityStateUnavailable
		if requested[definition.Name] {
			state = core.IntegrationCapabilityStateRequested
		}

		states = append(states, models.CapabilityState{
			Name:  definition.Name,
			State: state,
		})
	}

	return states, nil
}

func syncIntegration(
	registry *registry.Registry,
	baseURL string,
	webhooksBaseURL string,
	oidcProvider oidc.Provider,
	orgID string,
	newIntegration *models.Integration,
	integrationImpl core.Integration,
) (*pb.CreateIntegrationResponse, error) {
	logrus.Infof("syncing integration %s", newIntegration.ID)

	integrationCtx := contexts.NewIntegrationContext(
		database.Conn(),
		nil,
		newIntegration,
		registry.Encryptor,
		registry,
		nil,
	)

	syncErr := integrationImpl.Sync(core.SyncContext{
		Logger:          logging.ForIntegration(*newIntegration),
		HTTP:            registry.HTTPContext(),
		Integration:     integrationCtx,
		Configuration:   newIntegration.Configuration.Data(),
		BaseURL:         baseURL,
		WebhooksBaseURL: webhooksBaseURL,
		OrganizationID:  orgID,
		OIDC:            oidcProvider,
	})

	err := database.Conn().Save(newIntegration).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save integration after sync: %v", err)
	}

	if syncErr != nil {
		newIntegration.State = "error"
		newIntegration.StateDescription = syncErr.Error()
		err = database.Conn().Save(newIntegration).Error
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to save integration after sync: %v", err)
		}
	}

	proto, err := serializeIntegration(registry, newIntegration, []models.CanvasNodeReference{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to serialize integration: %v", err)
	}

	return &pb.CreateIntegrationResponse{
		Integration: proto,
	}, nil
}

func serializeIntegration(registry *registry.Registry, instance *models.Integration, nodeRefs []models.CanvasNodeReference) (*pb.Integration, error) {
	integration, err := registry.GetIntegration(instance.AppName)
	if err != nil {
		return nil, err
	}

	//
	// We do not return sensitive values when serializing integrations.
	//
	config, err := structpb.NewStruct(sanitizeConfigurationIfNeeded(integration, instance.Configuration.Data()))
	if err != nil {
		return nil, err
	}

	metadataMap := instance.Metadata.Data()
	if metadataMap == nil {
		metadataMap = map[string]any{}
	}

	metadata, err := structpb.NewStruct(metadataMap)
	if err != nil {
		return nil, err
	}

	proto := &pb.Integration{
		Metadata: &pb.Integration_Metadata{
			Id:              instance.ID.String(),
			Name:            instance.InstallationName,
			IntegrationName: instance.AppName,
		},
		Spec: &pb.Integration_Spec{
			Configuration: config,
		},
		Status: &pb.Integration_Status{
			State:            instance.State,
			StateDescription: instance.StateDescription,
			Metadata:         metadata,
			UsedIn:           []*pb.Integration_NodeRef{},
			Capabilities:     serializeCapabilities(registry, instance),
			LegacySetup:      isLegacySetup(instance),
		},
	}

	if instance.BrowserAction != nil {
		browserAction := instance.BrowserAction.Data()
		proto.Status.BrowserAction = &pb.BrowserAction{
			Description: browserAction.Description,
			Url:         browserAction.URL,
			Method:      browserAction.Method,
			FormFields:  browserAction.FormFields,
		}
	}

	for _, nodeRef := range nodeRefs {
		proto.Status.UsedIn = append(proto.Status.UsedIn, &pb.Integration_NodeRef{
			CanvasId:   nodeRef.CanvasID.String(),
			CanvasName: nodeRef.CanvasName,
			NodeId:     nodeRef.NodeID,
			NodeName:   nodeRef.NodeName,
		})
	}

	if instance.SetupState != nil {
		state := instance.SetupState.Data()
		proto.Status.SetupState = &pb.Integration_SetupState{
			PreviousSteps: []*pb.Integration_SetupStepDefinition{},
		}

		if state.CurrentStep != nil {
			proto.Status.SetupState.CurrentStep = serializeNextStep(*state.CurrentStep)
		}

		for _, step := range state.PreviousSteps {
			proto.Status.SetupState.PreviousSteps = append(proto.Status.SetupState.PreviousSteps, serializeNextStep(step))
		}
	}

	for _, property := range instance.Properties {
		proto.Status.Properties = append(proto.Status.Properties, &pb.Integration_Property{
			Name:        property.Name,
			Label:       property.Label,
			Description: property.Description,
			Type:        string(property.Type),
			Value:       integrationParameterValueToString(property.Value),
			Editable:    property.Editable,
		})
	}

	secrets, err := models.ListIntegrationSecrets(instance.ID)
	if err != nil {
		return nil, err
	}
	for _, secret := range secrets {
		proto.Status.Secrets = append(proto.Status.Secrets, &pb.Integration_Secret{
			Name:        secret.Name,
			Label:       secret.Label,
			Description: secret.Description,
			Editable:    secret.Editable,
		})
	}

	return proto, nil
}

func integrationParameterValueToString(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(encoded)
}

func serializeNextStep(step core.SetupStep) *pb.Integration_SetupStepDefinition {
	def := &pb.Integration_SetupStepDefinition{
		Type:         serializeNextStepType(step.Type),
		Name:         step.Name,
		Label:        step.Label,
		Instructions: step.Instructions,
		Inputs:       []*configpb.Field{},
	}

	for _, input := range step.Inputs {
		def.Inputs = append(def.Inputs, actions.ConfigurationFieldToProto(input))
	}

	if step.RedirectPrompt != nil {
		def.RedirectPrompt = &pb.Integration_SetupStepDefinition_RedirectPrompt{
			Url:        step.RedirectPrompt.URL,
			Method:     step.RedirectPrompt.Method,
			FormFields: step.RedirectPrompt.FormData,
		}
	}

	return def
}

func serializeNextStepType(stepType core.SetupStepType) pb.Integration_SetupStepDefinition_Type {
	switch stepType {
	case core.SetupStepTypeInputs:
		return pb.Integration_SetupStepDefinition_INPUTS
	case core.SetupStepTypeRedirectPrompt:
		return pb.Integration_SetupStepDefinition_REDIRECT_PROMPT
	case core.SetupStepTypeDone:
		return pb.Integration_SetupStepDefinition_DONE
	default:
		return pb.Integration_SetupStepDefinition_UNKNOWN
	}
}

func serializeCapabilities(registry *registry.Registry, integration *models.Integration) []*pb.Integration_CapabilityState {
	setupProvider, err := registry.GetSetupProvider(integration.AppName)

	//
	// If this is a legacy integration, all components are enabled.
	//
	if err != nil {
		impl, err := registry.GetIntegration(integration.AppName)
		if err != nil {
			return []*pb.Integration_CapabilityState{}
		}

		return serializeLegacyCapabilities(impl)
	}

	//
	// Otherwise, we use the capability states in the database.
	//
	capabilities := []core.Capability{}
	for _, group := range setupProvider.CapabilityGroups() {
		capabilities = append(capabilities, group.Capabilities...)
	}
	protos := []*pb.Integration_CapabilityState{}
	for _, capability := range integration.Capabilities {
		protos = append(protos, &pb.Integration_CapabilityState{
			Name:  capability.Name,
			State: CapabilityStateToProto(capability.State),
		})
	}

	return protos
}

func serializeLegacyCapabilities(integration core.Integration) []*pb.Integration_CapabilityState {
	protos := []*pb.Integration_CapabilityState{}

	for _, action := range integration.Actions() {
		protos = append(protos, &pb.Integration_CapabilityState{
			Name:  action.Name(),
			State: pb.Integration_CapabilityState_STATE_ENABLED,
		})
	}

	for _, trigger := range integration.Triggers() {
		protos = append(protos, &pb.Integration_CapabilityState{
			Name:  trigger.Name(),
			State: pb.Integration_CapabilityState_STATE_ENABLED,
		})
	}

	return protos
}

func ProtoToCapabilityState(s pb.Integration_CapabilityState_State) core.IntegrationCapabilityState {
	switch s {
	case pb.Integration_CapabilityState_STATE_ENABLED:
		return core.IntegrationCapabilityStateEnabled
	case pb.Integration_CapabilityState_STATE_DISABLED:
		return core.IntegrationCapabilityStateDisabled
	case pb.Integration_CapabilityState_STATE_REQUESTED:
		return core.IntegrationCapabilityStateRequested
	}
	return core.IntegrationCapabilityStateUnavailable
}

func CapabilityStateToProto(t core.IntegrationCapabilityState) pb.Integration_CapabilityState_State {
	switch t {
	case core.IntegrationCapabilityStateEnabled:
		return pb.Integration_CapabilityState_STATE_ENABLED
	case core.IntegrationCapabilityStateDisabled:
		return pb.Integration_CapabilityState_STATE_DISABLED
	case core.IntegrationCapabilityStateRequested:
		return pb.Integration_CapabilityState_STATE_REQUESTED
	case core.IntegrationCapabilityStateUnavailable:
		return pb.Integration_CapabilityState_STATE_UNAVAILABLE
	}
	return pb.Integration_CapabilityState_STATE_UNAVAILABLE
}

func encryptConfigurationIfNeeded(ctx context.Context, registry *registry.Registry, integration core.Integration, config map[string]any, installationID uuid.UUID, existingConfig map[string]any) (map[string]any, error) {
	result := maps.Clone(config)

	for _, field := range integration.Configuration() {
		if !field.Sensitive {
			continue
		}

		value, exists := config[field.Name]
		if !exists {
			continue
		}

		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("sensitive field %s must be a string", field.Name)
		}

		//
		// If the value is not <redacted>, encrypt it, since it's new.
		//
		if s != "<redacted>" {
			encrypted, err := registry.Encryptor.Encrypt(ctx, []byte(s), []byte(installationID.String()))
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt field %s: %v", field.Name, err)
			}

			result[field.Name] = base64.StdEncoding.EncodeToString(encrypted)
			continue
		}

		//
		// If <redacted> is used, just preserve the existing value.
		//
		if existingConfig == nil {
			continue
		}

		v, exists := existingConfig[field.Name]
		if exists {
			result[field.Name] = v
		}
	}

	return result, nil
}

func sanitizeConfigurationIfNeeded(integration core.Integration, config map[string]any) map[string]any {
	sanitized := maps.Clone(config)

	for _, field := range integration.Configuration() {
		if !field.Sensitive {
			continue
		}

		_, exists := config[field.Name]
		if !exists {
			continue
		}

		sanitized[field.Name] = "<redacted>"
	}

	return sanitized
}

func isLegacySetup(integration *models.Integration) bool {
	return len(integration.Capabilities) == 0
}
