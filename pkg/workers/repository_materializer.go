package workers

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	logrus "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const (
	repositoryMaterializerServiceName = "superplane." + messages.CanvasExchange + "." + messages.RepositoryBranchUpdatedRoutingKey + ".repository-materializer"
	repositoryMaterializerConnection  = "superplane"
)

type RepositoryMaterializerWorker struct {
	Consumer    *tackle.Consumer
	RabbitMQURL string
	GitProvider git.Provider
	Registry    *registry.Registry
	semaphore   *semaphore.Weighted
}

func NewRepositoryMaterializerWorker(rabbitMQURL string, gitProvider git.Provider, registry *registry.Registry) *RepositoryMaterializerWorker {
	logger := logging.NewTackleLogger(logrus.StandardLogger().WithFields(logrus.Fields{
		"worker": "RepositoryMaterializer",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &RepositoryMaterializerWorker{
		Consumer:    consumer,
		RabbitMQURL: rabbitMQURL,
		GitProvider: gitProvider,
		Registry:    registry,
		semaphore:   semaphore.NewWeighted(25),
	}
}

func (w *RepositoryMaterializerWorker) Start(ctx context.Context) {
	go w.startConsumerLoop(ctx)
	<-ctx.Done()
	w.Stop()
}

func (w *RepositoryMaterializerWorker) Stop() {
	w.Consumer.Stop()
}

func (w *RepositoryMaterializerWorker) startConsumerLoop(ctx context.Context) {
	options := tackle.Options{
		URL:            w.RabbitMQURL,
		ConnectionName: repositoryMaterializerConnection,
		Service:        repositoryMaterializerServiceName,
		RemoteExchange: messages.CanvasExchange,
		RoutingKey:     messages.RepositoryBranchUpdatedRoutingKey,
	}

	for {
		if ctx.Err() != nil {
			return
		}

		log.Println("Connecting to RabbitMQ queue for repository branch materialization")

		err := w.Consumer.Start(&options, w.ConsumeRepositoryBranchUpdated)
		if ctx.Err() != nil {
			return
		}

		if err != nil {
			w.log("Error consuming repository branch updated messages: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		w.log("Connection to RabbitMQ closed for repository branch materialization, reconnecting...")
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (w *RepositoryMaterializerWorker) ConsumeRepositoryBranchUpdated(delivery tackle.Delivery) error {
	message := &pb.RepositoryBranchUpdatedMessage{}
	if err := proto.Unmarshal(delivery.Body(), message); err != nil {
		w.log("Error unmarshaling repository branch updated message: %v", err)
		return err
	}

	if message.GetBranch() == models.CanvasGitBranchMain {
		return nil
	}

	canvasID, err := uuid.Parse(message.GetCanvasId())
	if err != nil {
		return nil
	}

	if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
		return err
	}
	defer w.semaphore.Release(1)

	repository, err := models.FindRepositoryUnscoped(canvasID)
	if err != nil {
		w.log("Error finding repository for canvas %s: %v", canvasID, err)
		return err
	}

	canvas, err := models.FindCanvasWithoutOrgScope(canvasID)
	if err != nil {
		w.log("Error finding canvas %s: %v", canvasID, err)
		return err
	}

	headSHA := message.GetHeadSha()
	if headSHA == "" {
		headSHA, err = w.GitProvider.Head(context.Background(), repository.RepoID, message.GetBranch())
		if err != nil {
			w.log("Error reading branch head for canvas %s branch %s: %v", canvasID, message.GetBranch(), err)
			return err
		}
	}

	state, err := models.FindRepositoryMaterializationState(canvasID, message.GetBranch())
	if err == nil && state.MaterializedSHA == headSHA && state.Status == models.MaterializationStatusReady {
		return nil
	}

	var draftBranch *models.CanvasDraftBranch
	draftBranch, _ = models.FindDraftBranch(canvasID, message.GetBranch())

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		mat := &materialize.DraftMaterializer{GitProvider: w.GitProvider, Registry: w.Registry}
		var ownerID *uuid.UUID
		if draftBranch != nil {
			ownerID = draftBranch.OwnerID
		}
		_, matErr := mat.MaterializeDraft(context.Background(), tx, canvas.OrganizationID, canvasID, message.GetBranch(), headSHA, ownerID)
		return matErr
	})
	if err != nil {
		w.log("Error materializing canvas %s branch %s: %v", canvasID, message.GetBranch(), err)
		return err
	}

	return nil
}

func (w *RepositoryMaterializerWorker) log(format string, v ...any) {
	log.Printf("[RepositoryMaterializer] "+format, v...)
}
