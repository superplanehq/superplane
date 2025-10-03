package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/alertworker"
)

type AlertWorker struct {
	shutdown chan struct{}
}

func NewAlertWorker() *AlertWorker {
	return &AlertWorker{
		shutdown: make(chan struct{}),
	}
}

func (e *AlertWorker) Start() error {
	log.Info("Starting AlertWorker worker")

	amqpURL, err := config.RabbitMQURL()
	if err != nil {
		return fmt.Errorf("failed to get RabbitMQ URL: %w", err)
	}

	routes := []struct {
		Exchange   string
		RoutingKey string
		Handler    func(delivery tackle.Delivery) error
	}{
		{messages.DeliveryHubCanvasExchange, messages.EventRejectionCreatedRoutingKey, e.createHandler(alertworker.HandleEventRejectionCreated)},
	}

	for _, route := range routes {
		go e.consumeMessages(amqpURL, route.Exchange, route.RoutingKey, route.Handler)
	}

	<-e.shutdown
	return nil
}

func (e *AlertWorker) createHandler(processFn func([]byte) (*models.Alert, error)) func(delivery tackle.Delivery) error {
	return func(delivery tackle.Delivery) error {
		messageBody := delivery.Body()
		alert, err := processFn(messageBody)
		if err != nil {
			log.Errorf("Error processing alert creation message: %v", err)
			return err
		}

		err = messages.NewAlertCreatedMessage(alert).Publish()
		if err != nil {
			log.Errorf("Error publishing alert created message: %v", err)
			return err
		}

		return nil
	}
}

func (e *AlertWorker) consumeMessages(amqpURL, exchange, routingKey string, handler func(delivery tackle.Delivery) error) {
	queueName := fmt.Sprintf("superplane.%s.%s.consumer", exchange, routingKey)

	for {
		log.Infof("Connecting to RabbitMQ queue %s for %s events", queueName, routingKey)

		logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
			"consumer":      "event_distributer",
			"route_handler": routingKey,
		}))

		consumer := tackle.NewConsumer()
		consumer.SetLogger(logger)

		err := consumer.Start(&tackle.Options{
			URL:            amqpURL,
			RemoteExchange: exchange,
			Service:        queueName,
			RoutingKey:     routingKey,
		}, handler)

		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", routingKey, err)

			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", routingKey)
		time.Sleep(5 * time.Second)
	}
}

func (e *AlertWorker) Shutdown(ctx context.Context) error {
	log.Info("Shutting down AlertWorker worker")
	close(e.shutdown)
	return nil
}
