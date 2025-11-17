package workers

import (
	"time"

	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/protobuf/proto"
)

const RBACPolicyReloadServiceName = "superplane" + "." + messages.WorkflowExchange + "." + messages.RBACPolicyReloadRoutingKey + ".worker-consumer"
const RBACPolicyReloadConnectionName = "superplane"

type RBACPolicyReloadConsumer struct {
	Consumer    *tackle.Consumer
	RabbitMQURL string
	AuthService authorization.Authorization
}

func NewRBACPolicyReloadConsumer(rabbitMQURL string, authService authorization.Authorization) *RBACPolicyReloadConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "rbac_policy_reload",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &RBACPolicyReloadConsumer{
		RabbitMQURL: rabbitMQURL,
		Consumer:    consumer,
		AuthService: authService,
	}
}

func (c *RBACPolicyReloadConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: RBACPolicyReloadConnectionName,
		Service:        RBACPolicyReloadServiceName,
		RemoteExchange: messages.WorkflowExchange,
		RoutingKey:     messages.RBACPolicyReloadRoutingKey,
	}

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.RBACPolicyReloadRoutingKey)

		err := c.Consumer.Start(&options, c.Consume)
		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.RBACPolicyReloadRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.RBACPolicyReloadRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (c *RBACPolicyReloadConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *RBACPolicyReloadConsumer) Consume(delivery tackle.Delivery) error {
	data := &pb.ReloadPolicyMessage{}
	err := proto.Unmarshal(delivery.Body(), data)
	if err != nil {
		log.Errorf("Error unmarshaling RBAC policy reload message: %v", err)
		return err
	}

	log.Info("Received RBAC policy reload message, reloading policies...")

	if err := c.AuthService.LoadPolicy(); err != nil {
		log.Errorf("Failed to reload RBAC policies: %v", err)
		return err
	}

	log.Info("Successfully reloaded RBAC policies")
	return nil
}
