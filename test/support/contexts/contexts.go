package contexts

import (
	"fmt"
	"net/http"
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

func (w *WebhookContext) ResetSecret() ([]byte, []byte, error) {
	return []byte(w.Secret), []byte(w.Secret), nil
}

func (w *WebhookContext) SetSecret(secret []byte) error {
	w.Secret = string(secret)
	return nil
}

func (w *WebhookContext) Setup() (string, error) {
	id := uuid.New()
	return id.String(), nil
}

func (w *WebhookContext) GetBaseURL() string {
	return "http://localhost:3000/api/v1"
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

type IntegrationContext struct {
	IntegrationID    string
	Configuration    map[string]any
	Metadata         any
	State            string
	StateDescription string
	BrowserAction    *core.BrowserAction
	Secrets          map[string]core.IntegrationSecret
	WebhookRequests  []any
	ResyncRequests   []time.Duration
	ActionRequests   []ActionRequest
	Subscriptions    []Subscription
}

type ActionRequest struct {
	ActionName string
	Parameters any
	Interval   time.Duration
}

type Subscription struct {
	ID            uuid.UUID
	Configuration any
}

func (c *IntegrationContext) ID() uuid.UUID {
	if c.IntegrationID != "" {
		return uuid.MustParse(c.IntegrationID)
	}

	return uuid.New()
}

func (c *IntegrationContext) GetMetadata() any {
	return c.Metadata
}

func (c *IntegrationContext) SetMetadata(metadata any) {
	c.Metadata = metadata
}

func (c *IntegrationContext) GetConfig(name string) ([]byte, error) {
	if c.Configuration == nil {
		return nil, fmt.Errorf("config not found: %s", name)
	}

	value, ok := c.Configuration[name]
	if !ok {
		return nil, fmt.Errorf("config not found: %s", name)
	}

	s, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("config is not a string: %s", name)
	}

	return []byte(s), nil
}

func (c *IntegrationContext) GetState() string {
	return ""
}

func (c *IntegrationContext) Ready() {
	c.State = "ready"
	c.StateDescription = ""
}

func (c *IntegrationContext) Error(message string) {
	c.State = "error"
	c.StateDescription = message
}

func (c *IntegrationContext) NewBrowserAction(action core.BrowserAction) {
	c.BrowserAction = &action
}

func (c *IntegrationContext) RemoveBrowserAction() {
	c.BrowserAction = nil
}

func (c *IntegrationContext) SetSecret(name string, value []byte) error {
	c.Secrets[name] = core.IntegrationSecret{Name: name, Value: value}
	return nil
}

func (c *IntegrationContext) GetSecrets() ([]core.IntegrationSecret, error) {
	secrets := make([]core.IntegrationSecret, 0, len(c.Secrets))
	for _, secret := range c.Secrets {
		secrets = append(secrets, secret)
	}
	return secrets, nil
}

func (c *IntegrationContext) RequestWebhook(configuration any) error {
	c.WebhookRequests = append(c.WebhookRequests, configuration)
	return nil
}

func (c *IntegrationContext) ScheduleResync(interval time.Duration) error {
	c.ResyncRequests = append(c.ResyncRequests, interval)
	return nil
}

func (c *IntegrationContext) ScheduleActionCall(actionName string, parameters any, interval time.Duration) error {
	c.ActionRequests = append(c.ActionRequests, ActionRequest{ActionName: actionName, Parameters: parameters, Interval: interval})
	return nil
}

func (c *IntegrationContext) ListSubscriptions() ([]core.IntegrationSubscriptionContext, error) {
	return nil, nil
}

func (c *IntegrationContext) FindSubscription(predicate func(core.IntegrationSubscriptionContext) bool) (core.IntegrationSubscriptionContext, error) {
	return nil, nil
}

func (c *IntegrationContext) Subscribe(subscription any) (*uuid.UUID, error) {
	s := Subscription{ID: uuid.New(), Configuration: subscription}
	c.Subscriptions = append(c.Subscriptions, s)
	return &s.ID, nil
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

	// Wrap payloads like the real ExecutionStateContext does
	wrappedPayloads := make([]any, 0, len(payloads))
	for _, payload := range payloads {
		wrappedPayloads = append(wrappedPayloads, map[string]any{
			"type":      payloadType,
			"timestamp": time.Now(),
			"data":      payload,
		})
	}
	c.Payloads = wrappedPayloads
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
	User   *core.User
	Users  map[string]*core.User
	Roles  map[string]struct{}
	Groups map[string]struct{}
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
	if c.User == nil {
		return false, fmt.Errorf("user not authenticated")
	}

	if c.Roles == nil {
		return false, nil
	}

	_, ok := c.Roles[role]
	return ok, nil
}

func (c *AuthContext) InGroup(group string) (bool, error) {
	if c.User == nil {
		return false, fmt.Errorf("user not authenticated")
	}

	if c.Groups == nil {
		return false, nil
	}

	_, ok := c.Groups[group]
	return ok, nil
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

type HTTPContext struct {
	Requests  []*http.Request
	Responses []*http.Response
}

func (c *HTTPContext) Do(request *http.Request) (*http.Response, error) {
	c.Requests = append(c.Requests, request)

	if len(c.Responses) == 0 {
		return nil, fmt.Errorf("no response mocked")
	}

	response := c.Responses[0]
	c.Responses = c.Responses[1:]
	return response, nil
}
