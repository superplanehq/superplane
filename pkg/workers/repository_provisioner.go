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
	"github.com/superplanehq/superplane/pkg/telemetry"
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

	repositoryProvisionerTriggerBackfill = "backfill"
	repositoryProvisionerTriggerEvent    = "event"

	repositoryProvisionerReasonCreateRepository = "create_repository"
	repositoryProvisionerReasonCommitSeedFiles  = "commit_seed_files"
	repositoryProvisionerReasonMarkError        = "mark_error"
	repositoryProvisionerReasonMarkReady        = "mark_ready"
)

type RepositoryProvisionerWorker struct {
	Consumer    *tackle.Consumer
	RabbitMQURL string
	Storage     git.Provider
	semaphore   *semaphore.Weighted
	metrics     repositoryProvisionerMetrics
}

type repositoryProvisionerMetrics interface {
	recordTickDuration(context.Context, time.Duration)
	recordRepositoriesCount(context.Context, int)
	recordRepository(context.Context, repositoryProvisioningMetric)
}

type repositoryProvisioningMetric struct {
	duration time.Duration
	trigger  string
	outcome  string
	reason   string
}

type telemetryRepositoryProvisionerMetrics struct{}

func (telemetryRepositoryProvisionerMetrics) recordTickDuration(ctx context.Context, duration time.Duration) {
	telemetry.RecordRepositoryProvisionerWorkerTickDuration(ctx, duration)
}

func (telemetryRepositoryProvisionerMetrics) recordRepositoriesCount(ctx context.Context, count int) {
	telemetry.RecordRepositoryProvisionerWorkerRepositoriesCount(ctx, count)
}

func (telemetryRepositoryProvisionerMetrics) recordRepository(
	ctx context.Context,
	record repositoryProvisioningMetric,
) {
	telemetry.RecordRepositoryProvisionerWorkerRepositoryProcessing(
		ctx,
		record.duration,
		record.trigger,
		record.outcome,
		record.reason,
	)
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
		metrics:     telemetryRepositoryProvisionerMetrics{},
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
	startedAt := time.Now()
	defer func() {
		w.metrics.recordTickDuration(ctx, time.Since(startedAt))
	}()

	repositories, err := models.ListPendingRepositories(canvasRepositoryProvisionerBatch)
	if err != nil {
		w.log("Error listing pending canvas repositories: %v", err)
		return
	}
	w.metrics.recordRepositoriesCount(ctx, len(repositories))

	for _, repository := range repositories {
		if err := w.semaphore.Acquire(ctx, 1); err != nil {
			return
		}

		go func(repository models.Repository) {
			defer w.semaphore.Release(1)

			if err := w.provisionRepository(ctx, repository, repositoryProvisionerTriggerBackfill); err != nil {
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

	err = w.provisionRepository(
		context.Background(),
		*repository,
		repositoryProvisionerTriggerEvent,
	)
	if err != nil {
		w.log("Error provisioning canvas repository for canvas %s: %v", canvasID, err)
		return err
	}

	return nil
}

func (w *RepositoryProvisionerWorker) provisionRepository(
	ctx context.Context,
	repository models.Repository,
	trigger string,
) error {
	startedAt := time.Now()
	outcome := executorOutcomeSuccess
	reason := executorReasonNone
	defer func() {
		w.metrics.recordRepository(ctx, repositoryProvisioningMetric{
			duration: time.Since(startedAt),
			trigger:  trigger,
			outcome:  outcome,
			reason:   reason,
		})
	}()

	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		repository, err := models.LockPendingRepository(tx, repository.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				outcome = executorOutcomeSkipped
				reason = executorReasonNotFound
				return nil
			}

			outcome = executorOutcomeFailed
			reason = classifyProcessError(err)
			return err
		}

		_, err = w.Storage.CreateRepository(ctx, repository.RepoID)
		if err != nil {
			w.log("Error creating repository for canvas %s: %v", repository.CanvasID, err)
			outcome = executorOutcomeFailed
			reason = repositoryProvisionerReasonCreateRepository
			if err := repository.MarkError(tx); err != nil {
				reason = repositoryProvisionerReasonMarkError
				return err
			}
			return nil
		}

		if err := w.commitSeedFiles(ctx, tx, repository); err != nil {
			w.log("Error committing seed files for canvas %s: %v", repository.CanvasID, err)
			outcome = executorOutcomeFailed
			reason = repositoryProvisionerReasonCommitSeedFiles
			if err := repository.MarkError(tx); err != nil {
				reason = repositoryProvisionerReasonMarkError
				return err
			}
			return nil
		}

		w.log("Repository created for canvas %s", repository.CanvasID)
		if err := repository.MarkReady(tx); err != nil {
			outcome = executorOutcomeFailed
			reason = repositoryProvisionerReasonMarkReady
			return err
		}

		return nil
	})
	if err != nil && outcome == executorOutcomeSuccess {
		outcome = executorOutcomeFailed
		reason = classifyProcessError(err)
	}

	return err
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
