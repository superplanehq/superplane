package contexts

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type codexSubscriptionContext struct {
	tx             *gorm.DB
	encryptor      crypto.Encryptor
	organizationID uuid.UUID
	factoryID      uuid.UUID
}

func NewCodexSubscriptionContext(tx *gorm.DB, canvas *models.Canvas, encryptor crypto.Encryptor) core.CodexSubscriptionContext {
	if canvas == nil || canvas.FactoryID == nil {
		return nil
	}
	return &codexSubscriptionContext{
		tx:             tx,
		encryptor:      encryptor,
		organizationID: canvas.OrganizationID,
		factoryID:      *canvas.FactoryID,
	}
}

func (c *codexSubscriptionContext) Reserve(taskID uuid.UUID) (core.CodexSubscriptionLease, error) {
	connection, err := models.ReserveFactoryCodexConnection(
		c.tx,
		c.organizationID,
		c.factoryID,
		taskID,
		time.Now(),
		c.encryptor,
	)
	if err != nil {
		return core.CodexSubscriptionLease{}, err
	}
	return core.CodexSubscriptionLease{
		ConnectionID: connection.ID,
		FactoryID:    c.factoryID,
		Generation:   connection.ReservationGeneration,
	}, nil
}
