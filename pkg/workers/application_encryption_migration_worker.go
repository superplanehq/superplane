package workers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type ApplicationEncryptionMigrationWorker struct {
	rabbitMQURL string
	encryptor   crypto.Encryptor
	registry    *registry.Registry
	locations   []models.EncryptedStorageLocation
	drainLock   sync.Mutex
}

const encryptionMigrationRecoveryInterval = 5 * time.Minute

func NewApplicationEncryptionMigrationWorker(rabbitMQURL string, encryptor crypto.Encryptor, registry *registry.Registry, locations []models.EncryptedStorageLocation) *ApplicationEncryptionMigrationWorker {
	return &ApplicationEncryptionMigrationWorker{rabbitMQURL: rabbitMQURL, encryptor: encryptor, registry: registry, locations: locations}
}

// Start wakes on migration requests. The recovery scan finds jobs left pending
// when a message is lost or a worker exits before finishing.
func (w *ApplicationEncryptionMigrationWorker) Start(ctx context.Context) {
	go w.consumeMigrationRequests(ctx)
	w.recoverMigrations(ctx)

	ticker := time.NewTicker(encryptionMigrationRecoveryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.recoverMigrations(ctx)
		}
	}
}

func (w *ApplicationEncryptionMigrationWorker) recoverMigrations(ctx context.Context) {
	if err := w.drainMigrations(ctx); err != nil && ctx.Err() == nil {
		log.WithError(err).Error("application encryption migration recovery failed")
	}
}

func (w *ApplicationEncryptionMigrationWorker) consumeMigrationRequests(ctx context.Context) {
	routingKey := messages.ApplicationEncryptionMigrationRequestedRoutingKey
	consumer := tackle.NewConsumer()
	consumer.SetLogger(logging.NewTackleLogger(log.WithField("worker", "ApplicationEncryptionMigrationWorker")))
	options := tackle.Options{
		URL:            w.rabbitMQURL,
		ConnectionName: "ApplicationEncryptionMigrationWorker",
		RemoteExchange: messages.CanvasExchange,
		Service:        messages.CanvasExchange + "." + routingKey + ".ApplicationEncryptionMigrationWorker",
		RoutingKey:     routingKey,
	}

	for ctx.Err() == nil {
		err := consumer.Start(&options, func(delivery tackle.Delivery) error {
			if err := w.consumeMigrationRequested(ctx, delivery); err != nil && ctx.Err() == nil {
				log.WithError(err).Error("application encryption migration request failed; recovery scan will retry")
			}
			// The job is durable in PostgreSQL. Acknowledge the wake-up and let
			// the recovery scan retry database failures without a message loop.
			return nil
		})
		if ctx.Err() != nil {
			return
		}
		log.WithError(err).Warn("application encryption migration consumer disconnected; reconnecting")
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (w *ApplicationEncryptionMigrationWorker) consumeMigrationRequested(ctx context.Context, _ tackle.Delivery) error {
	return w.drainMigrations(ctx)
}

func (w *ApplicationEncryptionMigrationWorker) drainMigrations(ctx context.Context) error {
	w.drainLock.Lock()
	defer w.drainLock.Unlock()

	for ctx.Err() == nil {
		processed, err := w.RunBatch(ctx)
		if err != nil {
			return err
		}
		if !processed {
			return nil
		}
	}
	return ctx.Err()
}

// RunBatch claims one short batch. PostgreSQL releases the job row lock after
// each batch or when a pod exits.
func (w *ApplicationEncryptionMigrationWorker) RunBatch(ctx context.Context) (bool, error) {
	var job *models.ApplicationEncryptionMigration
	var stateBefore, locationBefore string
	var issues []error
	var processingError error
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		job, err = models.LockNextApplicationEncryptionMigration(tx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			job = nil
			return nil
		}
		if err != nil {
			return err
		}
		stateBefore, locationBefore = job.State, job.StorageLocation
		processingError = tx.Transaction(func(batch *gorm.DB) error {
			if job.State == "auditing" {
				issues, err = encryption.NewAuditor(job, w.registry, w.locations).ProcessBatch(batch)
			} else {
				issues, err = encryption.NewMigrator(job, encryption.ForDB(batch, w.encryptor), w.registry, w.locations).ProcessBatch(ctx, batch)
			}
			return err
		})
		if processingError != nil {
			return job.Fail(tx, processingError)
		}
		return job.SaveProgress(tx)
	})
	if err != nil {
		if job != nil {
			return false, fmt.Errorf("migration %s: %w", job.ID, err)
		}
		return false, err
	}
	if job != nil {
		logEncryptionMigrationBatch(job, stateBefore, locationBefore, issues, processingError)
	}
	return job != nil, nil
}

func logEncryptionMigrationBatch(job *models.ApplicationEncryptionMigration, stateBefore, locationBefore string, issues []error, processingError error) {
	phase := "migration"
	if stateBefore == "auditing" {
		phase = "audit"
	}
	entry := log.WithFields(log.Fields{
		"migration_id":     job.ID,
		"target_key_id":    job.TargetKeyID,
		"storage_location": locationBefore,
	})
	if processingError != nil {
		entry.WithError(processingError).Error("application encryption " + phase + " failed")
		return
	}
	for _, issue := range issues {
		if stateBefore == "auditing" {
			entry.WithError(issue).Error("application encryption audit found a non-target value")
		} else {
			entry.WithError(issue).Error("application encryption value could not be migrated")
		}
	}
	if job.Cursor == "" {
		completed := entry
		if job.StorageLocation != locationBefore {
			completed = completed.WithField("next_storage_location", job.StorageLocation)
		}
		completed.Info("application encryption " + phase + " location completed")
	}
	if job.State == "failed" {
		entry.WithField("error_message", job.ErrorMessage).Error("application encryption " + phase + " failed")
		return
	}
	if job.State == "auditing" && stateBefore != "auditing" {
		entry.WithField("storage_location", job.StorageLocation).Info("application encryption audit started")
		return
	}
	if job.State == "completed" {
		entry.WithField("audit_result", "passed").Info("application encryption migration completed")
		return
	}
	if job.Cursor != "" {
		entry.WithField("cursor", job.Cursor).Info("application encryption " + phase + " batch committed")
	}
}
