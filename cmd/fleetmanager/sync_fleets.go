package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/fleets/adminclient"
	fleetconfig "github.com/superplanehq/superplane/pkg/fleets/config"
)

func syncConfiguredFleets(
	ctx context.Context,
	admin *adminclient.Client,
	fleets []fleetconfig.Fleet,
) error {
	for _, fleet := range fleets {
		if err := syncConfiguredFleet(ctx, admin, fleet); err != nil {
			return fmt.Errorf("sync fleet %q: %w", fleet.ID, err)
		}
	}
	return nil
}

func syncConfiguredFleet(ctx context.Context, admin *adminclient.Client, fleet fleetconfig.Fleet) error {
	if strings.TrimSpace(fleet.RunnerVersion) == "" {
		return fmt.Errorf("runnerVersion is required")
	}

	request := adminclient.UpdateFleetRequest{
		RunnerVersion: fleet.RunnerVersion,
		Spec: adminclient.FleetSpec{
			OperatingSystem: fleet.Spec.OperatingSystem,
			Architecture:    fleet.Spec.Architecture,
			CPUMillicores:   fleet.Spec.CPUMillicores,
			MemoryMB:        fleet.Spec.MemoryMB,
			DiskGB:          fleet.Spec.DiskGB,
			Capabilities:    fleet.Spec.Capabilities,
		},
	}

	if _, err := admin.DescribeFleet(ctx, fleet.ID); err == nil {
		_, err = admin.UpdateFleet(ctx, fleet.ID, request)
		return err
	} else if !adminclient.IsStatus(err, http.StatusNotFound) {
		return err
	}

	_, err := admin.CreateFleet(ctx, adminclient.CreateFleetRequest{
		FleetID:       fleet.ID,
		RunnerVersion: request.RunnerVersion,
		Spec:          request.Spec,
	})
	if adminclient.IsStatus(err, http.StatusConflict) {
		_, err = admin.UpdateFleet(ctx, fleet.ID, request)
	}
	return err
}
