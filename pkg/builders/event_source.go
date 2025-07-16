package builders

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type EventSourceBuilder struct {
	tx          *gorm.DB
	ctx         context.Context
	encryptor   crypto.Encryptor
	canvas      *models.Canvas
	name        string
	scope       string
	integration *models.Integration
	resource    integrations.Resource
}

func NewEventSourceBuilder(encryptor crypto.Encryptor) *EventSourceBuilder {
	return &EventSourceBuilder{
		ctx:       context.Background(),
		encryptor: encryptor,
	}
}

func (b *EventSourceBuilder) WithTransaction(tx *gorm.DB) *EventSourceBuilder {
	b.tx = tx
	return b
}

func (b *EventSourceBuilder) WithContext(ctx context.Context) *EventSourceBuilder {
	b.ctx = ctx
	return b
}

func (b *EventSourceBuilder) InCanvas(canvas *models.Canvas) *EventSourceBuilder {
	b.canvas = canvas
	return b
}

func (b *EventSourceBuilder) WithName(name string) *EventSourceBuilder {
	b.name = name
	return b
}

func (b *EventSourceBuilder) WithScope(scope string) *EventSourceBuilder {
	b.scope = scope
	return b
}

func (b *EventSourceBuilder) ForIntegration(integration *models.Integration) *EventSourceBuilder {
	b.integration = integration
	return b
}

func (b *EventSourceBuilder) ForResource(resource integrations.Resource) *EventSourceBuilder {
	b.resource = resource
	return b
}

func (b *EventSourceBuilder) Create() (*models.EventSource, string, error) {
	if b.tx != nil {
		return b.create(b.tx)
	}

	var plainKey string
	var eventSource *models.EventSource
	var err error
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		eventSource, plainKey, err = b.create(tx)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, "", err
	}

	return eventSource, plainKey, err
}

func (b *EventSourceBuilder) create(tx *gorm.DB) (*models.EventSource, string, error) {
	if b.integration == nil && b.resource == nil {
		return b.createWithoutIntegration(tx)
	}

	return b.createForIntegration(tx)
}

func (b *EventSourceBuilder) createWithoutIntegration(tx *gorm.DB) (*models.EventSource, string, error) {
	plainKey, encryptedKey, err := b.generateKey()
	if err != nil {
		return nil, "", err
	}

	eventSource, err := b.canvas.CreateEventSourceInTransaction(tx, b.name, encryptedKey, b.scope, nil)
	if err != nil {
		return nil, "", err
	}

	return eventSource, plainKey, nil
}

func (b *EventSourceBuilder) createForIntegration(tx *gorm.DB) (*models.EventSource, string, error) {
	//
	// Ensure resource record exists.
	//
	resource, err := b.findOrCreateResource(tx)
	if err != nil {
		return nil, "", err
	}

	//
	// Check if event source exists.
	// If it does, we might either update it or fail the creation.
	//
	existingSource, err := models.FindEventSourceForResourceInTransaction(tx, resource.ID)
	if err == nil {
		return b.createForExistingSource(tx, existingSource)
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", err
	}

	//
	// If event source does not exist, create it.
	//
	plainKey, encryptedKey, err := b.generateKey()
	if err != nil {
		return nil, "", err
	}

	eventSource, err := b.canvas.CreateEventSourceInTransaction(tx, b.name, encryptedKey, b.scope, &resource.ID)
	if err != nil {
		return nil, "", err
	}

	return eventSource, plainKey, nil
}

func (b *EventSourceBuilder) findOrCreateResource(tx *gorm.DB) (*models.Resource, error) {
	resource, err := models.FindResourceInTransaction(tx, b.integration.ID, b.resource.Type(), b.resource.Name())
	if err == nil {
		return resource, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return b.integration.CreateResourceInTransaction(tx, b.resource.Type(), b.resource.Id(), b.resource.Name())
}

func (b *EventSourceBuilder) createForExistingSource(tx *gorm.DB, eventSource *models.EventSource) (*models.EventSource, string, error) {
	//
	// If the creation is for an internal event source,
	// and there's already an existing one, no need to do anything.
	//
	if b.scope == models.EventSourceScopeInternal {
		return eventSource, "", nil
	}

	//
	// If the creation is for an external event source,
	// and there's already an existing external one, fail the creation, to avoid a duplicate.
	//
	if eventSource.Scope == models.EventSourceScopeExternal {
		return nil, "", fmt.Errorf("event source for %s %s already exists", b.resource.Type(), b.resource.Name())
	}

	//
	// If the creation is for an external event source,
	// and there's already an existing internal one, update its name and make it external.
	//
	now := time.Now()
	eventSource.Name = b.name
	eventSource.Scope = b.scope
	eventSource.UpdatedAt = &now
	err := tx.Save(eventSource).Error
	if err != nil {
		return nil, "", err
	}

	plainKey, err := eventSource.GetDecryptedKeyInTransaction(tx, b.ctx, b.encryptor)
	if err != nil {
		return nil, "", err
	}

	return eventSource, string(plainKey), nil
}

func (b *EventSourceBuilder) generateKey() (string, []byte, error) {
	//
	// If the event source is not tied to an integration resource,
	// we can use the event source name to authenticate the key encryption.
	//
	if b.resource == nil {
		return crypto.NewRandomKey(b.ctx, b.encryptor, b.name)
	}

	//
	// Otherwise, we use the resource ID to authenticate the key encryption,
	// to avoid having to update the webhook secret in the integration,
	// when the event source has its name updated when going from internal->external scope.
	//
	return crypto.NewRandomKey(b.ctx, b.encryptor, b.resource.Id())
}
