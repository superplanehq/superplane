package organizations

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/applications"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func InstallApplication(ctx context.Context, registry *registry.Registry, baseURL string, orgID string, appName, installationName string, appConfig *structpb.Struct) (*pb.InstallApplicationResponse, error) {
	app, err := registry.GetApplication(appName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "application %s not found", appName)
	}

	// Encrypt sensitive configuration fields before storing
	configMap := appConfig.AsMap()
	encryptedConfig, err := encryptSensitiveFields(ctx, registry, app, configMap, orgID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to encrypt sensitive configuration: %v", err)
	}

	appInstallation, err := models.CreateAppInstallation(uuid.MustParse(orgID), appName, installationName, encryptedConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create application installation: %v", err)
	}

	syncErr := app.Sync(applications.SyncContext{
		Configuration:  appInstallation.Configuration.Data(),
		BaseURL:        baseURL,
		OrganizationID: orgID,
		InstallationID: appInstallation.ID.String(),
		AppContext:     contexts.NewAppContext(database.Conn(), appInstallation, registry.Encryptor),
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

	proto, err := serializeAppInstallation(registry, appInstallation)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to serialize application installation: %v", err)
	}

	return &pb.InstallApplicationResponse{
		Installation: proto,
	}, nil
}

func serializeAppInstallation(registry *registry.Registry, appInstallation *models.AppInstallation) (*pb.AppInstallation, error) {
	// Get the app definition to check for sensitive fields
	app, err := registry.GetApplication(appInstallation.AppName)
	if err != nil {
		return nil, err
	}

	// Sanitize sensitive fields by replacing them with SHA256 hashes
	sanitizedConfig := sanitizeSensitiveFields(app, appInstallation.Configuration.Data())

	config, err := structpb.NewStruct(sanitizedConfig)
	if err != nil {
		return nil, err
	}

	metadata, err := structpb.NewStruct(appInstallation.Metadata.Data())
	if err != nil {
		return nil, err
	}

	proto := &pb.AppInstallation{
		Id:               appInstallation.ID.String(),
		AppName:          appInstallation.AppName,
		InstallationName: appInstallation.InstallationName,
		State:            appInstallation.State,
		StateDescription: appInstallation.StateDescription,
		Configuration:    config,
		Metadata:         metadata,
	}

	if appInstallation.BrowserAction != nil {
		browserAction := appInstallation.BrowserAction.Data()
		proto.BrowserAction = &pb.BrowserAction{
			Description: browserAction.Description,
			Url:         browserAction.URL,
			Method:      browserAction.Method,
			FormFields:  browserAction.FormFields,
		}
	}

	return proto, nil
}

// encryptSensitiveFields encrypts all sensitive configuration fields
func encryptSensitiveFields(ctx context.Context, registry *registry.Registry, app applications.Application, config map[string]any, orgID string) (map[string]any, error) {
	result := make(map[string]any)
	for k, v := range config {
		result[k] = v
	}

	configFields := app.Configuration()
	for _, field := range configFields {
		if field.Sensitive {
			value, exists := config[field.Name]
			if !exists {
				continue
			}

			// Convert value to string
			strValue, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("sensitive field %s must be a string", field.Name)
			}

			// Encrypt the value using orgID as associated data
			encrypted, err := registry.Encryptor.Encrypt(ctx, []byte(strValue), []byte(orgID))
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt field %s: %v", field.Name, err)
			}

			// Encode encrypted bytes as base64 string so it can be decoded by mapstructure
			result[field.Name] = base64.StdEncoding.EncodeToString(encrypted)
		}
	}

	return result, nil
}

// sanitizeSensitiveFields replaces encrypted sensitive configuration fields with their SHA256 hash
func sanitizeSensitiveFields(app applications.Application, config map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range config {
		result[k] = v
	}

	configFields := app.Configuration()
	for _, field := range configFields {
		if field.Sensitive {
			value, exists := config[field.Name]
			if !exists {
				continue
			}

			// Convert base64-encoded encrypted value to string
			var encryptedStr string
			switch v := value.(type) {
			case string:
				encryptedStr = v
			default:
				// If it's not a string, skip
				continue
			}

			// Compute SHA256 of the base64-encoded encrypted value
			hash := sha256.Sum256([]byte(encryptedStr))
			result[field.Name] = hex.EncodeToString(hash[:])
		}
	}

	return result
}
