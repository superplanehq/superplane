package control

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/superplanehq/superplane/pkg/config"
)

const Exchange = "superplane.runner-control"

type Notification struct {
	RunnerID string `json:"runner_id,omitempty"`
	FleetID  string `json:"fleet_id,omitempty"`
}

func Publish(notification Notification) error {
	body, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	url, err := config.RabbitMQURL()
	if err != nil {
		return err
	}
	connection, err := amqp.DialConfig(url, rabbitMQConfig("runner-control-publisher"))
	if err != nil {
		return err
	}
	defer connection.Close()
	channel, err := connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()
	if err := channel.ExchangeDeclare(Exchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return channel.PublishWithContext(
		ctx,
		Exchange,
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func Consume(ctx context.Context, gatewayID string, handle func(Notification)) error {
	url, err := config.RabbitMQURL()
	if err != nil {
		return err
	}
	connection, err := amqp.DialConfig(url, rabbitMQConfig("runner-gateway-"+gatewayID))
	if err != nil {
		return err
	}
	defer connection.Close()
	channel, err := connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()
	if err := channel.ExchangeDeclare(Exchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	queue, err := channel.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return err
	}
	if err := channel.QueueBind(queue.Name, "", Exchange, false, nil); err != nil {
		return err
	}
	deliveries, err := channel.Consume(queue.Name, "", true, true, false, false, nil)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("runner-control delivery channel closed")
			}
			var notification Notification
			if json.Unmarshal(delivery.Body, &notification) == nil {
				handle(notification)
			}
		}
	}
}

func rabbitMQConfig(connectionName string) amqp.Config {
	return amqp.Config{
		Properties: amqp.Table{"connection_name": connectionName},
		Dial: func(network, address string) (net.Conn, error) {
			return net.DialTimeout(network, address, 2*time.Second)
		},
	}
}

func ConsumeWithReconnect(ctx context.Context, gatewayID string, handle func(Notification)) {
	for {
		if ctx.Err() != nil {
			return
		}
		_ = Consume(ctx, gatewayID, handle)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}
