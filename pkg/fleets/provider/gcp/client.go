package gcpprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const (
	operationStatusDone       = "DONE"
	warningCodeUnreachable    = "UNREACHABLE"
	defaultOperationWaitRetry = 2 * time.Second
	// Create blocks the fleet loop, so the wait must end.
	defaultOperationWaitTimeout = 5 * time.Minute
)

// errOperationResultUnknown means that Compute Engine accepted the request,
// but the operation result could not be read. The instance can exist.
var errOperationResultUnknown = errors.New("compute operation result is unknown")

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
	service     *compute.Service
	waitRetry   time.Duration
	waitTimeout time.Duration
}

func NewSDK(ctx context.Context, options ...option.ClientOption) (ComputeAPI, error) {
	service, err := compute.NewService(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("create Compute Engine client: %w", err)
	}
	return &sdkCompute{
		service:     service,
		waitRetry:   defaultOperationWaitRetry,
		waitTimeout: defaultOperationWaitTimeout,
	}, nil
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
			found, err := instancesFromPage(page)
			if err != nil {
				return err
			}
			instances = append(instances, found...)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return instances, nil
}

// instancesFromPage fails on unreachable scopes. A partial list would hide
// instances that Fleet Manager must clean up.
func instancesFromPage(page *compute.InstanceAggregatedList) ([]*compute.Instance, error) {
	unreachable := slices.Clone(page.Unreachables)
	var instances []*compute.Instance
	for scope, scoped := range page.Items {
		if scoped.Warning != nil && scoped.Warning.Code == warningCodeUnreachable {
			unreachable = append(unreachable, scope)
			continue
		}
		instances = append(instances, scoped.Instances...)
	}
	if len(unreachable) > 0 {
		slices.Sort(unreachable)
		return nil, fmt.Errorf("list instances: unreachable scopes: %s", strings.Join(unreachable, ", "))
	}
	return instances, nil
}

func (s *sdkCompute) waitZoneOperation(
	ctx context.Context,
	project, zone string,
	operation *compute.Operation,
) error {
	ctx, cancel := context.WithTimeout(ctx, s.waitTimeout)
	defer cancel()
	for operation.Status != operationStatusDone {
		next, err := s.service.ZoneOperations.Wait(project, zone, operation.Name).Context(ctx).Do()
		if err == nil {
			operation = next
			continue
		}
		if !isTemporary(err) || !sleep(ctx, s.waitRetry) {
			return fmt.Errorf("%w: wait for compute operation %s: %w", errOperationResultUnknown, operation.Name, err)
		}
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

func isTemporary(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var apiError *googleapi.Error
	if !errors.As(err, &apiError) {
		return true
	}
	return apiError.Code == http.StatusTooManyRequests || apiError.Code >= http.StatusInternalServerError
}

func sleep(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
