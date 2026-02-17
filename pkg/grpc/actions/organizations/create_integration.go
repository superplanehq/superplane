package organizations

import (
	"context"
	"encoding/base64"
	"fmt"
	"maps"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/oidc"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func CreateIntegration(ctx context.Context, registry *registry.Registry, oidcProvider oidc.Provider, baseURL string, webhooksBaseURL string, orgID string, integrationName, name string, appConfig *structpb.Struct) (*pb.CreateIntegrationResponse, error) {
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

	//
	// We must encrypt the sensitive configuration fields before storing
	//
	installationID := uuid.New()
	integrationLogger := logging.ForIntegration(models.Integration{
		ID:      installationID,
		AppName: integrationName,
	})
	configuration, err := encryptConfigurationIfNeeded(ctx, registry, integration, appConfig.AsMap(), installationID, nil)
	if err != nil {
		integrationLogger.WithError(err).Error("failed to encrypt sensitive configuration")
		return nil, status.Error(codes.Internal, "failed to encrypt sensitive configuration")
	}

	newIntegration, err := models.CreateIntegration(installationID, org, integrationName, name, configuration)
	if err != nil {
		integrationLogger.WithError(err).Error("failed to create integration")
		return nil, status.Error(codes.Internal, "failed to create integration")
	}

	integrationCtx := contexts.NewIntegrationContext(
		database.Conn(),
		nil,
		newIntegration,
		registry.Encryptor,
		registry,
	)

	syncErr := integration.Sync(core.SyncContext{
		Logger:          logging.ForIntegration(*newIntegration),
		HTTP:            registry.HTTPContext(),
		Integration:     integrationCtx,
		Configuration:   newIntegration.Configuration.Data(),
		BaseURL:         baseURL,
		WebhooksBaseURL: webhooksBaseURL,
		OrganizationID:  orgID,
		OIDC:            oidcProvider,
	})

	err = database.Conn().Save(newIntegration).Error
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

	metadata, err := structpb.NewStruct(instance.Metadata.Data())
	if err != nil {
		return nil, err
	}

	proto := &pb.Integration{
		Metadata: &pb.Integration_Metadata{
			Id:   instance.ID.String(),
			Name: instance.InstallationName,
		},
		Spec: &pb.Integration_Spec{
			IntegrationName: instance.AppName,
			Configuration:   config,
		},
		Status: &pb.Integration_Status{
			State:            instance.State,
			StateDescription: instance.StateDescription,
			Metadata:         metadata,
			UsedIn:           []*pb.Integration_NodeRef{},
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

	return proto, nil
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
