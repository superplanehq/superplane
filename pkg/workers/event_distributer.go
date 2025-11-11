package workers

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"github.com/superplanehq/superplane/pkg/workers/eventdistributer"
)

// EventDistributer coordinates message consumption from RabbitMQ
// and distributes events to websocket clients
type EventDistributer struct {
	wsHub    *ws.Hub
	shutdown chan struct{}
}

// NewEventDistributer creates a new event distributer coordinator
func NewEventDistributer(wsHub *ws.Hub) *EventDistributer {
	return &EventDistributer{
		wsHub:    wsHub,
		shutdown: make(chan struct{}),
	}
}

// Start begins consuming messages from RabbitMQ for all relevant routing keys
func (e *EventDistributer) Start() error {
	log.Info("Starting EventDistributer worker")

	amqpURL, err := config.RabbitMQURL()
	if err != nil {
		return fmt.Errorf("failed to get RabbitMQ URL: %w", err)
	}

	// Define the routes to consume with their handlers
	routes := []struct {
		Exchange   string
		RoutingKey string
		Handler    func(delivery tackle.Delivery) error
	}{
		{messages.WorkflowExchange, messages.WorkflowEventCreatedRoutingKey, e.createHandler(eventdistributer.HandleWorkflowEventCreated)},
		{messages.WorkflowExchange, messages.WorkflowExecutionRoutingKey, e.createHandler(eventdistributer.HandleWorkflowExecution)},
	}

	// Start a consumer for each route
	for _, route := range routes {
		go e.consumeMessages(amqpURL, route.Exchange, route.RoutingKey, route.Handler)
	}

	// Block until shutdown signal
	<-e.shutdown
	return nil
}

// createHandler returns a tackle handler that calls the given processing function
func (e *EventDistributer) createHandler(processFn func([]byte, *ws.Hub) error) func(delivery tackle.Delivery) error {
	return func(delivery tackle.Delivery) error {
		// Call the Body() function to get the message body bytes
		messageBody := delivery.Body()
		err := processFn(messageBody, e.wsHub)
		if err != nil {
			log.Errorf("Error processing message: %v", err)
			// Don't return the error to avoid redelivery, just log it
		}
		return nil // Always ack the message regardless of processing success
	}
}

// consumeMessages sets up a consumer for a specific routing key
func (e *EventDistributer) consumeMessages(amqpURL, exchange, routingKey string, handler func(delivery tackle.Delivery) error) {
	//
	// Since we are using distributed architecture and the websocket connections are spreaded between many replicas,
	// We need to create different queue names for each replica, so all will instances receive the messages
	//
	randomSuffix, err := createRandomString(10)
	if err != nil {
		log.Errorf("Error creating random suffix for queueName: %v", err)
		return
	}
	queueName := fmt.Sprintf("superplane.%s.%s.consumer.%s", exchange, routingKey, randomSuffix)

	for {
		log.Infof("Connecting to RabbitMQ queue %s for %s events", queueName, routingKey)

		logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
			"consumer":      "event_distributer",
			"route_handler": routingKey,
		}))

		// Create a new consumer
		consumer := tackle.NewConsumer()
		consumer.SetLogger(logger)

		// Start the consumer with appropriate options
		err := consumer.Start(&tackle.Options{
			URL:            amqpURL,
			RemoteExchange: exchange,
			Service:        queueName,
			RoutingKey:     routingKey,
		}, handler)

		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", routingKey, err)
			// Wait before attempting to reconnect
			time.Sleep(5 * time.Second)
			continue
		}

		// If we reach here, the connection has been closed
		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", routingKey)
		time.Sleep(5 * time.Second)
	}
}

// Shutdown gracefully stops the worker
func (e *EventDistributer) Shutdown(ctx context.Context) error {
	log.Info("Shutting down EventDistributer worker")
	close(e.shutdown)
	return nil
}

func createRandomString(length int) (string, error) {
	charset := "abcdefghijklmnopqrstuvwxyz0123456789"

	if length <= 0 {
		return "", fmt.Errorf("length must be positive")
	}

	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	result := make([]byte, length)
	charsetLen := len(charset)
	for i, b := range randomBytes {
		result[i] = charset[int(b)%charsetLen]
	}

	return string(result), nil
}
