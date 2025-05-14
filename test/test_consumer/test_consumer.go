package testconsumer

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
)

const TestConsumerService = "TestConsumerService"
const TestExchangeName = "superplane.canvas-exchange"

type TestConsumer struct {
	amqpURL        string
	exchangeName   string
	routingKey     string
	consumer       *tackle.Consumer
	messageChannel chan bool
}

func New(amqpURL string, routingKey string) TestConsumer {
	return TestConsumer{
		amqpURL:        amqpURL,
		exchangeName:   TestExchangeName,
		routingKey:     routingKey,
		messageChannel: make(chan bool),
		consumer:       tackle.NewConsumer(),
	}
}

func (c *TestConsumer) Start() {
	randomServiceName := fmt.Sprintf("%s.%s", TestConsumerService, uuid.NewString())

	go c.consumer.Start(&tackle.Options{
		URL:            c.amqpURL,
		RemoteExchange: c.exchangeName,
		Service:        randomServiceName,
		RoutingKey:     c.routingKey,
	}, func(d tackle.Delivery) error {
		c.messageChannel <- true
		return nil
	})

	c.waitForInitialization()
}

func (c *TestConsumer) waitForInitialization() {
	for c.consumer.State != tackle.StateListening {
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *TestConsumer) Stop() {
	c.consumer.Stop()
}

func (c *TestConsumer) HasReceivedMessage() bool {
	select {
	case <-c.messageChannel:
		return true
	case <-time.After(1000 * time.Millisecond):
		return false
	}
}
