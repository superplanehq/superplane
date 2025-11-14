package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const RBACPolicyReloadRoutingKey = "rbac-policy-reload"

type RBACPolicyReloadMessage struct {
	message *pb.ReloadPolicyMessage
}

func NewRBACPolicyReloadMessage() RBACPolicyReloadMessage {
	return RBACPolicyReloadMessage{
		message: &pb.ReloadPolicyMessage{
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m RBACPolicyReloadMessage) Publish() error {
	return Publish(WorkflowExchange, RBACPolicyReloadRoutingKey, toBytes(m.message))
}
