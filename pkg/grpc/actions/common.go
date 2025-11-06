package actions

import (
	"context"
	"encoding/json"
	"fmt"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	configpb "github.com/superplanehq/superplane/pkg/protos/configuration"
	integrationpb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func ValidateUUIDs(ids ...string) error {
	return ValidateUUIDsArray(ids)
}

func ValidateUUIDsArray(ids []string) error {
	for _, id := range ids {
		_, err := uuid.Parse(id)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid UUID: %s", id)
		}
	}

	return nil
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

func ValidateResource(ctx context.Context, registry *registry.Registry, integration *models.Integration, resourceRef *integrationpb.ResourceRef) (integrations.Resource, error) {
	if resourceRef == nil {
		return nil, status.Error(codes.InvalidArgument, "resource reference is required")
	}

	if resourceRef.Type == "" || resourceRef.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "resource type and name are required")
	}

	//
	// If resource record does not exist yet, we need to go to the integration to find it.
	//
	integrationImpl, err := registry.NewResourceManager(ctx, integration)
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
	canvas, err := models.FindUnscopedCanvasByID(domainIdForResource.String())
	if err != nil {
		return "", nil, fmt.Errorf("canvas not found")
	}

	return models.DomainTypeOrganization, &canvas.OrganizationID, nil
}

func numberTypeOptionsToProto(opts *configuration.NumberTypeOptions) *configpb.NumberTypeOptions {
	if opts == nil {
		return nil
	}

	pbOpts := &configpb.NumberTypeOptions{}
	if opts.Min != nil {
		min := int32(*opts.Min)
		pbOpts.Min = &min
	}
	if opts.Max != nil {
		max := int32(*opts.Max)
		pbOpts.Max = &max
	}
	return pbOpts
}

func selectTypeOptionsToProto(opts *configuration.SelectTypeOptions) *configpb.SelectTypeOptions {
	if opts == nil {
		return nil
	}

	pbOpts := &configpb.SelectTypeOptions{
		Options: make([]*configpb.SelectOption, len(opts.Options)),
	}
	for i, opt := range opts.Options {
		pbOpts.Options[i] = &configpb.SelectOption{
			Label: opt.Label,
			Value: opt.Value,
		}
	}
	return pbOpts
}

func multiSelectTypeOptionsToProto(opts *configuration.MultiSelectTypeOptions) *configpb.MultiSelectTypeOptions {
	if opts == nil {
		return nil
	}

	pbOpts := &configpb.MultiSelectTypeOptions{
		Options: make([]*configpb.SelectOption, len(opts.Options)),
	}
	for i, opt := range opts.Options {
		pbOpts.Options[i] = &configpb.SelectOption{
			Label: opt.Label,
			Value: opt.Value,
		}
	}
	return pbOpts
}

func integrationTypeOptionsToProto(opts *configuration.IntegrationTypeOptions) *configpb.IntegrationTypeOptions {
	if opts == nil {
		return nil
	}

	return &configpb.IntegrationTypeOptions{
		Type: opts.Type,
	}
}

func resourceTypeOptionsToProto(opts *configuration.ResourceTypeOptions) *configpb.ResourceTypeOptions {
	if opts == nil {
		return nil
	}

	return &configpb.ResourceTypeOptions{
		Type: opts.Type,
	}
}

func listTypeOptionsToProto(opts *configuration.ListTypeOptions) *configpb.ListTypeOptions {
	if opts == nil || opts.ItemDefinition == nil {
		return nil
	}

	pbOpts := &configpb.ListTypeOptions{
		ItemDefinition: &configpb.ListItemDefinition{
			Type: opts.ItemDefinition.Type,
		},
	}

	if len(opts.ItemDefinition.Schema) > 0 {
		pbOpts.ItemDefinition.Schema = make([]*configpb.Field, len(opts.ItemDefinition.Schema))
		for i, schemaField := range opts.ItemDefinition.Schema {
			pbOpts.ItemDefinition.Schema[i] = ConfigurationFieldToProto(schemaField)
		}
	}

	return pbOpts
}

func objectTypeOptionsToProto(opts *configuration.ObjectTypeOptions) *configpb.ObjectTypeOptions {
	if opts == nil || len(opts.Schema) == 0 {
		return nil
	}

	pbOpts := &configpb.ObjectTypeOptions{
		Schema: make([]*configpb.Field, len(opts.Schema)),
	}
	for i, schemaField := range opts.Schema {
		pbOpts.Schema[i] = ConfigurationFieldToProto(schemaField)
	}

	return pbOpts
}

func timeTypeOptionsToProto(opts *configuration.TimeTypeOptions) *configpb.TimeTypeOptions {
	if opts == nil {
		return nil
	}

	pbOpts := &configpb.TimeTypeOptions{}
	if opts.Format != "" {
		pbOpts.Format = &opts.Format
	}
	return pbOpts
}

func dateTypeOptionsToProto(opts *configuration.DateTypeOptions) *configpb.DateTypeOptions {
	if opts == nil {
		return nil
	}

	pbOpts := &configpb.DateTypeOptions{}
	if opts.Format != "" {
		pbOpts.Format = &opts.Format
	}
	return pbOpts
}

func dateTimeTypeOptionsToProto(opts *configuration.DateTimeTypeOptions) *configpb.DateTimeTypeOptions {
	if opts == nil {
		return nil
	}

	pbOpts := &configpb.DateTimeTypeOptions{}
	if opts.Format != "" {
		pbOpts.Format = &opts.Format
	}
	return pbOpts
}

func typeOptionsToProto(opts *configuration.TypeOptions) *configpb.TypeOptions {
	if opts == nil {
		return nil
	}

	return &configpb.TypeOptions{
		Number:      numberTypeOptionsToProto(opts.Number),
		Select:      selectTypeOptionsToProto(opts.Select),
		MultiSelect: multiSelectTypeOptionsToProto(opts.MultiSelect),
		Integration: integrationTypeOptionsToProto(opts.Integration),
		Resource:    resourceTypeOptionsToProto(opts.Resource),
		List:        listTypeOptionsToProto(opts.List),
		Object:      objectTypeOptionsToProto(opts.Object),
		Time:        timeTypeOptionsToProto(opts.Time),
		Date:        dateTypeOptionsToProto(opts.Date),
		Datetime:    dateTimeTypeOptionsToProto(opts.DateTime),
	}
}

func ConfigurationFieldToProto(field configuration.Field) *configpb.Field {
	pbField := &configpb.Field{
		Name:        field.Name,
		Label:       field.Label,
		Type:        field.Type,
		Description: field.Description,
		Required:    field.Required,
		TypeOptions: typeOptionsToProto(field.TypeOptions),
	}

	if field.Default != nil {
		defaultBytes, err := json.Marshal(field.Default)
		if err == nil {
			defaultStr := string(defaultBytes)
			pbField.DefaultValue = &defaultStr
		}
	}

	if len(field.VisibilityConditions) > 0 {
		pbField.VisibilityConditions = make([]*configpb.VisibilityCondition, len(field.VisibilityConditions))
		for i, cond := range field.VisibilityConditions {
			pbField.VisibilityConditions[i] = &configpb.VisibilityCondition{
				Field:  cond.Field,
				Values: cond.Values,
			}
		}
	}

	if len(field.RequiredConditions) > 0 {
		pbField.RequiredConditions = make([]*configpb.RequiredCondition, len(field.RequiredConditions))
		for i, cond := range field.RequiredConditions {
			pbField.RequiredConditions[i] = &configpb.RequiredCondition{
				Field:  cond.Field,
				Values: cond.Values,
			}
		}
	}

	if len(field.ValidationRules) > 0 {
		pbField.ValidationRules = make([]*configpb.ValidationRule, len(field.ValidationRules))
		for i, rule := range field.ValidationRules {
			pbField.ValidationRules[i] = &configpb.ValidationRule{
				Type:        rule.Type,
				CompareWith: rule.CompareWith,
			}
			if rule.Message != "" {
				pbField.ValidationRules[i].Message = &rule.Message
			}
		}
	}

	return pbField
}

func protoToNumberTypeOptions(pbOpts *configpb.NumberTypeOptions) *configuration.NumberTypeOptions {
	if pbOpts == nil {
		return nil
	}

	opts := &configuration.NumberTypeOptions{}
	if pbOpts.Min != nil {
		min := int(*pbOpts.Min)
		opts.Min = &min
	}
	if pbOpts.Max != nil {
		max := int(*pbOpts.Max)
		opts.Max = &max
	}
	return opts
}

func protoToSelectTypeOptions(pbOpts *configpb.SelectTypeOptions) *configuration.SelectTypeOptions {
	if pbOpts == nil {
		return nil
	}

	opts := &configuration.SelectTypeOptions{
		Options: make([]configuration.FieldOption, len(pbOpts.Options)),
	}
	for i, pbOpt := range pbOpts.Options {
		opts.Options[i] = configuration.FieldOption{
			Label: pbOpt.Label,
			Value: pbOpt.Value,
		}
	}
	return opts
}

func protoToMultiSelectTypeOptions(pbOpts *configpb.MultiSelectTypeOptions) *configuration.MultiSelectTypeOptions {
	if pbOpts == nil {
		return nil
	}

	opts := &configuration.MultiSelectTypeOptions{
		Options: make([]configuration.FieldOption, len(pbOpts.Options)),
	}
	for i, pbOpt := range pbOpts.Options {
		opts.Options[i] = configuration.FieldOption{
			Label: pbOpt.Label,
			Value: pbOpt.Value,
		}
	}
	return opts
}

func protoToIntegrationTypeOptions(pbOpts *configpb.IntegrationTypeOptions) *configuration.IntegrationTypeOptions {
	if pbOpts == nil {
		return nil
	}

	return &configuration.IntegrationTypeOptions{
		Type: pbOpts.Type,
	}
}

func protoToResourceTypeOptions(pbOpts *configpb.ResourceTypeOptions) *configuration.ResourceTypeOptions {
	if pbOpts == nil {
		return nil
	}

	return &configuration.ResourceTypeOptions{
		Type: pbOpts.Type,
	}
}

func protoToListTypeOptions(pbOpts *configpb.ListTypeOptions) *configuration.ListTypeOptions {
	if pbOpts == nil || pbOpts.ItemDefinition == nil {
		return nil
	}

	opts := &configuration.ListTypeOptions{
		ItemDefinition: &configuration.ListItemDefinition{
			Type: pbOpts.ItemDefinition.Type,
		},
	}

	if len(pbOpts.ItemDefinition.Schema) > 0 {
		opts.ItemDefinition.Schema = make([]configuration.Field, len(pbOpts.ItemDefinition.Schema))
		for i, pbSchemaField := range pbOpts.ItemDefinition.Schema {
			opts.ItemDefinition.Schema[i] = ProtoToConfigurationField(pbSchemaField)
		}
	}

	return opts
}

func protoToObjectTypeOptions(pbOpts *configpb.ObjectTypeOptions) *configuration.ObjectTypeOptions {
	if pbOpts == nil || len(pbOpts.Schema) == 0 {
		return nil
	}

	opts := &configuration.ObjectTypeOptions{
		Schema: make([]configuration.Field, len(pbOpts.Schema)),
	}
	for i, pbSchemaField := range pbOpts.Schema {
		opts.Schema[i] = ProtoToConfigurationField(pbSchemaField)
	}

	return opts
}

func protoToTimeTypeOptions(pbOpts *configpb.TimeTypeOptions) *configuration.TimeTypeOptions {
	if pbOpts == nil {
		return nil
	}

	opts := &configuration.TimeTypeOptions{}
	if pbOpts.Format != nil {
		opts.Format = *pbOpts.Format
	}
	return opts
}

func protoToDateTypeOptions(pbOpts *configpb.DateTypeOptions) *configuration.DateTypeOptions {
	if pbOpts == nil {
		return nil
	}

	opts := &configuration.DateTypeOptions{}
	if pbOpts.Format != nil {
		opts.Format = *pbOpts.Format
	}
	return opts
}

func protoToDateTimeTypeOptions(pbOpts *configpb.DateTimeTypeOptions) *configuration.DateTimeTypeOptions {
	if pbOpts == nil {
		return nil
	}

	opts := &configuration.DateTimeTypeOptions{}
	if pbOpts.Format != nil {
		opts.Format = *pbOpts.Format
	}
	return opts
}

func protoToTypeOptions(pbOpts *configpb.TypeOptions) *configuration.TypeOptions {
	if pbOpts == nil {
		return nil
	}

	return &configuration.TypeOptions{
		Number:      protoToNumberTypeOptions(pbOpts.Number),
		Select:      protoToSelectTypeOptions(pbOpts.Select),
		MultiSelect: protoToMultiSelectTypeOptions(pbOpts.MultiSelect),
		Integration: protoToIntegrationTypeOptions(pbOpts.Integration),
		Resource:    protoToResourceTypeOptions(pbOpts.Resource),
		List:        protoToListTypeOptions(pbOpts.List),
		Object:      protoToObjectTypeOptions(pbOpts.Object),
		Time:        protoToTimeTypeOptions(pbOpts.Time),
		Date:        protoToDateTypeOptions(pbOpts.Date),
		DateTime:    protoToDateTimeTypeOptions(pbOpts.Datetime),
	}
}

func ProtoToConfigurationField(pbField *configpb.Field) configuration.Field {
	field := configuration.Field{
		Name:        pbField.Name,
		Label:       pbField.Label,
		Type:        pbField.Type,
		Description: pbField.Description,
		Required:    pbField.Required,
		TypeOptions: protoToTypeOptions(pbField.TypeOptions),
	}

	if pbField.DefaultValue != nil {
		field.Default = *pbField.DefaultValue
	}

	if len(pbField.VisibilityConditions) > 0 {
		field.VisibilityConditions = make([]configuration.VisibilityCondition, len(pbField.VisibilityConditions))
		for i, pbCond := range pbField.VisibilityConditions {
			field.VisibilityConditions[i] = configuration.VisibilityCondition{
				Field:  pbCond.Field,
				Values: pbCond.Values,
			}
		}
	}

	if len(pbField.RequiredConditions) > 0 {
		field.RequiredConditions = make([]configuration.RequiredCondition, len(pbField.RequiredConditions))
		for i, pbCond := range pbField.RequiredConditions {
			field.RequiredConditions[i] = configuration.RequiredCondition{
				Field:  pbCond.Field,
				Values: pbCond.Values,
			}
		}
	}

	if len(pbField.ValidationRules) > 0 {
		field.ValidationRules = make([]configuration.ValidationRule, len(pbField.ValidationRules))
		for i, pbRule := range pbField.ValidationRules {
			field.ValidationRules[i] = configuration.ValidationRule{
				Type:        pbRule.Type,
				CompareWith: pbRule.CompareWith,
			}
			if pbRule.Message != nil {
				field.ValidationRules[i].Message = *pbRule.Message
			}
		}
	}

	return field
}

func ProtoToNodes(nodes []*componentpb.Node) []models.Node {
	result := make([]models.Node, len(nodes))
	for i, node := range nodes {
		result[i] = models.Node{
			ID:            node.Id,
			Name:          node.Name,
			Type:          ProtoToNodeType(node.Type),
			Ref:           ProtoToNodeRef(node),
			Configuration: node.Configuration.AsMap(),
			Position:      ProtoToPosition(node.Position),
			IsCollapsed:   node.IsCollapsed,
		}
	}
	return result
}

func NodesToProto(nodes []models.Node) []*componentpb.Node {
	result := make([]*componentpb.Node, len(nodes))
	for i, node := range nodes {
		result[i] = &componentpb.Node{
			Id:          node.ID,
			Name:        node.Name,
			Type:        NodeTypeToProto(node.Type),
			Position:    PositionToProto(node.Position),
			IsCollapsed: node.IsCollapsed,
		}

		if node.Ref.Component != nil {
			result[i].Component = &componentpb.Node_ComponentRef{
				Name: node.Ref.Component.Name,
			}
		}

		if node.Ref.Blueprint != nil {
			result[i].Blueprint = &componentpb.Node_BlueprintRef{
				Id: node.Ref.Blueprint.ID,
			}
		}

		if node.Ref.Trigger != nil {
			result[i].Trigger = &componentpb.Node_TriggerRef{
				Name: node.Ref.Trigger.Name,
			}
		}

		if node.Configuration != nil {
			result[i].Configuration, _ = structpb.NewStruct(node.Configuration)
		}

		if node.Metadata != nil {
			result[i].Metadata, _ = structpb.NewStruct(node.Metadata)
		}
	}

	return result
}

func ProtoToEdges(edges []*componentpb.Edge) []models.Edge {
	result := make([]models.Edge, len(edges))
	for i, edge := range edges {
		result[i] = models.Edge{
			SourceID: edge.SourceId,
			TargetID: edge.TargetId,
			Channel:  edge.Channel,
		}
	}
	return result
}

func EdgesToProto(edges []models.Edge) []*componentpb.Edge {
	result := make([]*componentpb.Edge, len(edges))
	for i, edge := range edges {
		result[i] = &componentpb.Edge{
			SourceId: edge.SourceID,
			TargetId: edge.TargetID,
			Channel:  edge.Channel,
		}
	}
	return result
}

func ProtoToNodeType(nodeType componentpb.Node_Type) string {
	switch nodeType {
	case componentpb.Node_TYPE_COMPONENT:
		return models.NodeTypeComponent
	case componentpb.Node_TYPE_BLUEPRINT:
		return models.NodeTypeBlueprint
	case componentpb.Node_TYPE_TRIGGER:
		return models.NodeTypeTrigger
	default:
		return ""
	}
}

func NodeTypeToProto(nodeType string) componentpb.Node_Type {
	switch nodeType {
	case models.NodeTypeBlueprint:
		return componentpb.Node_TYPE_BLUEPRINT
	case models.NodeTypeTrigger:
		return componentpb.Node_TYPE_TRIGGER
	default:
		return componentpb.Node_TYPE_COMPONENT
	}
}

func ProtoToNodeRef(node *componentpb.Node) models.NodeRef {
	ref := models.NodeRef{}

	switch node.Type {
	case componentpb.Node_TYPE_COMPONENT:
		if node.Component != nil {
			ref.Component = &models.ComponentRef{
				Name: node.Component.Name,
			}
		}
	case componentpb.Node_TYPE_BLUEPRINT:
		if node.Blueprint != nil {
			ref.Blueprint = &models.BlueprintRef{
				ID: node.Blueprint.Id,
			}
		}
	case componentpb.Node_TYPE_TRIGGER:
		if node.Trigger != nil {
			ref.Trigger = &models.TriggerRef{
				Name: node.Trigger.Name,
			}
		}
	}

	return ref
}

func PositionToProto(position models.Position) *componentpb.Position {
	return &componentpb.Position{
		X: int32(position.X),
		Y: int32(position.Y),
	}
}

func ProtoToPosition(position *componentpb.Position) models.Position {
	if position == nil {
		return models.Position{X: 0, Y: 0}
	}
	return models.Position{
		X: int(position.X),
		Y: int(position.Y),
	}
}

// Verify if the workflow is acyclic using
// topological sort algorithm - kahn's - to detect cycles
func CheckForCycles(nodes []*componentpb.Node, edges []*componentpb.Edge) error {

	//
	// Build adjacency list
	//
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	//
	// Initialize all nodesm and build the graph
	//
	for _, node := range nodes {
		graph[node.Id] = []string{}
		inDegree[node.Id] = 0
	}

	for _, edge := range edges {
		graph[edge.SourceId] = append(graph[edge.SourceId], edge.TargetId)
		inDegree[edge.TargetId]++
	}

	// Kahn's algorithm for topological sort
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	visited := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		visited++

		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// If we visited all nodes, the graph is acyclic
	if visited != len(nodes) {
		return status.Error(codes.InvalidArgument, "graph contains a cycle")
	}

	return nil
}
