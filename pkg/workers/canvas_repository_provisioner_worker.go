package workers

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	logrus "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/git"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const (
	canvasRepositoryProvisionerServiceName   = "superplane." + messages.CanvasExchange + "." + messages.CanvasCreatedRoutingKey + ".canvas-repository-provisioner"
	canvasRepositoryProvisionerConnection    = "superplane"
	canvasRepositoryProvisionerBatch         = 100
	canvasRepositoryProvisionerBackfillEvery = 5 * time.Second
)

type CanvasRepositoryProvisionerWorker struct {
	Consumer    *tackle.Consumer
	RabbitMQURL string
	Storage     git.Provider
	Options     canvases.CanvasRepositoryStorageOptions
	semaphore   *semaphore.Weighted
}

func NewCanvasRepositoryProvisionerWorker(
	rabbitMQURL string,
	storage git.Provider,
	options canvases.CanvasRepositoryStorageOptions,
) *CanvasRepositoryProvisionerWorker {
	logger := logging.NewTackleLogger(logrus.StandardLogger().WithFields(logrus.Fields{
		"worker": "canvas_repository_provisioner",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &CanvasRepositoryProvisionerWorker{
		Consumer:    consumer,
		RabbitMQURL: rabbitMQURL,
		Storage:     storage,
		Options:     options,
		semaphore:   semaphore.NewWeighted(25),
	}
}

func (w *CanvasRepositoryProvisionerWorker) Start(ctx context.Context) {
	if w.Storage == nil {
		log.Println("Canvas repository provisioner not started because canvas storage is disabled")
		return
	}

	if count, err := models.ResetStuckProvisioningCanvasRepositories(); err != nil {
		w.log("Error resetting stuck provisioning canvas repositories: %v", err)
	} else if count > 0 {
		w.log("Reset %d stuck provisioning canvas repository record(s) back to pending", count)
	}

	go w.startBackfillLoop(ctx)
	go w.startConsumerLoop(ctx)

	<-ctx.Done()
	w.Stop()
}

func (w *CanvasRepositoryProvisionerWorker) Stop() {
	w.Consumer.Stop()
}

func (w *CanvasRepositoryProvisionerWorker) startConsumerLoop(ctx context.Context) {
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

func (w *CanvasRepositoryProvisionerWorker) startBackfillLoop(ctx context.Context) {
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

func (w *CanvasRepositoryProvisionerWorker) backfill(ctx context.Context) {
	repositories, err := models.ListPendingCanvasRepositories(canvasRepositoryProvisionerBatch)
	if err != nil {
		w.log("Error listing pending canvas repositories: %v", err)
		return
	}

	for _, repository := range repositories {
		if err := w.semaphore.Acquire(ctx, 1); err != nil {
			return
		}

		go func(repository models.CanvasRepository) {
			defer w.semaphore.Release(1)

			if err := w.provisionRepository(ctx, repository.CanvasID); err != nil {
				w.log("Error provisioning canvas repository for canvas %s: %v", repository.CanvasID, err)
			}
		}(repository)
	}
}

func (w *CanvasRepositoryProvisionerWorker) ConsumeCanvasCreated(delivery tackle.Delivery) error {
	message := &pb.CanvasMessage{}
	if err := proto.Unmarshal(delivery.Body(), message); err != nil {
		w.log("Error unmarshaling canvas created message: %v", err)
		return err
	}

	canvasID, err := uuid.Parse(message.GetCanvasId())
	if err != nil {
		w.log("Invalid canvas ID %s: %v", message.GetCanvasId(), err)
		return nil
	}

	if err := w.provisionRepository(context.Background(), canvasID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		w.log("Error provisioning canvas repository for canvas %s: %v", canvasID, err)
		return err
	}

	return nil
}

func (w *CanvasRepositoryProvisionerWorker) provisionRepository(ctx context.Context, canvasID uuid.UUID) error {
	repository, err := models.FindCanvasRepository(canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	if repository.Status != models.CanvasRepositoryStatusPending {
		return nil
	}

	canvas, err := models.FindCanvas(repository.OrganizationID, canvasID)
	if err != nil {
		return err
	}
	if canvas.IsTemplate {
		return nil
	}

	lockedRepository, err := canvases.BeginCanvasRepositoryProvisioning(canvasID)
	if err != nil {
		return err
	}
	if lockedRepository == nil {
		return nil
	}

	branch := defaultBranch(w.Options.DefaultBranch)

	_, err = w.Storage.CreateRepository(ctx, git.RepositorySpec{
		OrganizationID: lockedRepository.OrganizationID,
		CanvasID:       lockedRepository.CanvasID,
		RepoID:         lockedRepository.RepoID,
		DefaultBranch:  branch,
	})
	if err != nil {
		return canvases.FailCanvasRepositoryProvisioning(lockedRepository, err)
	}

	if err := w.Storage.InitRepository(ctx, git.RepositoryRef{
		RepoID:        lockedRepository.RepoID,
		DefaultBranch: branch,
	}, branch); err != nil {
		return canvases.FailCanvasRepositoryProvisioning(lockedRepository, err)
	}

	return canvases.CompleteCanvasRepositoryProvisioning(lockedRepository)
}

func defaultBranch(branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "main"
	}
	return branch
}

func (w *CanvasRepositoryProvisionerWorker) log(format string, v ...any) {
	log.Printf("[CanvasRepositoryProvisioner] "+format, v...)
}
