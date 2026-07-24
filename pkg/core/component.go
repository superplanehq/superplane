package core

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/oidc"
)

var (
	ErrSecretKeyNotFound   = errors.New("secret or key not found")
	ErrExecutionKVNotFound = errors.New("execution kv not found")
	ErrQueueItemDeferred   = errors.New("queue item deferred")
)

/*
 * ExecutionContext allows the component
 * to control the state and metadata of each execution of it.
 */
type ExecutionContext struct {
	ID             uuid.UUID
	WorkflowID     string
	OrganizationID string
	CanvasName     string
	NodeID         string
	NodeName       string
	SourceNodeID   string
	BaseURL        string
	Data           any
	Configuration  any
	Logger         *log.Entry
	HTTP           HTTPContext
	Metadata       MetadataWriter
	NodeMetadata   MetadataReader
	ExecutionState ExecutionStateContext
	Requests       RequestContext
	Auth           AuthReader
	Integration    IntegrationContext
	Secrets        SecretsContext
	CanvasMemory   CanvasMemoryContext
	Files          RepositoryFilesContext
	Webhook        NodeWebhookContext
	Expressions    ExpressionContext
	OIDC           oidc.Provider
	Apps           AppExecutionContext
	Runs           RunExecutionContext
}

type AppExecutionContext interface {
	Broadcast(payload any) error
}

type ExpressionContext interface {
	Run(expression string) (any, error)
	RunWithExtraVariables(expression string, variables map[string]any) (any, error)
}

/*
 * Components / triggers / applications should always
 * use this context instead of the net/http directly for executing HTTP requests.
 *
 * This makes it easy for us to write unit tests for the implementations,
 * and also makes it easier to control HTTP timeouts for everything in one place.
 */
type HTTPContext interface {
	Do(*http.Request) (*http.Response, error)
}

/*
 * ExecutionContext allows the component
 * to control the state and metadata of each execution of it.
 */
type SetupContext struct {
	Logger        *log.Entry
	Configuration any
	HTTP          HTTPContext
	Metadata      MetadataWriter
	Requests      RequestContext
	Auth          AuthReader
	Integration   IntegrationContext
	Webhook       NodeWebhookContext
	Files         RepositoryFilesContext
	Apps          AppContext
}

type CanvasMemoryContext interface {
	Add(namespace string, values any) error
	Find(namespace string, matches map[string]any) ([]any, error)
	FindFirst(namespace string, matches map[string]any) (any, error)
}

type RepositoryFilesContext interface {
	List() ([]string, error)
	Read(path string) (io.ReadCloser, error)
}

type CanvasMemoryRecord struct {
	ID     uuid.UUID
	Values any
}

/*
 * ExecutionStateContext allows components to control execution lifecycle.
 */
type ExecutionStateContext interface {
	IsFinished() bool
	SetKV(key, value string) error
	GetKV(key string) (string, error)

	/*
	 * Pass the execution, emitting a payload to the specified channel.
	 */
	Emit(channel, payloadType string, payloads []any) error

	/*
	 * Emit a payload but keep the execution active for further work.
	 */
	EmitAndContinue(channel, payloadType string, payloads []any) error

	/*
	 * Pass the execution, without emitting any payloads from it.
	 */
	Pass() error

	/*
	 * Fails the execution.
	 * No payloads are emitted.
	 */
	Fail(reason, message string) error
}

/*
 * RequestContext allows the execution to schedule
 * work with the processing engine.
 */
type RequestContext interface {

	//
	// Allows the scheduling of a certain component action at a later time
	//
	ScheduleActionCall(actionName string, parameters map[string]any, interval time.Duration) error
}

/*
 * ProcessQueueContext is provided to components to process a node's queue item.
 * It mirrors the data the queue worker would otherwise use to create executions.
 */
type ProcessQueueContext struct {
	WorkflowID    string
	NodeID        string
	RootEventID   string
	EventID       string
	SourceNodeID  string
	Configuration any
	Input         any
	Expressions   ExpressionContext

	//
	// Deletes the queue item
	//
	DequeueItem func() error

	//
	// Defers the queue item by moving it to the back of the node queue.
	//
	DeferQueueItem func() error

	//
	// Updates the state of the node
	//
	UpdateNodeState func(state string) error

	//
	// Creates a pending execution for this queue item.
	//
	CreateExecution func() (*ExecutionContext, error)

	//
	// Finds an execution by a key-value pair.
	// Returns an ExecutionContext.
	//
	FindExecutionByKV func(key string, value string) (*ExecutionContext, error)

	//
	// HasRunningExecutions reports whether this node currently has any
	// unfinished (running) executions.
	//
	HasRunningExecutions func() (bool, error)

	//
	// DefaultProcessing performs the default processing for the queue item.
	// Convenience method to avoid boilerplate in components that just want default behavior,
	// where an execution is created and the item is dequeued.
	//
	DefaultProcessing func() (*uuid.UUID, error)

	//
	// DistinctIncomingSources returns the distinct upstream
	// source nodes connected to this node (ignoring multiple channels from the
	// same source)
	//
	DistinctIncomingSources func() ([]Node, error)
}

type SecretsContext interface {
	GetKey(secretName, keyName string) ([]byte, error)
	GetSecretKeys(secretName string) (map[string][]byte, error)
	GetIntegrationKeys(installationName string) (map[string][]byte, error)
}

type User struct {
	ID    string `mapstructure:"id" json:"id"`
	Name  string `mapstructure:"name" json:"name"`
	Email string `mapstructure:"email" json:"email"`
}

type RoleRef struct {
	Name        string `mapstructure:"name" json:"name"`
	DisplayName string `mapstructure:"displayName" json:"displayName"`
}

type GroupRef struct {
	Name        string `mapstructure:"name" json:"name"`
	DisplayName string `mapstructure:"displayName" json:"displayName"`
}

type Node struct {
	ID string `mapstructure:"id" json:"id"`
}
