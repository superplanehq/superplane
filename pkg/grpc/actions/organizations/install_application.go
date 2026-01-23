package organizations

import (
	"context"
	"encoding/base64"
	"fmt"
	"maps"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/oidc"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func InstallApplication(ctx context.Context, registry *registry.Registry, oidcProvider oidc.Provider, baseURL string, webhooksBaseURL string, orgID string, appName, installationName string, appConfig *structpb.Struct) (*pb.InstallApplicationResponse, error) {
	app, err := registry.GetApplication(appName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "application %s not found", appName)
	}

	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization")
	}

	//
	// Check if an installation with this name already exists in the organization
	//
	_, err = models.FindAppInstallationByName(org, installationName)
	if err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "an installation with the name %s already exists in this organization", installationName)
	}

	//
	// We must encrypt the sensitive configuration fields before storing
	//
	installationID := uuid.New()
	configuration, err := encryptConfigurationIfNeeded(ctx, registry, app, appConfig.AsMap(), installationID, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to encrypt sensitive configuration: %v", err)
	}

	appInstallation, err := models.CreateAppInstallation(installationID, uuid.MustParse(orgID), appName, installationName, configuration)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create application installation: %v", err)
	}

	appCtx := contexts.NewAppInstallationContext(
		database.Conn(),
		nil,
		appInstallation,
		registry.Encryptor,
		registry,
	)

	syncErr := app.Sync(core.SyncContext{
		HTTP:            contexts.NewHTTPContext(registry.GetHTTPClient()),
		AppInstallation: appCtx,
		Configuration:   appInstallation.Configuration.Data(),
		BaseURL:         baseURL,
		WebhooksBaseURL: webhooksBaseURL,
		OrganizationID:  orgID,
		InstallationID:  appInstallation.ID.String(),
		OIDC:            oidcProvider,
	})

	err = database.Conn().Save(appInstallation).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save application installation after sync: %v", err)
	}

	if syncErr != nil {
		appInstallation.State = "error"
		appInstallation.StateDescription = syncErr.Error()
		err = database.Conn().Save(appInstallation).Error
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to save application installation after sync: %v", err)
		}
	}

	proto, err := serializeAppInstallation(registry, appInstallation, []models.WorkflowNodeReference{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to serialize application installation: %v", err)
	}

	return &pb.InstallApplicationResponse{
		Installation: proto,
	}, nil
}

func serializeAppInstallation(registry *registry.Registry, appInstallation *models.AppInstallation, nodeRefs []models.WorkflowNodeReference) (*pb.AppInstallation, error) {
	app, err := registry.GetApplication(appInstallation.AppName)
	if err != nil {
		return nil, err
	}

	//
	// We do not return sensitive values when serializing app installations.
	//
	config, err := structpb.NewStruct(sanitizeConfigurationIfNeeded(app, appInstallation.Configuration.Data()))
	if err != nil {
		return nil, err
	}

	metadata, err := structpb.NewStruct(appInstallation.Metadata.Data())
	if err != nil {
		return nil, err
	}

	proto := &pb.AppInstallation{
		Metadata: &pb.AppInstallation_Metadata{
			Id:   appInstallation.ID.String(),
			Name: appInstallation.InstallationName,
		},
		Spec: &pb.AppInstallation_Spec{
			AppName:       appInstallation.AppName,
			Configuration: config,
		},
		Status: &pb.AppInstallation_Status{
			State:            appInstallation.State,
			StateDescription: appInstallation.StateDescription,
			Metadata:         metadata,
			UsedIn:           []*pb.AppInstallation_NodeRef{},
		},
	}

	if appInstallation.BrowserAction != nil {
		browserAction := appInstallation.BrowserAction.Data()
		proto.Status.BrowserAction = &pb.BrowserAction{
			Description: browserAction.Description,
			Url:         browserAction.URL,
			Method:      browserAction.Method,
			FormFields:  browserAction.FormFields,
		}
	}

	for _, nodeRef := range nodeRefs {
		proto.Status.UsedIn = append(proto.Status.UsedIn, &pb.AppInstallation_NodeRef{
			WorkflowId:   nodeRef.WorkflowID.String(),
			WorkflowName: nodeRef.WorkflowName,
			NodeId:       nodeRef.NodeID,
			NodeName:     nodeRef.NodeName,
		})
	}

	return proto, nil
}

func encryptConfigurationIfNeeded(ctx context.Context, registry *registry.Registry, app core.Application, config map[string]any, installationID uuid.UUID, existingConfig map[string]any) (map[string]any, error) {
	result := maps.Clone(config)

	for _, field := range app.Configuration() {
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

func sanitizeConfigurationIfNeeded(app core.Application, config map[string]any) map[string]any {
	sanitized := maps.Clone(config)

	for _, field := range app.Configuration() {
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
