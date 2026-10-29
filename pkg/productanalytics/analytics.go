package productanalytics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	posthog "github.com/posthog/posthog-go"
	log "github.com/sirupsen/logrus"
)

const (
	cloudEndpoint     = "https://us.i.posthog.com"
	OrganizationGroup = "organization"
	schemaVersion     = 1
)

type EventName string

const (
	EventAccountDeleted             EventName = "account_deleted"
	EventOrganizationDeleted        EventName = "organization_deleted"
	EventFactoryDeleted             EventName = "factory_deleted"
	EventPaymentMethodRemoved       EventName = "payment_method_removed"
	EventOrganizationCreated        EventName = "organization_created"
	EventFactoryOnboardingCompleted EventName = "factory_onboarding_completed"
	EventFactoryTaskCreated         EventName = "factory_task_created"
	EventFactoryTaskRefined         EventName = "factory_task_refined"
	EventFactoryTaskStarted         EventName = "factory_task_started"
	EventFactoryTaskApproved        EventName = "factory_task_approved"
	EventFactoryTaskRejected        EventName = "factory_task_rejected"
	EventFactoryTaskArchived        EventName = "factory_task_archived"
	EventFactoryTaskMerged          EventName = "factory_task_merged"
)

type ActorType string

const (
	ActorUser     ActorType = "user"
	ActorSystem   ActorType = "system"
	ActorExternal ActorType = "external"
)

type Source string

const (
	SourceUI            Source = "ui"
	SourceAPI           Source = "api"
	SourceCLI           Source = "cli"
	SourceMCP           Source = "mcp"
	SourceAutomation    Source = "automation"
	SourceGitHubWebhook Source = "github_webhook"
	SourcePolarAPI      Source = "polar_api"
)

type Properties map[string]any

type sourceContextKey struct{}

type Event struct {
	Name           EventName
	EventID        uuid.UUID
	DistinctID     string
	OrganizationID string
	FactoryID      string
	TaskID         string
	ActorType      ActorType
	Source         Source
	Properties     Properties
}

type OrganizationGroupIdentity struct {
	ID        string
	Name      string
	Slug      string
	CreatedAt *time.Time
	AccountID string
}

type Client interface {
	Capture(Event)
	IdentifyOrganizationGroup(OrganizationGroupIdentity)
	Close() error
	Enabled() bool
}

type queue interface {
	Enqueue(posthog.Message) error
	Close() error
}

type client struct {
	queue queue
}

type noopClient struct{}

var current = struct {
	sync.RWMutex
	client Client
}{client: noopClient{}}

func New(projectKey string) (Client, error) {
	projectKey = strings.TrimSpace(projectKey)
	if projectKey == "" {
		return noopClient{}, nil
	}

	posthogClient, err := posthog.NewWithConfig(projectKey, posthog.Config{
		Endpoint:        cloudEndpoint,
		ShutdownTimeout: 10 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return newClient(posthogClient), nil
}

func newClient(queue queue) Client {
	return &client{queue: queue}
}

func SetCurrent(next Client) Client {
	if next == nil {
		next = noopClient{}
	}
	current.Lock()
	previous := current.client
	current.client = next
	current.Unlock()
	return previous
}

func Current() Client {
	current.RLock()
	defer current.RUnlock()
	return current.client
}

func Capture(event Event) {
	Current().Capture(event)
}

func IdentifyOrganizationGroup(identity OrganizationGroupIdentity) {
	Current().IdentifyOrganizationGroup(identity)
}

func WithSource(ctx context.Context, source Source) context.Context {
	return context.WithValue(ctx, sourceContextKey{}, source)
}

func SourceFromContext(ctx context.Context, fallback Source) Source {
	source, ok := ctx.Value(sourceContextKey{}).(Source)
	if !ok || source == "" {
		return fallback
	}
	return source
}

func Enabled() bool {
	return Current().Enabled()
}

func CloseCurrent() error {
	previous := SetCurrent(noopClient{})
	return previous.Close()
}

func StableEventID(name EventName, parts ...string) uuid.UUID {
	value := string(name) + ":" + strings.Join(parts, ":")
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(value))
}

func SyntheticDistinctID(organizationID string, actorType ActorType, identity string) string {
	value := strings.Join([]string{organizationID, string(actorType), identity}, ":")
	digest := sha256.Sum256([]byte(value))
	return "synthetic:" + hex.EncodeToString(digest[:])
}

func (c *client) Capture(event Event) {
	if !validEvent(event) {
		log.WithField("event", event.Name).Warn("Skipped invalid product analytics event")
		return
	}

	properties := posthog.NewProperties()
	for key, value := range event.Properties {
		properties.Set(key, value)
	}
	properties.
		Set("schema_version", schemaVersion).
		Set("actor_type", string(event.ActorType)).
		Set("source", string(event.Source))
	groups := posthog.NewGroups()
	if strings.TrimSpace(event.OrganizationID) != "" {
		properties.Set("organization_id", event.OrganizationID)
		groups.Set(OrganizationGroup, event.OrganizationID)
	} else {
		delete(properties, "organization_id")
	}
	if event.FactoryID != "" {
		properties.Set("factory_id", event.FactoryID)
	}
	if event.TaskID != "" {
		properties.Set("task_id", event.TaskID)
	}
	err := c.queue.Enqueue(posthog.Capture{
		Uuid:       event.EventID.String(),
		DistinctId: event.DistinctID,
		Event:      string(event.Name),
		Properties: properties,
		Groups:     groups,
	})
	if err != nil {
		log.WithError(err).WithField("event", event.Name).Warn("Failed to enqueue product analytics event")
	}
}

func (c *client) IdentifyOrganizationGroup(identity OrganizationGroupIdentity) {
	organizationID := strings.TrimSpace(identity.ID)
	accountID := strings.TrimSpace(identity.AccountID)
	if organizationID == "" || accountID == "" {
		return
	}

	properties := posthog.NewProperties()
	if name := strings.TrimSpace(identity.Name); name != "" {
		properties.Set("name", name)
	}
	if slug := strings.TrimSpace(identity.Slug); slug != "" {
		properties.Set("slug", slug)
	}
	if identity.CreatedAt != nil && !identity.CreatedAt.IsZero() {
		properties.Set("created_at", identity.CreatedAt.UTC().Format(time.RFC3339))
	}

	err := c.queue.Enqueue(posthog.GroupIdentify{
		Type:       OrganizationGroup,
		Key:        organizationID,
		DistinctId: accountID,
		Properties: properties,
	})
	if err != nil {
		log.WithError(err).WithField("organization_id", organizationID).Warn("Failed to enqueue organization group identify")
	}
}

func (c *client) Close() error {
	return c.queue.Close()
}

func (c *client) Enabled() bool {
	return true
}

func (noopClient) Capture(Event) {}

func (noopClient) IdentifyOrganizationGroup(OrganizationGroupIdentity) {}

func (noopClient) Close() error {
	return nil
}

func (noopClient) Enabled() bool {
	return false
}

func validEvent(event Event) bool {
	return knownEvent(event.Name) &&
		event.EventID != uuid.Nil &&
		strings.TrimSpace(event.DistinctID) != "" &&
		(event.Name == EventAccountDeleted || strings.TrimSpace(event.OrganizationID) != "") &&
		event.ActorType != "" &&
		event.Source != ""
}

func knownEvent(name EventName) bool {
	switch name {
	case EventAccountDeleted, EventOrganizationDeleted, EventFactoryDeleted, EventPaymentMethodRemoved,
		EventOrganizationCreated,
		EventFactoryOnboardingCompleted,
		EventFactoryTaskCreated,
		EventFactoryTaskRefined,
		EventFactoryTaskStarted,
		EventFactoryTaskApproved,
		EventFactoryTaskRejected,
		EventFactoryTaskArchived,
		EventFactoryTaskMerged:
		return true
	default:
		return false
	}
}
