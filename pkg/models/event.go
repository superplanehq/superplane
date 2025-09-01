package models

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	expr "github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/vm"
	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	EventStatePending = "pending"

	//
	// Event ends up in this state when:
	// - Event does not pass event source filters
	// - Source is not connected to anything
	// - Source is connected, but filters on all the connections reject it.
	//
	EventStateDiscarded = "discarded"

	//
	// Event ends up in this state when:
	// - Source is connected and not rejected by some connection filters.
	//
	EventStateProcessed = "processed"

	SourceTypeEventSource     = "event-source"
	SourceTypeStage           = "stage"
	SourceTypeConnectionGroup = "connection-group"

	ConnectionTargetTypeStage           = "stage"
	ConnectionTargetTypeConnectionGroup = "connection-group"
)

type Event struct {
	ID         uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	SourceID   uuid.UUID
	CanvasID   uuid.UUID
	SourceName string
	SourceType string
	Type       string
	State      string
	ReceivedAt *time.Time
	Raw        datatypes.JSON
	Headers    datatypes.JSON
}

type headerVisitor struct{}

type SourceFilter struct {
	SourceType string
	SourceID   string
}

// Visit implements the visitor pattern for header variables.
// Update header map keys to be case insensitive.
func (v *headerVisitor) Visit(node *ast.Node) {
	if memberNode, ok := (*node).(*ast.MemberNode); ok {
		memberName := strings.ToLower(memberNode.Node.String())
		if stringNode, ok := memberNode.Property.(*ast.StringNode); ok {
			stringNode.Value = strings.ToLower(stringNode.Value)
		}

		if memberName == "headers" {
			ast.Patch(node, &ast.MemberNode{
				Node:     &ast.IdentifierNode{Value: memberName},
				Property: memberNode.Property,
				Optional: false,
				Method:   false,
			})
		}
	}
}

func (e *Event) UpdateState(state string) error {
	return e.UpdateStateInTransaction(database.Conn(), state)
}

func (e *Event) UpdateStateInTransaction(tx *gorm.DB, state string) error {
	return tx.Model(e).
		Clauses(clause.Returning{}).
		Update("state", state).
		Error
}

func (e *Event) GetData() (map[string]any, error) {
	var obj map[string]any
	err := json.Unmarshal(e.Raw, &obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (e *Event) GetHeaders() (map[string]any, error) {
	var obj map[string]any
	err := json.Unmarshal(e.Headers, &obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (e *Event) EvaluateBoolExpression(expression string, filterType string) (bool, error) {
	//
	// We don't want the expression to run for more than 5 seconds.
	//
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//
	// Build our variable map.
	//
	variables, err := parseExpressionVariables(ctx, e, filterType)
	if err != nil {
		return false, fmt.Errorf("error parsing expression variables: %v", err)
	}

	//
	// Compile and run our expression.
	//
	program, err := CompileBooleanExpression(variables, expression, filterType)

	if err != nil {
		return false, fmt.Errorf("error compiling expression: %v", err)
	}

	output, err := expr.Run(program, variables)
	if err != nil {
		return false, fmt.Errorf("error running expression: %v", err)
	}

	//
	// Output of the expression must be a boolean.
	//
	v, ok := output.(bool)
	if !ok {
		return false, fmt.Errorf("expression does not return a boolean")
	}

	return v, nil
}

func (e *Event) EvaluateStringExpression(expression string) (string, error) {
	//
	// We don't want the expression to run for more than 5 seconds.
	//
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//
	// Build our variable map.
	//
	variables := map[string]interface{}{
		"ctx": ctx,
	}

	data, err := e.GetData()
	if err != nil {
		return "", err
	}

	variables["$"] = data

	//
	// Compile and run our expression.
	//
	program, err := expr.Compile(expression,
		expr.Env(variables),
		expr.AsKind(reflect.String),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),
	)

	if err != nil {
		return "", fmt.Errorf("error compiling expression: %v", err)
	}

	output, err := expr.Run(program, variables)
	if err != nil {
		return "", fmt.Errorf("error running expression: %v", err)
	}

	//
	// Output of the expression must be a string.
	//
	v, ok := output.(string)
	if !ok {
		return "", fmt.Errorf("expression does not return a string")
	}

	return v, nil
}

func CreateEvent(sourceID uuid.UUID, canvasID uuid.UUID, sourceName, sourceType string, eventType string, raw []byte, headers []byte) (*Event, error) {
	return CreateEventInTransaction(database.Conn(), sourceID, canvasID, sourceName, sourceType, eventType, raw, headers)
}

func CreateEventInTransaction(tx *gorm.DB, sourceID uuid.UUID, canvasID uuid.UUID, sourceName, sourceType string, eventType string, raw []byte, headers []byte) (*Event, error) {
	now := time.Now()

	event := Event{
		SourceID:   sourceID,
		CanvasID:   canvasID,
		SourceName: sourceName,
		SourceType: sourceType,
		Type:       eventType,
		State:      EventStatePending,
		ReceivedAt: &now,
		Raw:        datatypes.JSON(raw),
		Headers:    datatypes.JSON(headers),
	}

	err := tx.
		Clauses(clause.Returning{}).
		Create(&event).
		Error

	if err != nil {
		return nil, err
	}

	return &event, nil
}

func ListEventsBySourceID(sourceID uuid.UUID) ([]Event, error) {
	var events []Event
	return events, database.Conn().Where("source_id = ?", sourceID).Find(&events).Error
}

func ListPendingEvents() ([]Event, error) {
	var events []Event
	return events, database.Conn().Where("state = ?", EventStatePending).Find(&events).Error
}

func FindEventByID(id uuid.UUID) (*Event, error) {
	var event Event
	return &event, database.Conn().Where("id = ?", id).First(&event).Error
}

func FindLastEventBySourceID(sourceID uuid.UUID) (map[string]any, error) {
	var event Event
	err := database.Conn().
		Table("events").
		Select("raw").
		Where("source_id = ?", sourceID).
		Order("received_at DESC").
		First(&event).
		Error

	if err != nil {
		return nil, fmt.Errorf("error finding event: %v", err)
	}

	var m map[string]any
	err = json.Unmarshal(event.Raw, &m)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling data: %v", err)
	}

	return m, nil
}

func ListEventsByCanvasID(canvasID uuid.UUID, sourceType string, sourceIDStr string) ([]Event, error) {
	var events []Event

	query := database.Conn().Where("canvas_id = ?", canvasID)

	if sourceType != "" {
		query = query.Where("source_type = ?", sourceType)
	}

	if sourceIDStr != "" {
		sourceID, err := uuid.Parse(sourceIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid source ID: %v", err)
		}
		query = query.Where("source_id = ?", sourceID)
	}

	query = query.Order("received_at DESC")

	err := query.Find(&events).Error
	if err != nil {
		return nil, err
	}

	return events, nil
}

func BulkListEventsByCanvasIDAndSource(canvasID uuid.UUID, sourceType string, sourceIDStr string, limit int) ([]Event, error) {
	var events []Event

	query := database.Conn().Where("canvas_id = ?", canvasID)

	if sourceType != "" {
		query = query.Where("source_type = ?", sourceType)
	}

	if sourceIDStr != "" {
		sourceID, err := uuid.Parse(sourceIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid source ID: %v", err)
		}
		query = query.Where("source_id = ?", sourceID)
	}

	query = query.Order("received_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&events).Error
	if err != nil {
		return nil, err
	}

	return events, nil
}

func BulkListEventsByCanvasIDAndMultipleSources(canvasID uuid.UUID, sourceFilters []SourceFilter, limitPerSource int) (map[string][]Event, error) {
	if len(sourceFilters) == 0 {
		return map[string][]Event{}, nil
	}

	var events []Event
	query := database.Conn().Where("canvas_id = ?", canvasID)

	var conditions []string
	var args []interface{}

	for _, filter := range sourceFilters {
		if filter.SourceID != "" {
			sourceID, err := uuid.Parse(filter.SourceID)
			if err != nil {
				return nil, fmt.Errorf("invalid source ID %s: %v", filter.SourceID, err)
			}
			condition := "(source_type = ? AND source_id = ?)"
			conditions = append(conditions, condition)
			args = append(args, filter.SourceType, sourceID)
		} else {
			condition := "(source_type = ?)"
			conditions = append(conditions, condition)
			args = append(args, filter.SourceType)
		}
	}

	if len(conditions) > 0 {
		whereClause := strings.Join(conditions, " OR ")
		query = query.Where(whereClause, args...)
	}

	query = query.Order("source_id, source_type, received_at DESC")

	err := query.Find(&events).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string][]Event)
	sourceCounters := make(map[string]int)

	for _, event := range events {
		sourceKey := fmt.Sprintf("%s|%s", event.SourceType, event.SourceID.String())

		if limitPerSource > 0 {
			if sourceCounters[sourceKey] >= limitPerSource {
				continue
			}
			sourceCounters[sourceKey]++
		}

		result[sourceKey] = append(result[sourceKey], event)
	}

	return result, nil
}

// CompileBooleanExpression compiles a boolean expression.
//
// variables: the variables to be used in the expression.
// expression: the expression to be compiled.
// filterType: the type of the filter.
func CompileBooleanExpression(variables map[string]any, expression string, filterType string) (*vm.Program, error) {
	options := []expr.Option{
		expr.Env(variables),
		expr.AsBool(),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),
	}

	if filterType == FilterTypeHeader {
		options = append(options, expr.Patch(&headerVisitor{}))
	}

	return expr.Compile(expression, options...)
}

func parseExpressionVariables(ctx context.Context, e *Event, filterType string) (map[string]interface{}, error) {
	variables := map[string]interface{}{
		"ctx": ctx,
	}

	var content map[string]any
	headers := map[string]any{}
	payload := map[string]any{}
	var err error

	switch filterType {
	case FilterTypeData:
		content, err = e.GetData()
		if err != nil {
			return nil, err
		}

	case FilterTypeHeader:
		content, err = e.GetHeaders()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid filter type: %s", filterType)
	}

	for key, value := range content {
		if filterType == FilterTypeHeader {
			key = strings.ToLower(key)
			headers[key] = value
		} else {
			payload[key] = value
		}
	}

	variables["$"] = payload
	variables["headers"] = headers

	return variables, nil
}
