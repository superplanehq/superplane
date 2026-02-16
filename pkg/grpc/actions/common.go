package actions

import (
	"encoding/json"
	"slices"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	configpb "github.com/superplanehq/superplane/pkg/protos/configuration"
	triggerpb "github.com/superplanehq/superplane/pkg/protos/triggers"
	widgetpb "github.com/superplanehq/superplane/pkg/protos/widgets"
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

func DomainTypeToProto(domainType string) pbAuth.DomainType {
	switch domainType {
	case models.DomainTypeOrganization:
		return pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION
	default:
		return pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED
	}
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

func stringTypeOptionsToProto(opts *configuration.StringTypeOptions) *configpb.StringTypeOptions {
	if opts == nil {
		return nil
	}

	pbOpts := &configpb.StringTypeOptions{}
	if opts.MinLength != nil {
		minLength := int32(*opts.MinLength)
		pbOpts.MinLength = &minLength
	}
	if opts.MaxLength != nil {
		maxLength := int32(*opts.MaxLength)
		pbOpts.MaxLength = &maxLength
	}

	return pbOpts
}

func expressionTypeOptionsToProto(opts *configuration.ExpressionTypeOptions) *configpb.ExpressionTypeOptions {
	if opts == nil {
		return nil
	}

	pbOpts := &configpb.ExpressionTypeOptions{}
	if opts.MinLength != nil {
		minLength := int32(*opts.MinLength)
		pbOpts.MinLength = &minLength
	}
	if opts.MaxLength != nil {
		maxLength := int32(*opts.MaxLength)
		pbOpts.MaxLength = &maxLength
	}

	return pbOpts
}

func textTypeOptionsToProto(opts *configuration.TextTypeOptions) *configpb.TextTypeOptions {
	if opts == nil {
		return nil
	}

	pbOpts := &configpb.TextTypeOptions{}
	if opts.MinLength != nil {
		minLength := int32(*opts.MinLength)
		pbOpts.MinLength = &minLength
	}
	if opts.MaxLength != nil {
		maxLength := int32(*opts.MaxLength)
		pbOpts.MaxLength = &maxLength
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

func resourceTypeOptionsToProto(opts *configuration.ResourceTypeOptions) *configpb.ResourceTypeOptions {
	if opts == nil {
		return nil
	}

	return &configpb.ResourceTypeOptions{
		Type:           opts.Type,
		UseNameAsValue: opts.UseNameAsValue,
		Multi:          opts.Multi,
		Parameters:     parameterRefsToProto(opts.Parameters),
	}
}

func listTypeOptionsToProto(opts *configuration.ListTypeOptions) *configpb.ListTypeOptions {
	if opts == nil || opts.ItemDefinition == nil {
		return nil
	}

	pbOpts := &configpb.ListTypeOptions{
		ItemLabel: opts.ItemLabel,
		ItemDefinition: &configpb.ListItemDefinition{
			Type: opts.ItemDefinition.Type,
		},
	}

	if opts.MaxItems != nil {
		maxItems := int32(*opts.MaxItems)
		pbOpts.MaxItems = &maxItems
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

func anyPredicateListTypeOptionsToProto(opts *configuration.AnyPredicateListTypeOptions) *configpb.AnyPredicateListTypeOptions {
	if opts == nil {
		return nil
	}

	pbOpts := &configpb.AnyPredicateListTypeOptions{
		Operators: make([]*configpb.SelectOption, len(opts.Operators)),
	}
	for i, opt := range opts.Operators {
		pbOpts.Operators[i] = &configpb.SelectOption{
			Label: opt.Label,
			Value: opt.Value,
		}
	}
	return pbOpts
}

func typeOptionsToProto(opts *configuration.TypeOptions) *configpb.TypeOptions {
	if opts == nil {
		return nil
	}

	return &configpb.TypeOptions{
		Number:           numberTypeOptionsToProto(opts.Number),
		String_:          stringTypeOptionsToProto(opts.String),
		Expression:       expressionTypeOptionsToProto(opts.Expression),
		Text:             textTypeOptionsToProto(opts.Text),
		Select:           selectTypeOptionsToProto(opts.Select),
		MultiSelect:      multiSelectTypeOptionsToProto(opts.MultiSelect),
		Resource:         resourceTypeOptionsToProto(opts.Resource),
		List:             listTypeOptionsToProto(opts.List),
		AnyPredicateList: anyPredicateListTypeOptionsToProto(opts.AnyPredicateList),
		Object:           objectTypeOptionsToProto(opts.Object),
		Time:             timeTypeOptionsToProto(opts.Time),
		Date:             dateTypeOptionsToProto(opts.Date),
		Datetime:         dateTimeTypeOptionsToProto(opts.DateTime),
	}
}

func ConfigurationFieldToProto(field configuration.Field) *configpb.Field {
	pbField := &configpb.Field{
		Name:               field.Name,
		Label:              field.Label,
		Type:               field.Type,
		Description:        field.Description,
		Required:           field.Required,
		Sensitive:          &field.Sensitive,
		Togglable:          &field.Togglable,
		DisallowExpression: &field.DisallowExpression,
		TypeOptions:        typeOptionsToProto(field.TypeOptions),
	}

	if field.Default != nil {
		pbField.DefaultValue = defaultValueToProto(field.Default)
	}

	if field.Placeholder != "" {
		pbField.Placeholder = &field.Placeholder
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

func protoToStringTypeOptions(pbOpts *configpb.StringTypeOptions) *configuration.StringTypeOptions {
	if pbOpts == nil {
		return nil
	}

	opts := &configuration.StringTypeOptions{}
	if pbOpts.MinLength != nil {
		minLength := int(*pbOpts.MinLength)
		opts.MinLength = &minLength
	}
	if pbOpts.MaxLength != nil {
		maxLength := int(*pbOpts.MaxLength)
		opts.MaxLength = &maxLength
	}

	return opts
}

func protoToExpressionTypeOptions(pbOpts *configpb.ExpressionTypeOptions) *configuration.ExpressionTypeOptions {
	if pbOpts == nil {
		return nil
	}

	opts := &configuration.ExpressionTypeOptions{}
	if pbOpts.MinLength != nil {
		minLength := int(*pbOpts.MinLength)
		opts.MinLength = &minLength
	}
	if pbOpts.MaxLength != nil {
		maxLength := int(*pbOpts.MaxLength)
		opts.MaxLength = &maxLength
	}
	return opts
}

func protoToTextTypeOptions(pbOpts *configpb.TextTypeOptions) *configuration.TextTypeOptions {
	if pbOpts == nil {
		return nil
	}

	opts := &configuration.TextTypeOptions{}
	if pbOpts.MinLength != nil {
		minLength := int(*pbOpts.MinLength)
		opts.MinLength = &minLength
	}
	if pbOpts.MaxLength != nil {
		maxLength := int(*pbOpts.MaxLength)
		opts.MaxLength = &maxLength
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

func protoToResourceTypeOptions(pbOpts *configpb.ResourceTypeOptions) *configuration.ResourceTypeOptions {
	if pbOpts == nil {
		return nil
	}

	return &configuration.ResourceTypeOptions{
		Type:           pbOpts.Type,
		UseNameAsValue: pbOpts.UseNameAsValue,
		Multi:          pbOpts.Multi,
		Parameters:     protoToParameterRefs(pbOpts.Parameters),
	}
}

func parameterRefsToProto(params []configuration.ParameterRef) []*configpb.ParameterRef {
	if len(params) == 0 {
		return nil
	}

	out := make([]*configpb.ParameterRef, 0, len(params))
	for _, param := range params {
		pbParam := &configpb.ParameterRef{
			Name: param.Name,
		}
		if param.Value != nil {
			pbParam.Value = *param.Value
		}
		if param.ValueFrom != nil {
			pbParam.ValueFrom = &configpb.ParameterValueFrom{
				Field: param.ValueFrom.Field,
			}
		}
		out = append(out, pbParam)
	}
	return out
}

func protoToParameterRefs(params []*configpb.ParameterRef) []configuration.ParameterRef {
	if len(params) == 0 {
		return nil
	}

	out := make([]configuration.ParameterRef, 0, len(params))
	for _, param := range params {
		if param == nil {
			continue
		}
		ref := configuration.ParameterRef{
			Name: param.Name,
		}
		if param.Value != "" {
			value := param.Value
			ref.Value = &value
		}
		if param.ValueFrom != nil {
			ref.ValueFrom = &configuration.ParameterValueFrom{
				Field: param.ValueFrom.Field,
			}
		}
		out = append(out, ref)
	}
	return out
}

func protoToListTypeOptions(pbOpts *configpb.ListTypeOptions) *configuration.ListTypeOptions {
	if pbOpts == nil || pbOpts.ItemDefinition == nil {
		return nil
	}

	opts := &configuration.ListTypeOptions{
		ItemLabel: pbOpts.ItemLabel,
		ItemDefinition: &configuration.ListItemDefinition{
			Type: pbOpts.ItemDefinition.Type,
		},
	}

	if pbOpts.MaxItems != nil {
		maxItems := int(*pbOpts.MaxItems)
		opts.MaxItems = &maxItems
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

func protoToAnyPredicateListTypeOptions(pbOpts *configpb.AnyPredicateListTypeOptions) *configuration.AnyPredicateListTypeOptions {
	if pbOpts == nil {
		return nil
	}

	opts := &configuration.AnyPredicateListTypeOptions{
		Operators: make([]configuration.FieldOption, len(pbOpts.Operators)),
	}
	for i, pbOpt := range pbOpts.Operators {
		opts.Operators[i] = configuration.FieldOption{
			Label: pbOpt.Label,
			Value: pbOpt.Value,
		}
	}
	return opts
}

func protoToTypeOptions(pbOpts *configpb.TypeOptions) *configuration.TypeOptions {
	if pbOpts == nil {
		return nil
	}

	return &configuration.TypeOptions{
		Number:           protoToNumberTypeOptions(pbOpts.Number),
		String:           protoToStringTypeOptions(pbOpts.String_),
		Expression:       protoToExpressionTypeOptions(pbOpts.Expression),
		Text:             protoToTextTypeOptions(pbOpts.Text),
		Select:           protoToSelectTypeOptions(pbOpts.Select),
		MultiSelect:      protoToMultiSelectTypeOptions(pbOpts.MultiSelect),
		Resource:         protoToResourceTypeOptions(pbOpts.Resource),
		List:             protoToListTypeOptions(pbOpts.List),
		AnyPredicateList: protoToAnyPredicateListTypeOptions(pbOpts.AnyPredicateList),
		Object:           protoToObjectTypeOptions(pbOpts.Object),
		Time:             protoToTimeTypeOptions(pbOpts.Time),
		Date:             protoToDateTypeOptions(pbOpts.Date),
		DateTime:         protoToDateTimeTypeOptions(pbOpts.Datetime),
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
		field.Default = defaultValueFromProto(pbField.Type, *pbField.DefaultValue)
	}

	if pbField.Placeholder != nil {
		field.Placeholder = *pbField.Placeholder
	}

	if pbField.DisallowExpression != nil {
		field.DisallowExpression = *pbField.DisallowExpression
	}

	if pbField.Sensitive != nil {
		field.Sensitive = *pbField.Sensitive
	}

	if pbField.Togglable != nil {
		field.Togglable = *pbField.Togglable
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
		var integrationID *string
		if node.Integration != nil && node.Integration.Id != "" {
			integrationID = &node.Integration.Id
		}

		var errorMessage *string
		if node.ErrorMessage != "" {
			errorMessage = &node.ErrorMessage
		}

		var warningMessage *string
		if node.WarningMessage != "" {
			warningMessage = &node.WarningMessage
		}

		result[i] = models.Node{
			ID:             node.Id,
			Name:           node.Name,
			Type:           ProtoToNodeType(node.Type),
			Ref:            ProtoToNodeRef(node),
			Configuration:  node.Configuration.AsMap(),
			Position:       ProtoToPosition(node.Position),
			IsCollapsed:    node.IsCollapsed,
			IntegrationID:  integrationID,
			ErrorMessage:   errorMessage,
			WarningMessage: warningMessage,
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

		if node.Ref.Widget != nil {
			result[i].Widget = &componentpb.Node_WidgetRef{
				Name: node.Ref.Widget.Name,
			}
		}

		if node.Configuration != nil {
			result[i].Configuration, _ = structpb.NewStruct(node.Configuration)
		}

		if node.Metadata != nil {
			result[i].Metadata, _ = structpb.NewStruct(node.Metadata)
		}

		if node.IntegrationID != nil && *node.IntegrationID != "" {
			result[i].Integration = &componentpb.IntegrationRef{
				Id: *node.IntegrationID,
			}
		}

		if node.ErrorMessage != nil && *node.ErrorMessage != "" {
			result[i].ErrorMessage = *node.ErrorMessage
		}

		if node.WarningMessage != nil && *node.WarningMessage != "" {
			result[i].WarningMessage = *node.WarningMessage
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

// FindShadowedNameWarnings detects nodes with duplicate names within connected components.
// Only nodes that are connected (directly or transitively) and share the same name will be flagged.
// Returns a map of node ID -> warning message.
func FindShadowedNameWarnings(nodes []*componentpb.Node, edges []*componentpb.Edge) map[string]string {
	warnings := make(map[string]string)

	if len(nodes) == 0 {
		return warnings
	}

	// Build maps for node names and IDs
	nodeIDs := make(map[string]bool)
	nodeNameByID := make(map[string]string)

	for _, node := range nodes {
		if node.Type == componentpb.Node_TYPE_WIDGET {
			continue // Skip widgets
		}
		nodeIDs[node.Id] = true
		nodeNameByID[node.Id] = node.Name
	}

	// Find connected components using union-find
	parent := make(map[string]string)
	for id := range nodeIDs {
		parent[id] = id
	}

	var find func(x string) string
	find = func(x string) string {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}

	union := func(x, y string) {
		px, py := find(x), find(y)
		if px != py {
			parent[px] = py
		}
	}

	// Union nodes connected by edges
	for _, edge := range edges {
		if edge.SourceId != "" && edge.TargetId != "" {
			// Only union if both nodes are tracked (non-widgets)
			if nodeIDs[edge.SourceId] && nodeIDs[edge.TargetId] {
				union(edge.SourceId, edge.TargetId)
			}
		}
	}

	// Group nodes by connected component
	componentNodes := make(map[string][]string)
	for id := range nodeIDs {
		root := find(id)
		componentNodes[root] = append(componentNodes[root], id)
	}

	// Check for shadowed names within each connected component
	for _, nodeIDsInComponent := range componentNodes {
		nameToIDs := make(map[string][]string)
		for _, nodeID := range nodeIDsInComponent {
			name := nodeNameByID[nodeID]
			nameToIDs[name] = append(nameToIDs[name], nodeID)
		}

		for name, ids := range nameToIDs {
			if len(ids) > 1 {
				warningMsg := "Multiple components named \"" + name + "\""
				for _, nodeID := range ids {
					warnings[nodeID] = warningMsg
				}
			}
		}
	}

	return warnings
}

func ProtoToNodeType(nodeType componentpb.Node_Type) string {
	switch nodeType {
	case componentpb.Node_TYPE_COMPONENT:
		return models.NodeTypeComponent
	case componentpb.Node_TYPE_BLUEPRINT:
		return models.NodeTypeBlueprint
	case componentpb.Node_TYPE_TRIGGER:
		return models.NodeTypeTrigger
	case componentpb.Node_TYPE_WIDGET:
		return models.NodeTypeWidget
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
	case models.NodeTypeWidget:
		return componentpb.Node_TYPE_WIDGET
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
	case componentpb.Node_TYPE_WIDGET:
		if node.Widget != nil {
			ref.Widget = &models.WidgetRef{
				Name: node.Widget.Name,
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

func defaultValueToProto(value any) *string {
	// We are converting the "default value" of a configuration
	// field to its protobuf representation.

	// The field can have different types, and the default value
	// can be a simple scalar (string, number, boolean) or a complex
	// structure (list, object).

	// For simple scalar types, we can directly convert the default
	// value to a string and return it.

	if str, ok := value.(string); ok {
		return &str
	}

	// For complex types, we need to serialize the default value
	// to JSON so that we can reconstruct the original structure
	// when deserializing.

	defaultBytes, err := json.Marshal(value)
	if err == nil {
		str := string(defaultBytes)
		return &str
	}

	// This should never happen, as we should be able to
	// marshal any value to JSON. Panic in this case.
	panic("unable to marshal default value to JSON")
}

func defaultValueFromProto(fieldType, defaultValue string) any {
	switch fieldType {
	case configuration.FieldTypeString:
		fallthrough
	case configuration.FieldTypeExpression:
		fallthrough
	case configuration.FieldTypeText:
		fallthrough
	case configuration.FieldTypeGitRef:
		fallthrough
	case configuration.FieldTypeSelect:
		fallthrough
	case configuration.FieldTypeIntegrationResource:
		fallthrough
	case configuration.FieldTypeTime:
		fallthrough
	case configuration.FieldTypeDate:
		fallthrough
	case configuration.FieldTypeDateTime:
		fallthrough
	case configuration.FieldTypeDayInYear:
		fallthrough
	case configuration.FieldTypeUser:
		fallthrough
	case configuration.FieldTypeRole:
		fallthrough
	case configuration.FieldTypeGroup:
		// String-like types: preserve the raw string.
		return defaultValue

	case configuration.FieldTypeMultiSelect:
		fallthrough
	case configuration.FieldTypeList:
		fallthrough
	case configuration.FieldTypeAnyPredicateList:
		fallthrough
	case configuration.FieldTypeObject:
		// Complex/collection types: decode JSON into structured values.
		var v any
		if err := json.Unmarshal([]byte(defaultValue), &v); err == nil {
			return v
		}
		return defaultValue

	case configuration.FieldTypeBool:
		// Boolean: decode JSON into a bool, fall back to raw string.
		var v bool
		if err := json.Unmarshal([]byte(defaultValue), &v); err == nil {
			return v
		}
		return defaultValue

	case configuration.FieldTypeNumber:
		// Number: decode JSON into a float64, fall back to raw string.
		var v float64
		if err := json.Unmarshal([]byte(defaultValue), &v); err == nil {
			return v
		}
		return defaultValue

	default:
		return defaultValue
	}
}

func SerializeComponents(in []core.Component) []*componentpb.Component {
	out := make([]*componentpb.Component, len(in))
	for i, component := range in {
		outputChannels := component.OutputChannels(nil)
		channels := make([]*componentpb.OutputChannel, len(outputChannels))
		for j, channel := range outputChannels {
			channels[j] = &componentpb.OutputChannel{
				Name: channel.Name,
			}
		}

		configFields := component.Configuration()
		configuration := make([]*configpb.Field, len(configFields))
		for j, field := range configFields {
			configuration[j] = ConfigurationFieldToProto(field)
		}
		exampleOutput, _ := structpb.NewStruct(component.ExampleOutput())

		out[i] = &componentpb.Component{
			Name:           component.Name(),
			Label:          component.Label(),
			Description:    component.Description(),
			Icon:           component.Icon(),
			Color:          component.Color(),
			OutputChannels: channels,
			Configuration:  configuration,
			ExampleOutput:  exampleOutput,
		}
	}

	return out
}

func SerializeTriggers(in []core.Trigger) []*triggerpb.Trigger {
	out := make([]*triggerpb.Trigger, len(in))
	for i, trigger := range in {
		configFields := trigger.Configuration()
		configFields = AppendGlobalTriggerFields(configFields)
		configuration := make([]*configpb.Field, len(configFields))
		for j, field := range configFields {
			configuration[j] = ConfigurationFieldToProto(field)
		}
		exampleData, _ := structpb.NewStruct(trigger.ExampleData())

		out[i] = &triggerpb.Trigger{
			Name:          trigger.Name(),
			Label:         trigger.Label(),
			Description:   trigger.Description(),
			Icon:          trigger.Icon(),
			Color:         trigger.Color(),
			Configuration: configuration,
			ExampleData:   exampleData,
		}
	}
	return out
}

func AppendGlobalTriggerFields(fields []configuration.Field) []configuration.Field {
	if slices.ContainsFunc(fields, func(field configuration.Field) bool {
		return field.Name == "customName"
	}) {
		return fields
	}

	fields = append(fields, configuration.Field{
		Name:        "customName",
		Label:       "Run title (optional)",
		Type:        configuration.FieldTypeString,
		Togglable:   true,
		Description: "Optional run title template. Supports expressions like {{ $.data }}.",
		Placeholder: "Deploy {{ $.repository.name }} @ {{ $.head_commit.id }}",
	})

	return fields
}

func SerializeWidgets(in []core.Widget) []*widgetpb.Widget {
	out := make([]*widgetpb.Widget, len(in))
	for i, widget := range in {
		configFields := widget.Configuration()
		configuration := make([]*configpb.Field, len(configFields))
		for j, field := range configFields {
			configuration[j] = ConfigurationFieldToProto(field)
		}

		out[i] = &widgetpb.Widget{
			Name:          widget.Name(),
			Label:         widget.Label(),
			Description:   widget.Description(),
			Icon:          widget.Icon(),
			Color:         widget.Color(),
			Configuration: configuration,
		}
	}
	return out
}
