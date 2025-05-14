package messages

import (
	"github.com/renderedtext/go-tackle"
	config "github.com/superplanehq/superplane/pkg/config"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const DeliveryHubCanvasExchange = "superplane.canvas-exchange"

func Publish(exchange string, routingKey string, message []byte) error {
	amqpURL, err := config.RabbitMQURL()

	if err != nil {
		return err
	}

	return tackle.PublishMessage(&tackle.PublishParams{
		Body:       message,
		AmqpURL:    amqpURL,
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
