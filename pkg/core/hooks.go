package core

import (
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
)

type HookType string

const (
	HookTypeInternal HookType = "internal"
	HookTypeUser     HookType = "user"
)

type Hook struct {
	Type       HookType
	Name       string
	Parameters []configuration.Field
}

/*
 * Context for executing a action hook.
 */
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
 * Context for executing a trigger hook.
 */
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
 * Context for executing a integration hook.
 */
type IntegrationHookContext struct {
	Name            string
	Parameters      any
	Configuration   any
	WebhooksBaseURL string
	Logger          *log.Entry
	Requests        RequestContext
	Integration     IntegrationContext
	HTTP            HTTPContext
}
