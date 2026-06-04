package workers

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	logrus "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/crypto"
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
	Consumer       *tackle.Consumer
	RabbitMQURL    string
	GitProvider    git.Provider
	Registry       *registry.Registry
	Encryptor      crypto.Encryptor
	AuthService    authorization.Authorization
	WebhookBaseURL string
	semaphore      *semaphore.Weighted
}

func NewRepositoryMaterializerWorker(
	rabbitMQURL string,
	gitProvider git.Provider,
	registry *registry.Registry,
	encryptor crypto.Encryptor,
	authService authorization.Authorization,
	webhookBaseURL string,
) *RepositoryMaterializerWorker {
	logger := logging.NewTackleLogger(logrus.StandardLogger().WithFields(logrus.Fields{
		"worker": "RepositoryMaterializer",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &RepositoryMaterializerWorker{
		Consumer:       consumer,
		RabbitMQURL:    rabbitMQURL,
		GitProvider:    gitProvider,
		Registry:       registry,
		Encryptor:      encryptor,
		AuthService:    authService,
		WebhookBaseURL: webhookBaseURL,
		semaphore:      semaphore.NewWeighted(25),
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

	canvasID, err := uuid.Parse(message.GetCanvasId())
	if err != nil {
		return nil
	}

	if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
		return err
	}
	defer w.semaphore.Release(1)

	canvas, err := models.FindCanvasWithoutOrgScope(canvasID)
	if err != nil {
		w.log("Error finding canvas %s: %v", canvasID, err)
		return err
	}

	if message.GetMaterializationStatus() == models.MaterializationStatusDeleted {
		return nil
	}

	var removed []string
	reconcileErr := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		removed, err = materialize.ReconcileDraftBranchDeletionsFromGit(
			context.Background(),
			tx,
			w.GitProvider,
			canvasID,
			materialize.ReconcileDraftBranchDeletionsOptions{},
		)
		return err
	})
	if reconcileErr != nil {
		w.log("Error reconciling draft branch deletions for canvas %s: %v", canvasID, reconcileErr)
		return reconcileErr
	}
	materialize.PublishDraftBranchDeletionEvents(canvasID.String(), removed)

	headSHA := message.GetHeadSha()
	if headSHA == "" {
		repository, headErr := models.FindRepositoryUnscoped(canvasID)
		if headErr != nil {
			w.log("Error finding repository for canvas %s: %v", canvasID, headErr)
			return headErr
		}

		headSHA, err = w.GitProvider.Head(context.Background(), repository.RepoID, message.GetBranch())
		if err != nil {
			w.log("Error reading branch head for canvas %s branch %s: %v", canvasID, message.GetBranch(), err)
			return err
		}
	}

	if message.GetBranch() == models.CanvasGitBranchMain {
		if w.shouldSkipMainMaterialization(canvasID, headSHA) {
			return nil
		}

		err = database.Conn().Transaction(func(tx *gorm.DB) error {
			_, syncErr := materialize.SyncLiveFromGit(
				context.Background(),
				tx,
				w.GitProvider,
				w.Registry,
				w.Encryptor,
				w.AuthService,
				w.WebhookBaseURL,
				canvas.OrganizationID,
				canvasID,
				materialize.SyncLiveFromGitOptions{HeadSHA: headSHA},
			)
			return syncErr
		})
		if err != nil {
			w.log("Error materializing live canvas %s at %s: %v", canvasID, headSHA, err)
			return err
		}

		return nil
	}

	state, err := models.FindRepositoryMaterializationState(canvasID, message.GetBranch())
	if err == nil && state.MaterializedSHA == headSHA && state.Status == models.MaterializationStatusReady {
		draftBranch, draftErr := models.FindDraftBranch(canvasID, message.GetBranch())
		if draftErr == nil && draftBranch != nil {
			return nil
		}
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var createdBy *uuid.UUID
		if pushedByUserID := strings.TrimSpace(message.GetPushedByUserId()); pushedByUserID != "" {
			if userID, parseErr := uuid.Parse(pushedByUserID); parseErr == nil {
				createdBy = &userID
			}
		}

		_, syncErr := materialize.SyncDraftBranchFromGit(
			context.Background(),
			tx,
			w.GitProvider,
			w.Registry,
			canvas.OrganizationID,
			canvasID,
			message.GetBranch(),
			materialize.SyncDraftBranchOptions{
				HeadSHA:   headSHA,
				CreatedBy: createdBy,
			},
		)
		return syncErr
	})
	if err != nil {
		w.log("Error materializing canvas %s branch %s: %v", canvasID, message.GetBranch(), err)
		return err
	}

	return nil
}

func (w *RepositoryMaterializerWorker) shouldSkipMainMaterialization(canvasID uuid.UUID, headSHA string) bool {
	state, err := models.FindRepositoryMaterializationState(canvasID, models.CanvasGitBranchMain)
	if err != nil {
		return false
	}
	if state.MaterializedSHA != headSHA || state.Status != models.MaterializationStatusReady {
		return false
	}

	canvas, err := models.FindCanvasWithoutOrgScope(canvasID)
	if err != nil {
		return false
	}

	return canvas.LiveVersionID != nil && *canvas.LiveVersionID == headSHA
}

func (w *RepositoryMaterializerWorker) log(format string, v ...any) {
	log.Printf("[RepositoryMaterializer] "+format, v...)
}
