package messages

import (
	"github.com/renderedtext/go-tackle"
	config "github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/logging"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const CanvasExchange = "superplane.canvas-exchange"

func Publish(exchange string, routingKey string, message []byte) error {
	amqpURL, err := config.RabbitMQURL()

	if err != nil {
		return err
	}

	publisher, err := tackle.NewPublisher(amqpURL, tackle.PublisherOptions{})
	if err != nil {
		return err
	}
	publisher.SetLogger(logging.NewQuietPublisherLogger())
	defer publisher.Close()

	if err := publisher.ExchangeDeclare(exchange); err != nil {
		return err
	}

	return publisher.Publish(&tackle.PublishParams{
		Body:       message,
		RoutingKey: routingKey,
		Exchange:   exchange,
	})
}

func toBytes(m protoreflect.ProtoMessage) []byte {
	body, err := proto.Marshal(m)
	if err != nil {
		return nil
	}
	return body
}
