package gcpprovider

import (
	"context"
	"fmt"
	"strings"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

const operationStatusDone = "DONE"

type ComputeAPI interface {
	// InsertInstance returns after the zone operation is done.
	InsertInstance(ctx context.Context, project, zone string, instance *compute.Instance) error
	DeleteInstance(ctx context.Context, project, zone, name string) error
	ListInstances(ctx context.Context, project, filter string) ([]*compute.Instance, error)
}

// operationError carries the errors of a finished zone operation.
// Capacity errors, such as ZONE_RESOURCE_POOL_EXHAUSTED, arrive here.
type operationError struct {
	codes   []string
	message string
}

func (e *operationError) Error() string {
	return fmt.Sprintf("compute operation failed: %s", e.message)
}

type sdkCompute struct {
	service *compute.Service
}

func NewSDK(ctx context.Context, options ...option.ClientOption) (ComputeAPI, error) {
	service, err := compute.NewService(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("create Compute Engine client: %w", err)
	}
	return &sdkCompute{service: service}, nil
}

func (s *sdkCompute) InsertInstance(
	ctx context.Context,
	project, zone string,
	instance *compute.Instance,
) error {
	operation, err := s.service.Instances.Insert(project, zone, instance).Context(ctx).Do()
	if err != nil {
		return err
	}
	return s.waitZoneOperation(ctx, project, zone, operation)
}

func (s *sdkCompute) DeleteInstance(ctx context.Context, project, zone, name string) error {
	_, err := s.service.Instances.Delete(project, zone, name).Context(ctx).Do()
	return err
}

func (s *sdkCompute) ListInstances(
	ctx context.Context,
	project, filter string,
) ([]*compute.Instance, error) {
	var instances []*compute.Instance
	err := s.service.Instances.AggregatedList(project).
		Filter(filter).
		ReturnPartialSuccess(true).
		Pages(ctx, func(page *compute.InstanceAggregatedList) error {
			for _, scoped := range page.Items {
				instances = append(instances, scoped.Instances...)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (s *sdkCompute) waitZoneOperation(
	ctx context.Context,
	project, zone string,
	operation *compute.Operation,
) error {
	for operation.Status != operationStatusDone {
		next, err := s.service.ZoneOperations.Wait(project, zone, operation.Name).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("wait for compute operation %s: %w", operation.Name, err)
		}
		operation = next
	}
	if operation.Error == nil || len(operation.Error.Errors) == 0 {
		return nil
	}
	failure := &operationError{}
	messages := make([]string, 0, len(operation.Error.Errors))
	for _, item := range operation.Error.Errors {
		failure.codes = append(failure.codes, item.Code)
		messages = append(messages, item.Code+": "+item.Message)
	}
	failure.message = strings.Join(messages, "; ")
	return failure
}
