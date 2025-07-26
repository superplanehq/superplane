package actions

import (
	"context"
	"errors"
	"fmt"
	"sort"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	integrationpb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ValidateUUIDs(ids ...string) error {
	for _, id := range ids {
		_, err := uuid.Parse(id)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid UUID: %s", id)
		}
	}

	return nil
}

func ExecutionResultToProto(result string) pb.Execution_Result {
	switch result {
	case models.ResultFailed:
		return pb.Execution_RESULT_FAILED
	case models.ResultPassed:
		return pb.Execution_RESULT_PASSED
	default:
		return pb.Execution_RESULT_UNKNOWN
	}
}

func FindConnectionSourceID(canvas *models.Canvas, connection *pb.Connection) (*uuid.UUID, error) {
	switch connection.Type {
	case pb.Connection_TYPE_STAGE:
		stage, err := canvas.FindStageByName(connection.Name)
		if err != nil {
			return nil, fmt.Errorf("stage %s not found", connection.Name)
		}

		return &stage.ID, nil

	case pb.Connection_TYPE_EVENT_SOURCE:
		eventSource, err := canvas.FindEventSourceByName(connection.Name)
		if err != nil {
			return nil, fmt.Errorf("event source %s not found", connection.Name)
		}

		return &eventSource.ID, nil

	case pb.Connection_TYPE_CONNECTION_GROUP:
		connectionGroup, err := canvas.FindConnectionGroupByName(connection.Name)
		if err != nil {
			return nil, fmt.Errorf("connection group %s not found", connection.Name)
		}

		return &connectionGroup.ID, nil

	default:
		return nil, errors.New("invalid type")
	}
}

func ValidateConnections(canvas *models.Canvas, connections []*pb.Connection) ([]models.Connection, error) {
	cs := []models.Connection{}

	if len(connections) == 0 {
		return nil, fmt.Errorf("connections must not be empty")
	}

	for _, connection := range connections {
		sourceID, err := FindConnectionSourceID(canvas, connection)
		if err != nil {
			return nil, fmt.Errorf("invalid connection: %v", err)
		}

		filters, err := ValidateFilters(connection.Filters)
		if err != nil {
			return nil, err
		}

		cs = append(cs, models.Connection{
			SourceID:       *sourceID,
			SourceName:     connection.Name,
			SourceType:     protoToConnectionType(connection.Type),
			FilterOperator: ProtoToFilterOperator(connection.FilterOperator),
			Filters:        filters,
		})
	}

	return cs, nil
}

func ValidateFilters(in []*pb.Filter) ([]models.Filter, error) {
	filters := []models.Filter{}
	for i, f := range in {
		filter, err := validateFilter(f)
		if err != nil {
			return nil, fmt.Errorf("invalid filter [%d]: %v", i, err)
		}

		filters = append(filters, *filter)
	}

	return filters, nil
}

func validateFilter(filter *pb.Filter) (*models.Filter, error) {
	switch filter.Type {
	case pb.FilterType_FILTER_TYPE_DATA:
		return validateDataFilter(filter.Data)
	case pb.FilterType_FILTER_TYPE_HEADER:
		return validateHeaderFilter(filter.Header)
	default:
		return nil, fmt.Errorf("invalid filter type: %s", filter.Type)
	}
}

func validateDataFilter(filter *pb.DataFilter) (*models.Filter, error) {
	if filter == nil {
		return nil, fmt.Errorf("no filter provided")
	}

	if filter.Expression == "" {
		return nil, fmt.Errorf("expression is empty")
	}

	return &models.Filter{
		Type: models.FilterTypeData,
		Data: &models.DataFilter{
			Expression: filter.Expression,
		},
	}, nil
}

func validateHeaderFilter(filter *pb.HeaderFilter) (*models.Filter, error) {
	if filter == nil {
		return nil, fmt.Errorf("no filter provided")
	}

	if filter.Expression == "" {
		return nil, fmt.Errorf("expression is empty")
	}

	return &models.Filter{
		Type: models.FilterTypeHeader,
		Header: &models.HeaderFilter{
			Expression: filter.Expression,
		},
	}, nil
}

func protoToConnectionType(t pb.Connection_Type) string {
	switch t {
	case pb.Connection_TYPE_STAGE:
		return models.SourceTypeStage
	case pb.Connection_TYPE_EVENT_SOURCE:
		return models.SourceTypeEventSource
	case pb.Connection_TYPE_CONNECTION_GROUP:
		return models.SourceTypeConnectionGroup
	default:
		return ""
	}
}

func ProtoToFilterOperator(in pb.FilterOperator) string {
	switch in {
	case pb.FilterOperator_FILTER_OPERATOR_OR:
		return models.FilterOperatorOr
	default:
		return models.FilterOperatorAnd
	}
}

func filterOperatorToProto(in string) pb.FilterOperator {
	switch in {
	case models.FilterOperatorOr:
		return pb.FilterOperator_FILTER_OPERATOR_OR
	default:
		return pb.FilterOperator_FILTER_OPERATOR_AND
	}
}

func SerializeConnections(in []models.Connection) ([]*pb.Connection, error) {
	connections := []*pb.Connection{}

	for _, c := range in {
		filters, err := serializeFilters(c.Filters)
		if err != nil {
			return nil, fmt.Errorf("invalid filters: %v", err)
		}

		connections = append(connections, &pb.Connection{
			Type:           ConnectionTypeToProto(c.SourceType),
			Name:           c.SourceName,
			FilterOperator: filterOperatorToProto(c.FilterOperator),
			Filters:        filters,
		})
	}

	//
	// Sort them by name so we have some predictability here.
	//
	sort.SliceStable(connections, func(i, j int) bool {
		return connections[i].Name < connections[j].Name
	})

	return connections, nil
}

func serializeFilters(in []models.Filter) ([]*pb.Filter, error) {
	filters := []*pb.Filter{}

	for _, f := range in {
		filter, err := serializeFilter(f)
		if err != nil {
			return nil, fmt.Errorf("invalid filter: %v", err)
		}

		filters = append(filters, filter)
	}

	return filters, nil
}

func serializeFilter(in models.Filter) (*pb.Filter, error) {
	switch in.Type {
	case models.FilterTypeData:
		return &pb.Filter{
			Type: pb.FilterType_FILTER_TYPE_DATA,
			Data: &pb.DataFilter{
				Expression: in.Data.Expression,
			},
		}, nil
	case models.FilterTypeHeader:
		return &pb.Filter{
			Type: pb.FilterType_FILTER_TYPE_HEADER,
			Header: &pb.HeaderFilter{
				Expression: in.Header.Expression,
			},
		}, nil
	default:
		return nil, fmt.Errorf("invalid filter type: %s", in.Type)
	}
}

func ConnectionTypeToProto(t string) pb.Connection_Type {
	switch t {
	case models.SourceTypeStage:
		return pb.Connection_TYPE_STAGE
	case models.SourceTypeEventSource:
		return pb.Connection_TYPE_EVENT_SOURCE
	case models.SourceTypeConnectionGroup:
		return pb.Connection_TYPE_CONNECTION_GROUP
	default:
		return pb.Connection_TYPE_UNKNOWN
	}
}

func ProtoToDomainType(domainType pbAuth.DomainType) (string, error) {
	switch domainType {
	case pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION:
		return models.DomainTypeOrganization, nil
	case pbAuth.DomainType_DOMAIN_TYPE_CANVAS:
		return models.DomainTypeCanvas, nil
	default:
		return "", status.Error(codes.InvalidArgument, "invalid domain type")
	}
}

func DomainTypeToProto(domainType string) pbAuth.DomainType {
	switch domainType {
	case models.DomainTypeCanvas:
		return pbAuth.DomainType_DOMAIN_TYPE_CANVAS
	case models.DomainTypeOrganization:
		return pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION
	default:
		return pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED
	}
}

func ValidateIntegration(canvas *models.Canvas, integrationRef *integrationpb.IntegrationRef) (*models.Integration, error) {
	if integrationRef.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "integration name is required")
	}

	//
	// If the integration used is on the organization level, we need to find it there.
	//
	if integrationRef.DomainType == pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION {
		integration, err := models.FindIntegrationByName(models.DomainTypeOrganization, canvas.OrganizationID, integrationRef.Name)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "integration %s not found", integrationRef.Name)
		}

		return integration, nil
	}

	//
	// Otherwise, we look for it on the canvas level.
	//
	integration, err := models.FindIntegrationByName(models.DomainTypeCanvas, canvas.ID, integrationRef.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "integration %s not found", integrationRef.Name)
	}

	return integration, nil
}

func ValidateResource(ctx context.Context, encryptor crypto.Encryptor, integration *models.Integration, resourceRef *integrationpb.ResourceRef) (integrations.Resource, error) {
	if resourceRef == nil {
		return nil, status.Error(codes.InvalidArgument, "resource reference is required")
	}

	if resourceRef.Type == "" || resourceRef.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "resource type and name are required")
	}

	//
	// If resource record does not exist yet, we need to go to the integration to find it.
	//
	integrationImpl, err := integrations.NewIntegration(ctx, integration, encryptor)
	if err != nil {
		return nil, fmt.Errorf("error starting integration implementation: %v", err)
	}

	resource, err := integrationImpl.Get(resourceRef.Type, resourceRef.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s %s not found: %v", resourceRef.Type, resourceRef.Name, err)
	}

	return resource, nil
}

func GetDomainForSecret(domainTypeForResource string, domainIdForResource *uuid.UUID, domainType pbAuth.DomainType) (string, *uuid.UUID, error) {
	domainTypeForSecret, err := ProtoToDomainType(domainType)
	if err != nil {
		domainTypeForSecret = domainTypeForResource
	}

	//
	// If an organization-level resource is being created,
	// the secret must be on the organization level as well.
	//
	if domainTypeForResource == models.DomainTypeOrganization {
		if domainTypeForSecret != models.DomainTypeOrganization {
			return "", nil, fmt.Errorf("integration on organization level must use organization-level secret")
		}

		return domainTypeForSecret, domainIdForResource, nil
	}

	//
	// If a canvas-level resource is being created and a canvas-level secret is being used,
	// we can just re-use the same domain type and ID for the resource.
	//
	if domainTypeForSecret == models.DomainTypeCanvas {
		return domainTypeForSecret, domainIdForResource, nil
	}

	//
	// If a canvas-level resource is being created and is using a org-level secret,
	// we need to find the organization ID for the canvas where the resource is being created.
	//
	canvas, err := models.FindCanvasByID(domainIdForResource.String())
	if err != nil {
		return "", nil, fmt.Errorf("canvas not found")
	}

	return models.DomainTypeOrganization, &canvas.OrganizationID, nil
}
