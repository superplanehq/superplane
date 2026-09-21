package workers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

const (
	FactorySentryResolveServiceName    = "superplane" + "." + messages.CanvasExchange + "." + messages.FactoryWorkOrderNotificationRoutingKey + ".sentry-resolve-consumer"
	FactorySentryResolveConnectionName = "superplane"
	sentryAppName                      = "sentry"
)

type sentryWorkOrderOrigin struct {
	IntegrationID uuid.UUID
	IssueID       string
}

// FactorySentryResolveConsumer resolves the originating Sentry issue after a
// Sentry-intake task closes as completed. It shares the work-order
// notification routing key with FactoryNotificationConsumer and uses its
// own queue so Sentry failures never block email delivery.
type FactorySentryResolveConsumer struct {
	Consumer    *tackle.Consumer
	RabbitMQURL string
	Encryptor   crypto.Encryptor
	Registry    *registry.Registry
	httpContext core.HTTPContext
}

func NewFactorySentryResolveConsumer(
	rabbitMQURL string,
	encryptor crypto.Encryptor,
	componentRegistry *registry.Registry,
) *FactorySentryResolveConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "factory_sentry_resolve",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &FactorySentryResolveConsumer{
		RabbitMQURL: rabbitMQURL,
		Consumer:    consumer,
		Encryptor:   encryptor,
		Registry:    componentRegistry,
	}
}

func (c *FactorySentryResolveConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: FactorySentryResolveConnectionName,
		Service:        FactorySentryResolveServiceName,
		RemoteExchange: messages.CanvasExchange,
		RoutingKey:     messages.FactoryWorkOrderNotificationRoutingKey,
	}

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.FactoryWorkOrderNotificationRoutingKey)

		err := c.Consumer.Start(&options, c.Consume)
		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.FactoryWorkOrderNotificationRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.FactoryWorkOrderNotificationRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (c *FactorySentryResolveConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *FactorySentryResolveConsumer) Consume(delivery tackle.Delivery) error {
	var message messages.FactoryWorkOrderNotificationMessage
	if err := json.Unmarshal(delivery.Body(), &message); err != nil {
		log.Errorf("Error unmarshaling work order notification message: %v", err)
		return err
	}

	return c.process(database.Conn(), message)
}

func (c *FactorySentryResolveConsumer) process(
	db *gorm.DB,
	message messages.FactoryWorkOrderNotificationMessage,
) error {
	if !isCompletedWorkOrderMessage(message) {
		return nil
	}

	orderID, err := uuid.Parse(message.OrderID)
	if err != nil {
		log.Warnf("Skipping Sentry resolve with invalid order id %q", message.OrderID)
		return nil
	}

	order, err := models.FindUnscopedWorkOrder(db, orderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !matchesWorkOrderMessage(order, message) {
		return nil
	}
	if !isCompletedWorkOrder(order) {
		return nil
	}

	origin, err := resolveSentryWorkOrderOrigin(db, order)
	if err != nil {
		return err
	}
	if origin == nil {
		return nil
	}

	return c.resolveSentryIssue(db, origin)
}

func isCompletedWorkOrderMessage(message messages.FactoryWorkOrderNotificationMessage) bool {
	return message.EventType == factoryevents.EventTypeOrderStatusUpdated &&
		message.ToState == models.FactoryWorkOrderStateClosed &&
		message.Result == models.FactoryWorkOrderResultCompleted
}

func matchesWorkOrderMessage(order *models.FactoryWorkOrder, message messages.FactoryWorkOrderNotificationMessage) bool {
	if order == nil {
		return false
	}
	if message.OrganizationID != "" && order.OrganizationID.String() != message.OrganizationID {
		return false
	}
	if message.FactoryID != "" && order.FactoryID.String() != message.FactoryID {
		return false
	}
	return true
}

func isCompletedWorkOrder(order *models.FactoryWorkOrder) bool {
	return order.State == models.FactoryWorkOrderStateClosed &&
		order.Result == models.FactoryWorkOrderResultCompleted
}

func resolveSentryWorkOrderOrigin(tx *gorm.DB, order *models.FactoryWorkOrder) (*sentryWorkOrderOrigin, error) {
	if order == nil || order.SourceRunID == nil {
		return nil, nil
	}

	run, err := models.FindUnscopedCanvasRun(tx, *order.SourceRunID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	event, err := models.FindRootEventForRun(tx, run.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	issueID, ok := sentry.IssueIDFromEventData(event.Data.Data())
	if !ok {
		return nil, nil
	}

	version, err := models.FindCanvasVersionInTransaction(tx, run.WorkflowID, run.VersionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	integrationID, ok := integrationIDFromVersionNode(version, event.NodeID)
	if !ok {
		return nil, nil
	}

	integration, err := models.FindUnscopedIntegrationInTransaction(tx, integrationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if integration.AppName != sentryAppName || integration.State != models.IntegrationStateReady {
		return nil, nil
	}

	return &sentryWorkOrderOrigin{
		IntegrationID: integration.ID,
		IssueID:       issueID,
	}, nil
}

func integrationIDFromVersionNode(version *models.CanvasVersion, nodeID string) (uuid.UUID, bool) {
	if version == nil {
		return uuid.Nil, false
	}

	for _, node := range version.Nodes {
		if node.ID != nodeID {
			continue
		}
		if node.IntegrationID == nil {
			return uuid.Nil, false
		}

		id, err := uuid.Parse(strings.TrimSpace(*node.IntegrationID))
		if err != nil {
			return uuid.Nil, false
		}
		return id, true
	}

	return uuid.Nil, false
}

func (c *FactorySentryResolveConsumer) resolveSentryIssue(tx *gorm.DB, origin *sentryWorkOrderOrigin) error {
	integration, err := models.FindUnscopedIntegrationInTransaction(tx, origin.IntegrationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if integration.AppName != sentryAppName || integration.State != models.IntegrationStateReady {
		return nil
	}

	client, err := sentry.NewClient(
		c.http(),
		contexts.NewIntegrationContext(tx, nil, integration, c.Encryptor, c.Registry, nil),
	)
	if err != nil {
		return c.mapSentryError(origin.IssueID, err)
	}

	issue, err := client.GetIssue(origin.IssueID)
	if err != nil {
		return c.mapSentryError(origin.IssueID, err)
	}
	if sentry.IssueStatusIsSettled(issue.Status) {
		return nil
	}

	_, err = client.UpdateIssue(origin.IssueID, sentry.UpdateIssueRequest{Status: sentry.IssueStatusResolved})
	if err != nil {
		return c.mapSentryError(origin.IssueID, err)
	}

	return nil
}

func (c *FactorySentryResolveConsumer) mapSentryError(issueID string, err error) error {
	if sentry.IsRetryableAPIError(err) {
		return fmt.Errorf("failed to resolve Sentry issue %s: %w", issueID, err)
	}

	log.WithError(err).Warnf("Skipping Sentry resolve for issue %s", issueID)
	return nil
}

func (c *FactorySentryResolveConsumer) http() core.HTTPContext {
	if c.httpContext != nil {
		return c.httpContext
	}
	return c.Registry.HTTPContext()
}
