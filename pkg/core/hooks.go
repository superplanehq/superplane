package core

import (
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
)

type HookType string

const (
	HookTypeInternal = "internal"
	HookTypeUser     = "user"
)

type Hook struct {
	Type       HookType
	Name       string
	Parameters []configuration.Field
}

/*
 * Hook provider for Action implementations.
 */
type ActionHookProvider interface {
	Hooks() []Hook
	HandleHook(ActionHookContext) error
}

type ActionHookContext struct {
	Name           string
	Configuration  any
	Parameters     map[string]any
	Logger         *log.Entry
	HTTP           HTTPContext
	Metadata       MetadataWriter
	ExecutionState ExecutionStateContext
	Auth           AuthReader
	Requests       RequestContext
	Integration    IntegrationContext
	Notifications  NotificationContext
	Secrets        SecretsContext
}

/*
 * Hook provider for Trigger implementations.
 */
type TriggerHookProvider interface {
	Hooks() []Hook

	// TODO: returning map[string]any here is a bad idea
	HandleHook(TriggerHookContext) (map[string]any, error)
}

type TriggerHookContext struct {
	Name          string
	Parameters    map[string]any
	Configuration any
	Logger        *log.Entry
	HTTP          HTTPContext
	Metadata      MetadataWriter
	Requests      RequestContext
	Events        EventContext
	Webhook       NodeWebhookContext
	Integration   IntegrationContext
}

/*
 * Hook provider for Integration implementations.
 */
type IntegrationHookProvider interface {
	Hooks() []Hook
	HandleHook(ctx IntegrationHookContext) error
}

type IntegrationHookContext struct {
	Name            string
	Parameters      any
	Configuration   any
	WebhooksBaseURL string
	Logger          *logrus.Entry
	Requests        RequestContext
	Integration     IntegrationContext
	HTTP            HTTPContext
}
