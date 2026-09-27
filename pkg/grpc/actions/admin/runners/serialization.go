package runners

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/runners"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func serializeFleet(fleet *models.RunnerFleet) *pb.Fleet {
	spec := fleet.Spec.Data()

	return &pb.Fleet{
		Id:      fleet.Slug,
		Enabled: fleet.Enabled,
		Spec: &pb.FleetSpec{
			OperatingSystem:            spec.OperatingSystem,
			Architecture:               spec.Architecture,
			CpuMillicores:              spec.CPUMillicores,
			MemoryMb:                   spec.MemoryMB,
			DiskGb:                     spec.DiskGB,
			Capabilities:               append([]string(nil), spec.Capabilities...),
			MaxExecutionTimeoutSeconds: spec.MaxExecutionTimeoutSeconds,
		},
		RunnerVersion: fleet.RunnerVersion,
		CreatedAt:     timestamppb.New(fleet.CreatedAt),
		UpdatedAt:     timestamppb.New(fleet.UpdatedAt),
	}
}

func fleetSpecFromProto(spec *pb.FleetSpec) models.RunnerFleetSpec {
	return models.RunnerFleetSpec{
		OperatingSystem:            spec.GetOperatingSystem(),
		Architecture:               spec.GetArchitecture(),
		CPUMillicores:              spec.GetCpuMillicores(),
		MemoryMB:                   spec.GetMemoryMb(),
		DiskGB:                     spec.GetDiskGb(),
		Capabilities:               append([]string(nil), spec.GetCapabilities()...),
		MaxExecutionTimeoutSeconds: spec.GetMaxExecutionTimeoutSeconds(),
	}
}

func serializeRunner(runner *models.Runner, fleetID string) *pb.Runner {
	out := &pb.Runner{
		Id:            runner.ID.String(),
		FleetId:       fleetID,
		State:         runner.State,
		RunnerVersion: runner.RunnerVersion,
		Ephemeral:     runner.Ephemeral,
		CreatedAt:     timestamppb.New(runner.CreatedAt),
		UpdatedAt:     timestamppb.New(runner.UpdatedAt),
	}
	if runner.RegisteredAt != nil {
		out.RegisteredAt = timestamppb.New(*runner.RegisteredAt)
	}
	if runner.LastSeenAt != nil {
		out.LastSeenAt = timestamppb.New(*runner.LastSeenAt)
	}
	if runner.TerminationReason != nil {
		reason := *runner.TerminationReason
		out.TerminationReason = &reason
	}
	if runner.TerminatedAt != nil {
		out.TerminatedAt = timestamppb.New(*runner.TerminatedAt)
	}
	return out
}

func serializeTask(task *models.RunnerTask, fleetID string) *pb.Task {
	out := &pb.Task{
		Id:             task.ID.String(),
		OrganizationId: task.OrganizationID.String(),
		FleetId:        fleetID,
		State:          task.State,
		QueuedAt:       timestamppb.New(task.QueuedAt),
	}
	if task.RunnerID != nil {
		runnerID := task.RunnerID.String()
		out.RunnerId = &runnerID
	}
	if task.ReservedAt != nil {
		out.ReservedAt = timestamppb.New(*task.ReservedAt)
	}
	if task.StartedAt != nil {
		out.StartedAt = timestamppb.New(*task.StartedAt)
	}
	if task.FinishedAt != nil {
		out.FinishedAt = timestamppb.New(*task.FinishedAt)
	}
	return out
}
