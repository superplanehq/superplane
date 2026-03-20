package workers

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	organizationpb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const (
	UsageSyncServiceName    = "superplane." + messages.CanvasExchange + "." + messages.OrganizationCreatedRoutingKey + ".worker-consumer"
	UsageSyncConnectionName = "superplane"
	usageSyncBackfillBatch  = 100
	usageSyncBackfillEvery  = 1 * time.Minute
)

type UsageSyncWorker struct {
	Consumer     *tackle.Consumer
	RabbitMQURL  string
	UsageService usage.Service
}

func NewUsageSyncWorker(rabbitMQURL string, usageService usage.Service) *UsageSyncWorker {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"worker": "usage_sync",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &UsageSyncWorker{
		Consumer:     consumer,
		RabbitMQURL:  rabbitMQURL,
		UsageService: usageService,
	}
}

func (w *UsageSyncWorker) Start(ctx context.Context) {
	if w.UsageService == nil || !w.UsageService.Enabled() {
		log.Info("Usage sync worker not started because usage is disabled")
		return
	}

	go w.startBackfillLoop(ctx)

	options := tackle.Options{
		URL:            w.RabbitMQURL,
		ConnectionName: UsageSyncConnectionName,
		Service:        UsageSyncServiceName,
		RemoteExchange: messages.CanvasExchange,
		RoutingKey:     messages.OrganizationCreatedRoutingKey,
	}

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.OrganizationCreatedRoutingKey)

		err := w.Consumer.Start(&options, w.Consume)
		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.OrganizationCreatedRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.OrganizationCreatedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (w *UsageSyncWorker) Stop() {
	w.Consumer.Stop()
}

func (w *UsageSyncWorker) Consume(delivery tackle.Delivery) error {
	data := &organizationpb.OrganizationCreated{}
	err := proto.Unmarshal(delivery.Body(), data)
	if err != nil {
		log.Errorf("Error unmarshaling organization created message: %v", err)
		return err
	}

	organizationID, err := uuid.Parse(data.OrganizationId)
	if err != nil {
		log.Errorf("Invalid organization ID %s: %v", data.OrganizationId, err)
		return nil
	}

	if err := usage.SyncOrganization(context.Background(), w.UsageService, organizationID.String()); err != nil {
		if errors.Is(err, usage.ErrNoBillingAccountCandidate) {
			log.Warnf("Skipping usage sync for organization %s: %v", organizationID, err)
			return nil
		}

		log.Errorf("Failed to sync usage for organization %s: %v", organizationID, err)
		return err
	}

	log.Infof("Successfully synced usage for organization %s", organizationID)
	return nil
}

func (w *UsageSyncWorker) startBackfillLoop(ctx context.Context) {
	w.backfill(ctx)

	ticker := time.NewTicker(usageSyncBackfillEvery)
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

func (w *UsageSyncWorker) backfill(ctx context.Context) {
	organizations, err := models.ListOrganizationsPendingUsageSync(usageSyncBackfillBatch)
	if err != nil {
		log.Errorf("Failed to list organizations pending usage sync: %v", err)
		return
	}

	if len(organizations) == 0 {
		return
	}

	for _, organization := range organizations {
		err := usage.SyncOrganization(ctx, w.UsageService, organization.ID.String())
		if err == nil {
			log.Infof("Backfilled usage sync for organization %s", organization.ID)
			continue
		}

		if errors.Is(err, usage.ErrNoBillingAccountCandidate) || errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warnf("Skipping usage sync backfill for organization %s: %v", organization.ID, err)
			continue
		}

		log.Errorf("Failed to backfill usage sync for organization %s: %v", organization.ID, err)
	}
}
