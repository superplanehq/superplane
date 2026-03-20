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
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const (
	UsageSyncCreatedServiceName     = "superplane." + messages.CanvasExchange + "." + messages.OrganizationCreatedRoutingKey + ".worker-consumer"
	UsageSyncPlanChangedServiceName = "superplane." + messages.CanvasExchange + "." + messages.OrganizationPlanChangedRoutingKey + ".worker-consumer"
	UsageSyncConnectionName         = "superplane"
	usageSyncBackfillBatch          = 100
	usageSyncBackfillEvery          = 1 * time.Minute
	usageLimitsRefreshBatch         = 100
	usageLimitsRefreshAfter         = 1 * time.Hour
)

type UsageSyncWorker struct {
	CreatedConsumer     *tackle.Consumer
	PlanChangedConsumer *tackle.Consumer
	RabbitMQURL         string
	UsageService        usage.Service
}

func NewUsageSyncWorker(rabbitMQURL string, usageService usage.Service) *UsageSyncWorker {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"worker": "usage_sync",
	}))

	createdConsumer := tackle.NewConsumer()
	createdConsumer.SetLogger(logger)

	planChangedConsumer := tackle.NewConsumer()
	planChangedConsumer.SetLogger(logger)

	return &UsageSyncWorker{
		CreatedConsumer:     createdConsumer,
		PlanChangedConsumer: planChangedConsumer,
		RabbitMQURL:         rabbitMQURL,
		UsageService:        usageService,
	}
}

func (w *UsageSyncWorker) Start(ctx context.Context) {
	if w.UsageService == nil || !w.UsageService.Enabled() {
		log.Info("Usage sync worker not started because usage is disabled")
		return
	}

	go w.startBackfillLoop(ctx)

	go w.startConsumerLoop(
		ctx,
		w.CreatedConsumer,
		UsageSyncCreatedServiceName,
		messages.OrganizationCreatedRoutingKey,
		w.Consume,
	)

	go w.startConsumerLoop(
		ctx,
		w.PlanChangedConsumer,
		UsageSyncPlanChangedServiceName,
		messages.OrganizationPlanChangedRoutingKey,
		w.ConsumeOrganizationPlanChanged,
	)

	<-ctx.Done()
	w.Stop()
}

func (w *UsageSyncWorker) Stop() {
	w.CreatedConsumer.Stop()
	w.PlanChangedConsumer.Stop()
}

func (w *UsageSyncWorker) startConsumerLoop(
	ctx context.Context,
	consumer *tackle.Consumer,
	serviceName string,
	routingKey string,
	handler func(tackle.Delivery) error,
) {
	options := tackle.Options{
		URL:            w.RabbitMQURL,
		ConnectionName: UsageSyncConnectionName,
		Service:        serviceName,
		RemoteExchange: messages.CanvasExchange,
		RoutingKey:     routingKey,
	}

	for {
		if ctx.Err() != nil {
			return
		}

		log.Infof("Connecting to RabbitMQ queue for %s events", routingKey)

		err := consumer.Start(&options, handler)
		if ctx.Err() != nil {
			return
		}

		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", routingKey, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", routingKey)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
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

func (w *UsageSyncWorker) ConsumeOrganizationPlanChanged(delivery tackle.Delivery) error {
	data := &usagepb.OrganizationPlanChanged{}
	err := proto.Unmarshal(delivery.Body(), data)
	if err != nil {
		log.Errorf("Error unmarshaling organization plan changed message: %v", err)
		return err
	}

	organizationID, err := uuid.Parse(data.OrganizationId)
	if err != nil {
		log.Errorf("Invalid organization ID %s: %v", data.OrganizationId, err)
		return nil
	}

	var retentionWindowDays *int32
	if data.Limits != nil {
		retentionWindowDays = &data.Limits.RetentionWindowDays
	}

	syncedAt := time.Now()
	if data.Timestamp != nil {
		syncedAt = data.Timestamp.AsTime()
	}

	applied, err := models.MarkOrganizationUsageLimitsSyncedIfNewer(organizationID.String(), retentionWindowDays, syncedAt)
	if err != nil {
		log.Errorf("Failed to apply usage plan change for organization %s: %v", organizationID, err)
		return err
	}

	if !applied {
		log.Warnf("Skipping stale or unknown usage plan change for organization %s", organizationID)
		return nil
	}

	log.Infof("Applied usage plan change for organization %s", organizationID)
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
	} else {
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

	staleBefore := time.Now().Add(-usageLimitsRefreshAfter)
	organizations, err = models.ListOrganizationsPendingUsageLimitsRefresh(staleBefore, usageLimitsRefreshBatch)
	if err != nil {
		log.Errorf("Failed to list organizations pending usage limits refresh: %v", err)
		return
	}

	for _, organization := range organizations {
		_, err := usage.RefreshOrganizationLimitsCache(ctx, w.UsageService, organization.ID.String())
		if err == nil {
			log.Infof("Refreshed usage limits cache for organization %s", organization.ID)
			continue
		}

		if errors.Is(err, usage.ErrNoBillingAccountCandidate) || errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warnf("Skipping usage limits refresh for organization %s: %v", organization.ID, err)
			continue
		}

		if status.Code(err) == codes.ResourceExhausted {
			log.Warnf("Skipping usage limits refresh for organization %s: %v", organization.ID, err)
			continue
		}

		log.Errorf("Failed to refresh usage limits cache for organization %s: %v", organization.ID, err)
	}
}
