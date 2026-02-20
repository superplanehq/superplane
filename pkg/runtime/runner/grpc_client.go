package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	runnerpb "github.com/superplanehq/superplane/pkg/runtime/runner/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

type grpcClient struct {
	conn      *grpc.ClientConn
	client    runnerpb.RuntimeRunnerClient
	authToken string
}

func newGRPCClient(cfg Config) (*grpcClient, error) {
	conn, err := grpc.NewClient(
		cfg.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial runtime runner: %w", err)
	}

	return &grpcClient{
		conn:      conn,
		client:    runnerpb.NewRuntimeRunnerClient(conn),
		authToken: cfg.AuthToken,
	}, nil
}

func (c *grpcClient) SetupTrigger(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error) {
	response, err := c.client.SetupTrigger(c.withAuth(ctx), &runnerpb.SetupTriggerRequest{
		Name:    name,
		Request: toProtoEnvelope(req.Request),
		Context: toProtoContext(req.Context),
		Input:   toProtoStruct(req.Input),
	})
	if err != nil {
		return nil, err
	}

	return fromProtoOperationResponse(response), nil
}

func (c *grpcClient) SetupComponent(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error) {
	response, err := c.client.SetupComponent(c.withAuth(ctx), &runnerpb.SetupComponentRequest{
		Name:    name,
		Request: toProtoEnvelope(req.Request),
		Context: toProtoContext(req.Context),
		Input:   toProtoStruct(req.Input),
	})
	if err != nil {
		return nil, err
	}

	return fromProtoOperationResponse(response), nil
}

func (c *grpcClient) ExecuteComponent(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error) {
	response, err := c.client.ExecuteComponent(c.withAuth(ctx), &runnerpb.ExecuteComponentRequest{
		Name:    name,
		Request: toProtoEnvelope(req.Request),
		Context: toProtoContext(req.Context),
		Input:   toProtoStruct(req.Input),
	})
	if err != nil {
		return nil, err
	}

	return fromProtoOperationResponse(response), nil
}

func (c *grpcClient) SyncIntegration(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error) {
	response, err := c.client.SyncIntegration(c.withAuth(ctx), &runnerpb.SyncIntegrationRequest{
		Name:    name,
		Request: toProtoEnvelope(req.Request),
		Context: toProtoContext(req.Context),
		Input:   toProtoStruct(req.Input),
	})
	if err != nil {
		return nil, err
	}

	return fromProtoOperationResponse(response), nil
}

func (c *grpcClient) CleanupIntegration(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error) {
	response, err := c.client.CleanupIntegration(c.withAuth(ctx), &runnerpb.CleanupIntegrationRequest{
		Name:    name,
		Request: toProtoEnvelope(req.Request),
		Context: toProtoContext(req.Context),
		Input:   toProtoStruct(req.Input),
	})
	if err != nil {
		return nil, err
	}

	return fromProtoOperationResponse(response), nil
}

func (c *grpcClient) ListCapabilities(ctx context.Context) ([]Capability, error) {
	response, err := c.client.ListCapabilities(c.withAuth(ctx), &runnerpb.ListCapabilitiesRequest{})
	if err != nil {
		return nil, err
	}

	capabilities := make([]Capability, 0, len(response.Capabilities))
	for _, item := range response.Capabilities {
		capabilities = append(capabilities, Capability{
			Kind:       strings.ToLower(strings.TrimPrefix(item.Kind.String(), "CAPABILITY_KIND_")),
			Name:       item.Name,
			Operations: item.Operations,
			SchemaHash: item.SchemaHash,
		})
	}

	return capabilities, nil
}

func (c *grpcClient) withAuth(ctx context.Context) context.Context {
	if strings.TrimSpace(c.authToken) == "" {
		return ctx
	}

	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.authToken)
}

func toProtoEnvelope(request RequestEnvelope) *runnerpb.RequestEnvelope {
	return &runnerpb.RequestEnvelope{
		RequestId: request.RequestID,
		Version:   request.Version,
		TimeoutMs: request.TimeoutMS,
	}
}

func toProtoContext(context RuntimeContext) *runnerpb.RuntimeContext {
	return &runnerpb.RuntimeContext{
		OrganizationId: context.OrganizationID,
		WorkspaceId:    context.WorkspaceID,
		UserId:         context.UserID,
		CanvasId:       context.CanvasID,
		NodeId:         context.NodeID,
		Labels:         toStringMap(context.Labels),
		Metadata:       toProtoStruct(context.Metadata),
	}
}

func toStringMap(input map[string]any) map[string]string {
	if len(input) == 0 {
		return nil
	}

	output := map[string]string{}
	for key, value := range input {
		output[key] = fmt.Sprintf("%v", value)
	}

	return output
}

func toProtoStruct(input any) *structpb.Struct {
	if input == nil {
		return nil
	}

	if object, ok := input.(map[string]any); ok {
		result, err := structpb.NewStruct(object)
		if err == nil {
			return result
		}
	}

	data, err := json.Marshal(input)
	if err != nil {
		return nil
	}

	object := map[string]any{}
	if err := json.Unmarshal(data, &object); err != nil {
		return nil
	}

	result, err := structpb.NewStruct(object)
	if err != nil {
		return nil
	}

	return result
}

func fromProtoOperationResponse(response *runnerpb.RuntimeResponse) *OperationResponse {
	output := map[string]any{}
	if response.GetOutput() != nil {
		output = response.GetOutput().AsMap()
	}

	var operationError *Error
	if response.GetError() != nil {
		operationError = &Error{
			Code:    strings.ToLower(strings.TrimPrefix(response.GetError().Code.String(), "RUNTIME_ERROR_CODE_")),
			Message: response.GetError().Message,
		}
		if response.GetError().Details != nil {
			operationError.Details = response.GetError().Details.AsMap()
		}
	}

	logs := make([]Log, 0, len(response.GetLogs()))
	for _, item := range response.GetLogs() {
		logEntry := Log{
			Level:   item.Level,
			Message: item.Message,
		}
		if item.Fields != nil {
			logEntry.Fields = item.Fields.AsMap()
		}
		logs = append(logs, logEntry)
	}

	return &OperationResponse{
		OK:      response.Ok,
		Output:  output,
		Logs:    logs,
		Error:   operationError,
		Metrics: response.Metrics,
	}
}
