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
	EventStatePending   = "pending"
	EventStateDiscarded = "discarded"
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
	SourceName string
	SourceType string
	State      string
	ReceivedAt *time.Time
	Raw        datatypes.JSON
	Headers    datatypes.JSON
	Message    string
}

type headerVisitor struct{}

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

func (e *Event) Discard() error {
	return database.Conn().Model(e).
		Update("state", EventStateDiscarded).
		Error
}

func (e *Event) MarkAsProcessed() error {
	return e.MarkAsProcessedInTransaction(database.Conn())
}

func (e *Event) MarkAsProcessedInTransaction(tx *gorm.DB) error {
	return tx.Model(e).
		Update("state", EventStateProcessed).
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

	for key, value := range data {
		variables[key] = value
	}

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

func CreateEvent(sourceID uuid.UUID, sourceName, sourceType string, raw []byte, headers []byte, message string) (*Event, error) {
	return CreateEventInTransaction(database.Conn(), sourceID, sourceName, sourceType, raw, headers, message)
}

func CreateEventInTransaction(tx *gorm.DB, sourceID uuid.UUID, sourceName, sourceType string, raw []byte, headers []byte, message string) (*Event, error) {
	now := time.Now()

	event := Event{
		SourceID:   sourceID,
		SourceName: sourceName,
		SourceType: sourceType,
		State:      EventStatePending,
		ReceivedAt: &now,
		Raw:        datatypes.JSON(raw),
		Headers:    datatypes.JSON(headers),
		Message:    message,
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
			variables[key] = value
		}
	}

	variables["headers"] = headers

	return variables, nil
}

// GenerateEventMessage generates a human-readable message for an event based on its source type and payload
func GenerateEventMessage(sourceType string, raw []byte, headers []byte) string {
	// Parse the event payload
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "Event received"
	}

	// Parse headers to determine event source type
	var headerMap map[string]interface{}
	if err := json.Unmarshal(headers, &headerMap); err != nil {
		return "Event received"
	}

	// Check headers for GitHub webhook signature
	if _, hasGitHubSig := headerMap["X-Hub-Signature-256"]; hasGitHubSig {
		return generateGitHubEventMessage(payload)
	}

	// Check headers for Semaphore webhook signature
	if _, hasSemaphoreSig := headerMap["X-Semaphore-Signature-256"]; hasSemaphoreSig {
		return generateSemaphoreEventMessage(payload)
	}

	// Check for specific event types in payload
	if eventType, ok := payload["event_type"].(string); ok {
		switch eventType {
		case "github":
			return generateGitHubEventMessage(payload)
		case "semaphore":
			return generateSemaphoreEventMessage(payload)
		}
	}

	return "Event received"
}

// generateGitHubEventMessage generates a message for GitHub webhook events
func generateGitHubEventMessage(payload map[string]interface{}) string {
	action, _ := payload["action"].(string)

	if repo, ok := payload["repository"].(map[string]interface{}); ok {
		repoName, _ := repo["name"].(string)

		if action != "" && repoName != "" {
			switch action {
			case "opened":
				if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
					if title, ok := pr["title"].(string); ok {
						return fmt.Sprintf("Pull request opened: %s in %s", title, repoName)
					}
				}
				if issue, ok := payload["issue"].(map[string]interface{}); ok {
					if title, ok := issue["title"].(string); ok {
						return fmt.Sprintf("Issue opened: %s in %s", title, repoName)
					}
				}
				return fmt.Sprintf("Pull request opened in %s", repoName)
			case "closed":
				if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
					if title, ok := pr["title"].(string); ok {
						return fmt.Sprintf("Pull request closed: %s in %s", title, repoName)
					}
				}
				return fmt.Sprintf("Pull request closed in %s", repoName)
			case "synchronize":
				if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
					if title, ok := pr["title"].(string); ok {
						return fmt.Sprintf("Pull request updated: %s in %s", title, repoName)
					}
				}
				return fmt.Sprintf("Pull request updated in %s", repoName)
			case "push":
				if ref, ok := payload["ref"].(string); ok {
					return fmt.Sprintf("Push to %s in %s", ref, repoName)
				}
				return fmt.Sprintf("Push to %s", repoName)
			}
		}

		// Handle push events
		if ref, ok := payload["ref"].(string); ok {
			if commits, ok := payload["commits"].([]interface{}); ok {
				commitCount := len(commits)
				if commitCount > 0 {
					lastCommit := commits[commitCount-1].(map[string]interface{})
					if message, ok := lastCommit["message"].(string); ok && message != "" {
						return message
					}
				}
			}
			return fmt.Sprintf("Push to %s in %s", ref, repoName)
		}

		return fmt.Sprintf("GitHub event in %s", repoName)
	}

	return "GitHub event received"
}

// generateSemaphoreEventMessage generates a message for Semaphore webhook events
func generateSemaphoreEventMessage(payload map[string]interface{}) string {
	commitMessage := extractCommitMessage(payload)
	
	if pipelineMsg := generatePipelineMessage(payload, commitMessage); pipelineMsg != "" {
		return pipelineMsg
	}
	
	if jobMsg := generateJobMessage(payload); jobMsg != "" {
		return jobMsg
	}
	
	return "Semaphore event received"
}

// extractCommitMessage extracts the commit message from the revision object
func extractCommitMessage(payload map[string]interface{}) string {
	revision, ok := payload["revision"].(map[string]interface{})
	if !ok {
		return ""
	}
	
	commitMessage, _ := revision["commit_message"].(string)
	if commitMessage == "empty" {
		return ""
	}
	
	return commitMessage
}

// generatePipelineMessage generates a message for pipeline events
func generatePipelineMessage(payload map[string]interface{}, commitMessage string) string {
	pipeline, ok := payload["pipeline"].(map[string]interface{})
	if !ok {
		return ""
	}
	
	pipelineName, _ := pipeline["name"].(string)
	if pipelineName == "" {
		return ""
	}
	
	status := getPipelineStatus(pipeline)
	if status == "" {
		return fmt.Sprintf("Pipeline %s event", pipelineName)
	}
	
	baseMessage := formatPipelineStatus(pipelineName, status)
	return addCommitContext(baseMessage, commitMessage)
}

// getPipelineStatus extracts the pipeline status, preferring result over state
func getPipelineStatus(pipeline map[string]interface{}) string {
	if result, ok := pipeline["result"].(string); ok && result != "" {
		return result
	}
	
	if state, ok := pipeline["state"].(string); ok && state != "" {
		return state
	}
	
	return ""
}

// formatPipelineStatus formats the pipeline status message
func formatPipelineStatus(pipelineName, status string) string {
	switch status {
	case "passed":
		return fmt.Sprintf("Pipeline %s passed", pipelineName)
	case "failed":
		return fmt.Sprintf("Pipeline %s failed", pipelineName)
	case "running":
		return fmt.Sprintf("Pipeline %s started", pipelineName)
	case "canceled":
		return fmt.Sprintf("Pipeline %s canceled", pipelineName)
	case "stopped":
		return fmt.Sprintf("Pipeline %s stopped", pipelineName)
	default:
		return fmt.Sprintf("Pipeline %s %s", pipelineName, status)
	}
}

// generateJobMessage generates a message for job events from blocks
func generateJobMessage(payload map[string]interface{}) string {
	blocks, ok := payload["blocks"].([]interface{})
	if !ok {
		return ""
	}
	
	for _, block := range blocks {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		
		jobs, ok := blockMap["jobs"].([]interface{})
		if !ok {
			continue
		}
		
		for _, job := range jobs {
			jobMap, ok := job.(map[string]interface{})
			if !ok {
				continue
			}
			
			jobName, _ := jobMap["name"].(string)
			result, _ := jobMap["result"].(string)
			
			if jobName != "" && result != "" {
				return formatJobStatus(jobName, result)
			}
		}
	}
	
	return ""
}

// formatJobStatus formats the job status message
func formatJobStatus(jobName, result string) string {
	switch result {
	case "passed":
		return fmt.Sprintf("Job %s passed", jobName)
	case "failed":
		return fmt.Sprintf("Job %s failed", jobName)
	case "running":
		return fmt.Sprintf("Job %s started", jobName)
	case "canceled":
		return fmt.Sprintf("Job %s canceled", jobName)
	case "stopped":
		return fmt.Sprintf("Job %s stopped", jobName)
	default:
		return fmt.Sprintf("Job %s %s", jobName, result)
	}
}

// addCommitContext adds commit message context to the base message if available
func addCommitContext(baseMessage, commitMessage string) string {
	if commitMessage != "" {
		return fmt.Sprintf("%s: %s", baseMessage, commitMessage)
	}
	return baseMessage
}
