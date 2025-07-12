package integrations

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	ResourceTypeTask         = "task"
	ResourceTypeTaskTrigger  = "task-trigger"
	ResourceTypeProject      = "project"
	ResourceTypeWorkflow     = "workflow"
	ResourceTypeNotification = "notification"
	ResourceTypeSecret       = "secret"
	ResourceTypePipeline     = "pipeline"
	ResourceTypeRepository   = "repository"
)

type Integration interface {
	Get(resourceType, id string) (Resource, error)
	Create(resourceType string, params any) (Resource, error)
	List(resourceType string) ([]Resource, error)
}

type Resource interface {
	Id() string
	Name() string
	Type() string
}

func NewIntegration(ctx context.Context, integration *models.Integration, encryptor crypto.Encryptor) (Integration, error) {
	switch integration.Type {
	case models.IntegrationTypeSemaphore:
		//
		// TODO: if integration can be on organization level, and integration can use secret,
		// then secret should also be on organization level as well.
		//
		secretInfo := integration.Auth.Data().Token.ValueFrom.Secret
		secret, err := models.FindSecretByName(integration.DomainID.String(), secretInfo.Name)
		if err != nil {
			return nil, err
		}

		//
		// TODO: this handling should account for possibly other types of secrets
		//
		decrypted, err := encryptor.Decrypt(ctx, secret.Data, []byte(secretInfo.Name))
		if err != nil {
			return nil, err
		}

		var keys map[string]string
		err = json.Unmarshal(decrypted, &keys)
		if err != nil {
			return nil, err
		}

		token, ok := keys[secretInfo.Key]
		if !ok {
			return nil, fmt.Errorf("secret %s does not contain key %s", secretInfo.Name, secretInfo.Key)
		}

		return NewSemaphoreIntegration(integration.URL, token)
	default:
		return nil, fmt.Errorf("unsupported integration type %s", integration.Type)
	}
}
