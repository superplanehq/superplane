package workers

import (
	"bytes"
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	logrus "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const seedFilesCommitMessage = "Seed repository with files from app source"

const (
	canvasRepositoryProvisionerServiceName   = "superplane." + messages.CanvasExchange + "." + messages.CanvasCreatedRoutingKey + ".canvas-repository-provisioner"
	canvasRepositoryProvisionerConnection    = "superplane"
	canvasRepositoryProvisionerBatch         = 100
	canvasRepositoryProvisionerBackfillEvery = time.Minute
)

type RepositoryProvisionerWorker struct {
	Consumer    *tackle.Consumer
	RabbitMQURL string
	Storage     git.Provider
	semaphore   *semaphore.Weighted
}

func NewRepositoryProvisionerWorker(rabbitMQURL string, storage git.Provider) *RepositoryProvisionerWorker {
	logger := logging.NewTackleLogger(logrus.StandardLogger().WithFields(logrus.Fields{
		"worker": "RepositoryProvisioner",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &RepositoryProvisionerWorker{
		Consumer:    consumer,
		RabbitMQURL: rabbitMQURL,
		Storage:     storage,
		semaphore:   semaphore.NewWeighted(25),
	}
}

func (w *RepositoryProvisionerWorker) Start(ctx context.Context) {
	go w.startBackfillLoop(ctx)
	go w.startConsumerLoop(ctx)

	<-ctx.Done()
	w.Stop()
}

func (w *RepositoryProvisionerWorker) Stop() {
	w.Consumer.Stop()
}

func (w *RepositoryProvisionerWorker) startConsumerLoop(ctx context.Context) {
	options := tackle.Options{
		URL:            w.RabbitMQURL,
		ConnectionName: canvasRepositoryProvisionerConnection,
		Service:        canvasRepositoryProvisionerServiceName,
		RemoteExchange: messages.CanvasExchange,
		RoutingKey:     messages.CanvasCreatedRoutingKey,
	}

	for {
		if ctx.Err() != nil {
			return
		}

		log.Println("Connecting to RabbitMQ queue for canvas-created canvas repository provisioning")

		err := w.Consumer.Start(&options, w.ConsumeCanvasCreated)
		if ctx.Err() != nil {
			return
		}

		if err != nil {
			w.log("Error consuming canvas-created messages: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		w.log("Connection to RabbitMQ closed for canvas-created, reconnecting...")
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (w *RepositoryProvisionerWorker) startBackfillLoop(ctx context.Context) {
	w.backfill(ctx)

	ticker := time.NewTicker(canvasRepositoryProvisionerBackfillEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.backfill(ctx)
		}
	}
}

func (w *RepositoryProvisionerWorker) backfill(ctx context.Context) {
	repositories, err := models.ListPendingRepositories(canvasRepositoryProvisionerBatch)
	if err != nil {
		w.log("Error listing pending canvas repositories: %v", err)
		return
	}

	for _, repository := range repositories {
		if err := w.semaphore.Acquire(ctx, 1); err != nil {
			return
		}

		go func(repository models.Repository) {
			defer w.semaphore.Release(1)

			if err := w.provisionRepository(ctx, repository); err != nil {
				w.log("Error provisioning repository for canvas %s: %v", repository.CanvasID, err)
			}
		}(repository)
	}
}

func (w *RepositoryProvisionerWorker) ConsumeCanvasCreated(delivery tackle.Delivery) error {
	message := &pb.CanvasMessage{}
	if err := proto.Unmarshal(delivery.Body(), message); err != nil {
		w.log("Error unmarshaling canvas created message: %v", err)
		return err
	}

	canvasID, err := uuid.Parse(message.GetCanvasId())
	if err != nil {
		return nil
	}

	repository, err := models.FindRepositoryUnscoped(canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}

		w.log("Error finding repository for canvas %s: %v", canvasID, err)
		return err
	}

	err = w.provisionRepository(context.Background(), *repository)
	if err != nil {
		w.log("Error provisioning canvas repository for canvas %s: %v", canvasID, err)
		return err
	}

	return nil
}

func (w *RepositoryProvisionerWorker) provisionRepository(ctx context.Context, repository models.Repository) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		repository, err := models.LockPendingRepository(tx, repository.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}

			return err
		}

		_, err = w.Storage.CreateRepository(ctx, repository.RepoID)
		if err != nil {
			w.log("Error creating repository for canvas %s: %v", repository.CanvasID, err)
			return repository.MarkError(tx)
		}

		if err := w.commitSeedFiles(ctx, tx, repository); err != nil {
			w.log("Error committing seed files for canvas %s: %v", repository.CanvasID, err)
			return repository.MarkError(tx)
		}

		w.log("Repository created for canvas %s", repository.CanvasID)
		return repository.MarkReady(tx)
	})
}

// commitSeedFiles applies any persisted seed files as the canvas repository's
// initial content (after the empty README.md created by CreateRepository) and
// deletes the seed rows once the commit succeeds. Repositories without seed
// files are a no-op.
func (w *RepositoryProvisionerWorker) commitSeedFiles(ctx context.Context, tx *gorm.DB, repository *models.Repository) error {
	seedFiles, err := models.ListRepositorySeedFilesInTransaction(tx, repository.ID)
	if err != nil {
		return err
	}

	if len(seedFiles) == 0 {
		return nil
	}

	operations := make([]git.FileOperation, 0, len(seedFiles))
	for _, file := range seedFiles {
		operations = append(operations, git.FileOperation{
			Path:      file.Path,
			Content:   bytes.NewReader(file.Content),
			SizeBytes: int64(len(file.Content)),
		})
	}

	if _, err := w.Storage.Commit(ctx, repository.RepoID, git.CommitOptions{
		Branch:     "main",
		BaseBranch: "main",
		Message:    seedFilesCommitMessage,
		Author:     git.SuperPlaneBotAuthor(),
		Operations: operations,
	}); err != nil {
		return err
	}

	return models.DeleteRepositorySeedFilesInTransaction(tx, repository.ID)
}

func (w *RepositoryProvisionerWorker) log(format string, v ...any) {
	log.Printf("[RepositoryProvisioner] "+format, v...)
}
