package factories

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const (
	workOrderEventTypeCreated  = "created"
	workOrderEventTypeAssigned = "assigned"
	workOrderEventTypeClosed   = "closed"
)

func serializeFactory(factory *models.Factory) *pb.Factory {
	return &pb.Factory{
		Id:          factory.ID.String(),
		Name:        factory.Name,
		Description: factory.Description,
	}
}

func serializeFactories(factories []models.Factory) []*pb.Factory {
	result := make([]*pb.Factory, len(factories))
	for i := range factories {
		result[i] = serializeFactory(&factories[i])
	}
	return result
}

func serializeIntegrationRef(integration *models.Integration) *pb.IntegrationRef {
	if integration == nil {
		return nil
	}

	id := integration.ID.String()
	name := integration.InstallationName
	return &pb.IntegrationRef{
		Id:   &id,
		Name: &name,
	}
}

func mapToProtoStruct(m map[string]any) (*structpb.Struct, error) {
	if m == nil {
		m = map[string]any{}
	}
	return structpb.NewStruct(m)
}

func serializeFactorySource(tx *gorm.DB, organizationID uuid.UUID, source *models.FactorySource) (*pb.FactorySource, error) {
	integration, err := models.FindIntegrationInTransaction(tx, organizationID, source.IntegrationID)
	if err != nil {
		return nil, err
	}

	configuration, err := mapToProtoStruct(source.Configuration.Data())
	if err != nil {
		return nil, err
	}

	return &pb.FactorySource{
		Id:            source.ID.String(),
		Name:          source.Name,
		Integration:   serializeIntegrationRef(integration),
		Configuration: configuration,
	}, nil
}

func serializeFactorySources(tx *gorm.DB, organizationID uuid.UUID, sources []models.FactorySource) ([]*pb.FactorySource, error) {
	result := make([]*pb.FactorySource, len(sources))
	for i := range sources {
		source, err := serializeFactorySource(tx, organizationID, &sources[i])
		if err != nil {
			return nil, err
		}
		result[i] = source
	}
	return result, nil
}

func serializeWorkOrder(order *models.FactoryWorkOrder) *pb.WorkOrder {
	protoOrder := &pb.WorkOrder{
		Id:          order.ID.String(),
		Title:       order.Title,
		Description: order.Description,
		State:       serializeWorkOrderState(order.State),
		Result:      serializeWorkOrderResult(order.Result),
		CreatedAt:   timestamppb.New(order.CreatedAt),
		UpdatedAt:   timestamppb.New(order.UpdatedAt),
		Assignees:   serializeWorkOrderAssignees(order.Assignees),
		Attributes:  serializeWorkOrderAttributes(order.Attributes),
	}

	if order.SourceID != nil {
		protoOrder.Source = &pb.WorkOrder_SourceRef{
			Id:   order.SourceID.String(),
			Name: order.SourceName,
			Key:  order.SourceKey,
		}
	}

	return protoOrder
}

func serializeWorkOrders(orders []models.FactoryWorkOrder) []*pb.WorkOrder {
	result := make([]*pb.WorkOrder, len(orders))
	for i := range orders {
		result[i] = serializeWorkOrder(&orders[i])
	}
	return result
}

func serializeWorkOrderState(state string) pb.WorkOrder_State {
	switch state {
	case models.FactoryWorkOrderStateOpen:
		return pb.WorkOrder_STATE_OPEN
	case models.FactoryWorkOrderStateClosed:
		return pb.WorkOrder_STATE_CLOSED
	default:
		return pb.WorkOrder_STATE_UNSPECIFIED
	}
}

func serializeWorkOrderResult(result string) pb.WorkOrder_Result {
	switch result {
	case models.FactoryWorkOrderResultCompleted:
		return pb.WorkOrder_RESULT_COMPLETED
	case models.FactoryWorkOrderResultRejected:
		return pb.WorkOrder_RESULT_REJECTED
	default:
		return pb.WorkOrder_RESULT_UNSPECIFIED
	}
}

func serializeWorkOrderAssignees(assignees []models.FactoryWorkOrderAssignee) []*pb.WorkOrder_Assignee {
	result := make([]*pb.WorkOrder_Assignee, 0, len(assignees))
	for _, assignee := range assignees {
		name := assignee.UserID.String()
		if assignee.User != nil {
			name = assignee.User.Name
		}
		result = append(result, &pb.WorkOrder_Assignee{
			Id:   assignee.UserID.String(),
			Name: name,
		})
	}
	return result
}

func serializeWorkOrderAttributes(attributes []models.FactoryWorkOrderAttribute) []*pb.WorkOrder_Attribute {
	result := make([]*pb.WorkOrder_Attribute, len(attributes))
	for i, attribute := range attributes {
		result[i] = &pb.WorkOrder_Attribute{
			Type:  serializeWorkOrderAttributeType(attribute.Type),
			Name:  attribute.Name,
			Value: attribute.Value,
		}
	}
	return result
}

func serializeWorkOrderAttributeType(attributeType string) pb.WorkOrder_Attribute_Type {
	switch attributeType {
	case "string":
		return pb.WorkOrder_Attribute_TYPE_STRING
	case "number":
		return pb.WorkOrder_Attribute_TYPE_NUMBER
	case "url":
		return pb.WorkOrder_Attribute_TYPE_URL
	default:
		return pb.WorkOrder_Attribute_TYPE_UNSPECIFIED
	}
}

func serializeWorkOrderEvent(event *models.FactoryWorkOrderEvent) (*pb.WorkOrderEvent, error) {
	content, err := mapToProtoStruct(event.Content.Data())
	if err != nil {
		return nil, err
	}

	return &pb.WorkOrderEvent{
		Id:        event.ID.String(),
		Type:      event.Type,
		Content:   content,
		CreatedAt: timestamppb.New(event.CreatedAt),
	}, nil
}

func serializeWorkOrderEvents(events []models.FactoryWorkOrderEvent) ([]*pb.WorkOrderEvent, error) {
	result := make([]*pb.WorkOrderEvent, len(events))
	for i := range events {
		event, err := serializeWorkOrderEvent(&events[i])
		if err != nil {
			return nil, err
		}
		result[i] = event
	}
	return result, nil
}

func serializeFactoryAgent(agent *models.FactoryAgent) *pb.FactoryAgent {
	spec := agent.Spec.Data()
	protoAgent := &pb.FactoryAgent{
		Id:          agent.ID.String(),
		Name:        agent.Name,
		Description: agent.Description,
		Kind:        spec.Kind,
		Model:       spec.Model,
		Machine: &pb.FactoryAgent_Machine{
			Type: spec.Machine.Type,
		},
		EnvFrom: serializeFactoryAgentEnvSources(spec.EnvFrom),
		Env:     serializeFactoryAgentEnvVars(spec.Env),
	}
	return protoAgent
}

func serializeFactoryAgents(agents []models.FactoryAgent) []*pb.FactoryAgent {
	result := make([]*pb.FactoryAgent, len(agents))
	for i := range agents {
		result[i] = serializeFactoryAgent(&agents[i])
	}
	return result
}

func serializeFactoryAgentEnvSources(sources []models.FactoryAgentEnvSource) []*pb.FactoryAgent_EnvSource {
	result := make([]*pb.FactoryAgent_EnvSource, len(sources))
	for i, source := range sources {
		protoSource := &pb.FactoryAgent_EnvSource{}
		switch source.Source {
		case "secret":
			protoSource.Source = pb.FactoryAgent_EnvSource_TYPE_SECRET
			if source.Secret != nil {
				protoSource.Secret = &pb.FactoryAgent_EnvSource_SecretRef{
					Name: source.Secret.Name,
					Key:  source.Secret.Key,
				}
			}
		case "integration":
			protoSource.Source = pb.FactoryAgent_EnvSource_TYPE_INTEGRATION
			if source.Integration != nil {
				protoSource.Integration = serializeFactoryAgentIntegrationRef(source.Integration)
			}
		}
		result[i] = protoSource
	}
	return result
}

func serializeFactoryAgentIntegrationRef(ref *models.FactoryAgentIntegrationRef) *pb.IntegrationRef {
	if ref == nil {
		return nil
	}
	protoRef := &pb.IntegrationRef{}
	if ref.ID != nil {
		protoRef.Id = ref.ID
	}
	if ref.Name != nil {
		protoRef.Name = ref.Name
	}
	return protoRef
}

func serializeFactoryAgentEnvVars(vars []models.FactoryAgentEnvVar) []*pb.FactoryAgent_EnvVar {
	result := make([]*pb.FactoryAgent_EnvVar, len(vars))
	for i, envVar := range vars {
		protoVar := &pb.FactoryAgent_EnvVar{}
		switch envVar.Source {
		case "secret":
			protoVar.Source = pb.FactoryAgent_EnvVar_SOURCE_SECRET
			if envVar.Secret != nil {
				protoVar.Secret = &pb.FactoryAgent_EnvVar_SecretKeyRef{
					Name: envVar.Secret.Name,
					Key:  envVar.Secret.Key,
				}
			}
		case "inline":
			protoVar.Source = pb.FactoryAgent_EnvVar_SOURCE_INLINE
			protoVar.Value = envVar.Value
		}
		result[i] = protoVar
	}
	return result
}

func serializeAgentAssignment(assignment *models.FactoryAgentAssignment) *pb.AgentAssignment {
	return &pb.AgentAssignment{
		Id:           assignment.ID.String(),
		AgentId:      assignment.AgentID.String(),
		OrderId:      assignment.WorkOrderID.String(),
		Instructions: assignment.Instructions,
		State:        serializeAgentAssignmentState(assignment.State),
		CreatedAt:    timestamppb.New(assignment.CreatedAt),
		UpdatedAt:    timestamppb.New(assignment.UpdatedAt),
	}
}

func serializeAgentAssignments(assignments []models.FactoryAgentAssignment) []*pb.AgentAssignment {
	result := make([]*pb.AgentAssignment, len(assignments))
	for i := range assignments {
		result[i] = serializeAgentAssignment(&assignments[i])
	}
	return result
}

func serializeAgentAssignmentState(state string) pb.AgentAssignment_State {
	switch state {
	case models.FactoryAgentAssignmentStatePending:
		return pb.AgentAssignment_STATE_PENDING
	case models.FactoryAgentAssignmentStateStarted:
		return pb.AgentAssignment_STATE_STARTED
	case models.FactoryAgentAssignmentStateCompleted:
		return pb.AgentAssignment_STATE_COMPLETED
	case models.FactoryAgentAssignmentStateFailed:
		return pb.AgentAssignment_STATE_FAILED
	default:
		return pb.AgentAssignment_STATE_UNSPECIFIED
	}
}

func assigneeIDsToStrings(assigneeIDs []uuid.UUID) []string {
	result := make([]string, len(assigneeIDs))
	for i, assigneeID := range assigneeIDs {
		result[i] = assigneeID.String()
	}
	return result
}
