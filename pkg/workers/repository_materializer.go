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
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
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

	materializer := &materialize.BranchMaterializer{
		GitProvider:    w.GitProvider,
		Registry:       w.Registry,
		Encryptor:      w.Encryptor,
		AuthService:    w.AuthService,
		WebhookBaseURL: w.WebhookBaseURL,
	}

	// Deletion notifications carry no branch tip to materialize: the handler has
	// already removed the branch from git, so drop the database projection.
	if message.GetMaterializationStatus() == pb.MaterializationStatus_MATERIALIZATION_STATUS_DELETED {
		if err := materializer.ReconcileBranchDeletion(context.Background(), canvasID, message.GetBranch()); err != nil {
			w.log("Error reconciling deletion for canvas %s branch %s: %v", canvasID, message.GetBranch(), err)
			return err
		}
		return nil
	}

	// Materialization requests must name the commit to project. A message
	// without a head SHA is malformed and can never succeed, so drop it rather
	// than retry it forever.
	if strings.TrimSpace(message.GetHeadSha()) == "" {
		w.log("Dropping materialization for canvas %s branch %s: missing head sha", canvasID, message.GetBranch())
		return nil
	}

	if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
		return err
	}
	defer w.semaphore.Release(1)

	var pushedBy *uuid.UUID
	if pushedByUserID := strings.TrimSpace(message.GetPushedByUserId()); pushedByUserID != "" {
		if userID, parseErr := uuid.Parse(pushedByUserID); parseErr == nil {
			pushedBy = &userID
		}
	}

	if err := materializer.MaterializeBranch(context.Background(), canvasID, message.GetBranch(), message.GetHeadSha(), pushedBy); err != nil {
		w.log("Error materializing canvas %s branch %s: %v", canvasID, message.GetBranch(), err)
		return err
	}

	return nil
}

func (w *RepositoryMaterializerWorker) log(format string, v ...any) {
	log.Printf("[RepositoryMaterializer] "+format, v...)
}
