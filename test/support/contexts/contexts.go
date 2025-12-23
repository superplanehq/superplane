package contexts

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
)

type EventContext struct {
	Payloads []Payload
}

type Payload struct {
	Type string
	Data any
}

func (e *EventContext) Emit(payloadType string, payload any) error {
	e.Payloads = append(e.Payloads, Payload{Type: payloadType, Data: payload})
	return nil
}

func (e *EventContext) Count() int {
	return len(e.Payloads)
}

type WebhookContext struct {
	Secret string
}

func (w *WebhookContext) GetSecret() ([]byte, error) {
	return []byte(w.Secret), nil
}

func (w *WebhookContext) Setup(options *core.WebhookSetupOptions) error {
	return nil
}

type MetadataContext struct {
	Metadata any
}

func (m *MetadataContext) Get() any {
	return m.Metadata
}

func (m *MetadataContext) Set(metadata any) error {
	m.Metadata = metadata
	return nil
}

type AppInstallationContext struct {
	Metadata         any
	State            string
	StateDescription string
	BrowserAction    *core.BrowserAction
	Secrets          map[string]core.InstallationSecret
	WebhookRequests  []any
}

func (c *AppInstallationContext) ID() uuid.UUID {
	return uuid.New()
}

func (c *AppInstallationContext) GetMetadata() any {
	return c.Metadata
}

func (c *AppInstallationContext) SetMetadata(metadata any) {
	c.Metadata = metadata
}

func (c *AppInstallationContext) GetConfig(name string) ([]byte, error) {
	return nil, nil
}

func (c *AppInstallationContext) GetState() string {
	return ""
}

func (c *AppInstallationContext) SetState(state, stateDescription string) {
	c.State = state
	c.StateDescription = stateDescription
}

func (c *AppInstallationContext) NewBrowserAction(action core.BrowserAction) {
	c.BrowserAction = &action
}

func (c *AppInstallationContext) RemoveBrowserAction() {
	c.BrowserAction = nil
}

func (c *AppInstallationContext) SetSecret(name string, value []byte) error {
	c.Secrets[name] = core.InstallationSecret{Name: name, Value: value}
	return nil
}

func (c *AppInstallationContext) GetSecrets() ([]core.InstallationSecret, error) {
	secrets := make([]core.InstallationSecret, 0, len(c.Secrets))
	for _, secret := range c.Secrets {
		secrets = append(secrets, secret)
	}
	return secrets, nil
}

func (c *AppInstallationContext) RequestWebhook(configuration any) error {
	c.WebhookRequests = append(c.WebhookRequests, configuration)
	return nil
}

type ExecutionStateContext struct {
	Finished       bool
	Passed         bool
	FailureReason  string
	FailureMessage string
	Channel        string
	Type           string
	Payloads       []any
	KVs            map[string]string
}

func (c *ExecutionStateContext) IsFinished() bool {
	return c.Finished
}

func (c *ExecutionStateContext) Pass() error {
	c.Finished = true
	c.Passed = true
	return nil
}

func (c *ExecutionStateContext) Emit(channel, payloadType string, payloads []any) error {
	c.Finished = true
	c.Passed = true
	c.Channel = channel
	c.Type = payloadType
	c.Payloads = payloads
	return nil
}

func (c *ExecutionStateContext) Fail(reason, message string) error {
	c.Finished = true
	c.Passed = false
	c.FailureReason = reason
	c.FailureMessage = message
	return nil
}

func (c *ExecutionStateContext) SetKV(key, value string) error {
	c.KVs[key] = value
	return nil
}

type AuthContext struct {
	User  *core.User
	Users map[string]*core.User
}

func (c *AuthContext) AuthenticatedUser() *core.User {
	return c.User
}

func (c *AuthContext) GetUser(id uuid.UUID) (*core.User, error) {
	if c.Users != nil {
		if user, ok := c.Users[id.String()]; ok {
			return user, nil
		}
	}

	return nil, fmt.Errorf("user not found: %s", id.String())
}

func (c *AuthContext) HasRole(role string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (c *AuthContext) InGroup(group string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

type RequestContext struct {
	Duration time.Duration
	Action   string
	Params   map[string]any
}

func (c *RequestContext) ScheduleActionCall(action string, params map[string]any, duration time.Duration) error {
	c.Action = action
	c.Params = params
	c.Duration = duration
	return nil
}
