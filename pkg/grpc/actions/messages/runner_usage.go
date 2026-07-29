package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/runners"
)

const RunnersExchange = "superplane.runners-exchange"

const RunnerTaskFinishedRoutingKey = "runner.task.finished"

type RunnerTaskFinishedMessage struct {
	message *pb.RunnerTaskFinishedMessage
}

func NewRunnerTaskFinishedMessage(organizationID, taskID string, durationSeconds int64) RunnerTaskFinishedMessage {
	return RunnerTaskFinishedMessage{
		message: &pb.RunnerTaskFinishedMessage{
			OrganizationId:  organizationID,
			TaskId:          taskID,
			DurationSeconds: durationSeconds,
		},
	}
}

func (m RunnerTaskFinishedMessage) Publish() error {
	return Publish(RunnersExchange, RunnerTaskFinishedRoutingKey, toBytes(m.message))
}
