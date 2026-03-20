package sendemail

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const (
	testUserID1 = "00000000-0000-0000-0000-000000000001"
	testUserID2 = "00000000-0000-0000-0000-000000000002"
)

func newAuthWithUsers(users map[string]string) *contexts.AuthContext {
	userMap := map[string]*core.User{}
	for id, email := range users {
		userMap[id] = &core.User{ID: id, Email: email}
	}
	return &contexts.AuthContext{Users: userMap}
}

type mockNotificationContext struct {
	calls     []notificationCall
	err       error
	available bool
}

type notificationCall struct {
	title     string
	body      string
	receivers core.NotificationReceivers
}

func (m *mockNotificationContext) Send(title, body, _, _ string, receivers core.NotificationReceivers) error {
	m.calls = append(m.calls, notificationCall{
		title:     title,
		body:      body,
		receivers: receivers,
	})
	return m.err
}

func (m *mockNotificationContext) IsAvailable() bool {
	return m.available
}

func TestSendEmail_BasicProperties(t *testing.T) {
	c := &SendEmail{}

	assert.Equal(t, "sendEmail", c.Name())
	assert.Equal(t, "Send Email Notification", c.Label())
	assert.Equal(t, "mail", c.Icon())
	assert.Equal(t, "gray", c.Color())
	assert.Contains(t, c.Description(), "email")
	assert.NotEmpty(t, c.Documentation())
}

func TestSendEmail_OutputChannels(t *testing.T) {
	c := &SendEmail{}
	channels := c.OutputChannels(nil)

	assert.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}

func TestSendEmail_Configuration(t *testing.T) {
	c := &SendEmail{}
	fields := c.Configuration()

	assert.NotEmpty(t, fields)

	fieldNames := make([]string, len(fields))
	for i, f := range fields {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "recipients")
	assert.Contains(t, fieldNames, "subject")
	assert.Contains(t, fieldNames, "body")
	assert.NotContains(t, fieldNames, "url")
	assert.NotContains(t, fieldNames, "urlLabel")
	assert.NotContains(t, fieldNames, "recipientMode")
	assert.NotContains(t, fieldNames, "to")
}

func TestSendEmail_Setup(t *testing.T) {
	c := &SendEmail{}

	t.Run("valid user recipient", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "user", "user": testUserID1},
				},
				"subject": "Test",
				"body":    "Body",
			},
			Metadata: metadataCtx,
		}

		err := c.Setup(ctx)
		require.NoError(t, err)

		metadata := metadataCtx.Metadata.(OutputMetadata)
		assert.Equal(t, "Test", metadata.Subject)
	})

	t.Run("valid role recipient", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "role", "role": "admin"},
				},
				"subject": "Test",
				"body":    "Body",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := c.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("valid group recipient", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "group", "group": "engineering"},
				},
				"subject": "Test",
				"body":    "Body",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := c.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("multiple recipients of different types", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "user", "user": testUserID1},
					map[string]any{"type": "group", "group": "eng"},
					map[string]any{"type": "role", "role": "admin"},
				},
				"subject": "Test",
				"body":    "Body",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := c.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("empty recipients", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipients": []any{},
				"subject":    "Test",
				"body":       "Body",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := c.Setup(ctx)
		assert.ErrorContains(t, err, "at least one recipient")
	})

	t.Run("user recipient missing user", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "user", "user": ""},
				},
				"subject": "Test",
				"body":    "Body",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := c.Setup(ctx)
		assert.ErrorContains(t, err, "user is required")
	})

	t.Run("unknown recipient type", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "unknown"},
				},
				"subject": "Test",
				"body":    "Body",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := c.Setup(ctx)
		assert.ErrorContains(t, err, "unknown type")
	})

	t.Run("missing subject", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "user", "user": testUserID1},
				},
				"subject": "",
				"body":    "Body",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := c.Setup(ctx)
		assert.ErrorContains(t, err, "subject is required")
	})

	t.Run("missing body", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "user", "user": testUserID1},
				},
				"subject": "Test",
				"body":    "",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := c.Setup(ctx)
		assert.ErrorContains(t, err, "body is required")
	})
}

func TestSendEmail_Execute(t *testing.T) {
	c := &SendEmail{}

	t.Run("resolves user IDs to emails and sends", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		notifCtx := &mockNotificationContext{available: true}
		authCtx := newAuthWithUsers(map[string]string{
			testUserID1: "alice@example.com",
		})

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "user", "user": testUserID1},
					map[string]any{"type": "group", "group": "engineering"},
					map[string]any{"type": "role", "role": "admin"},
				},
				"subject": "Status Update",
				"body":    "Everything is running smoothly.",
			},
			ExecutionState: stateCtx,
			Notifications:  notifCtx,
			Auth:           authCtx,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, notifCtx.calls, 1)
		call := notifCtx.calls[0]
		assert.Equal(t, "Status Update", call.title)
		assert.Equal(t, "Everything is running smoothly.", call.body)
		assert.ElementsMatch(t, []string{"alice@example.com"}, call.receivers.Emails)
		assert.ElementsMatch(t, []string{"engineering"}, call.receivers.Groups)
		assert.ElementsMatch(t, []string{"admin"}, call.receivers.Roles)

		assert.True(t, stateCtx.Passed)
		assert.True(t, stateCtx.Finished)
		assert.Equal(t, "default", stateCtx.Channel)
		assert.Equal(t, PayloadType, stateCtx.Type)
	})

	t.Run("fails if user not found", func(t *testing.T) {
		authCtx := newAuthWithUsers(map[string]string{})

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "user", "user": testUserID1},
				},
				"subject": "Test",
				"body":    "Body",
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Notifications:  &mockNotificationContext{available: true},
			Auth:           authCtx,
		}

		err := c.Execute(ctx)
		assert.ErrorContains(t, err, "failed to resolve user")
	})

	t.Run("missing notification context", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "user", "user": testUserID1},
				},
				"subject": "Test",
				"body":    "Body",
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Notifications:  nil,
		}

		err := c.Execute(ctx)
		assert.ErrorContains(t, err, "notification context is not available")
	})

	t.Run("email delivery not configured", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "user", "user": testUserID1},
				},
				"subject": "Test",
				"body":    "Body",
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Notifications:  &mockNotificationContext{available: false},
		}

		err := c.Execute(ctx)
		assert.ErrorContains(t, err, "email delivery is not configured")
	})

	t.Run("missing subject", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "user", "user": testUserID1},
				},
				"subject": "",
				"body":    "Body",
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Notifications:  &mockNotificationContext{available: true},
		}

		err := c.Execute(ctx)
		assert.ErrorContains(t, err, "subject is required")
	})

	t.Run("missing body", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"recipients": []any{
					map[string]any{"type": "user", "user": testUserID1},
				},
				"subject": "Test",
				"body":    "",
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Notifications:  &mockNotificationContext{available: true},
		}

		err := c.Execute(ctx)
		assert.ErrorContains(t, err, "body is required")
	})
}

func TestBuildReceivers(t *testing.T) {
	authCtx := newAuthWithUsers(map[string]string{
		testUserID1: "alice@example.com",
		testUserID2: "bob@example.com",
	})

	t.Run("resolves user IDs to emails", func(t *testing.T) {
		config := Config{
			Recipients: []Recipient{
				{Type: RecipientTypeUser, User: testUserID1},
				{Type: RecipientTypeUser, User: testUserID2},
				{Type: RecipientTypeGroup, Group: "eng"},
				{Type: RecipientTypeRole, Role: "admin"},
			},
		}

		receivers, err := buildReceivers(config, authCtx)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"alice@example.com", "bob@example.com"}, receivers.Emails)
		assert.ElementsMatch(t, []string{"eng"}, receivers.Groups)
		assert.ElementsMatch(t, []string{"admin"}, receivers.Roles)
	})

	t.Run("deduplicates recipients", func(t *testing.T) {
		config := Config{
			Recipients: []Recipient{
				{Type: RecipientTypeUser, User: testUserID1},
				{Type: RecipientTypeUser, User: testUserID1},
				{Type: RecipientTypeGroup, Group: "eng"},
				{Type: RecipientTypeGroup, Group: "eng"},
			},
		}

		receivers, err := buildReceivers(config, authCtx)
		require.NoError(t, err)
		assert.Len(t, receivers.Emails, 1)
		assert.Len(t, receivers.Groups, 1)
	})

	t.Run("skips empty user values", func(t *testing.T) {
		config := Config{
			Recipients: []Recipient{
				{Type: RecipientTypeUser, User: ""},
				{Type: RecipientTypeGroup, Group: "eng"},
			},
		}

		receivers, err := buildReceivers(config, authCtx)
		require.NoError(t, err)
		assert.Empty(t, receivers.Emails)
		assert.ElementsMatch(t, []string{"eng"}, receivers.Groups)
	})

	t.Run("returns error for unknown user", func(t *testing.T) {
		config := Config{
			Recipients: []Recipient{
				{Type: RecipientTypeUser, User: "00000000-0000-0000-0000-000000000099"},
			},
		}

		_, err := buildReceivers(config, authCtx)
		assert.ErrorContains(t, err, "failed to resolve user")
	})
}
