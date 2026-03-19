package sendemail

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type mockNotificationContext struct {
	calls     []notificationCall
	err       error
	available bool
}

type notificationCall struct {
	title     string
	body      string
	url       string
	urlLabel  string
	receivers core.NotificationReceivers
}

func (m *mockNotificationContext) Send(title, body, url, urlLabel string, receivers core.NotificationReceivers) error {
	m.calls = append(m.calls, notificationCall{
		title:     title,
		body:      body,
		url:       url,
		urlLabel:  urlLabel,
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

	assert.Contains(t, fieldNames, "recipientMode")
	assert.Contains(t, fieldNames, "to")
	assert.Contains(t, fieldNames, "recipients")
	assert.Contains(t, fieldNames, "subject")
	assert.Contains(t, fieldNames, "body")
	assert.Contains(t, fieldNames, "url")
	assert.Contains(t, fieldNames, "urlLabel")
}

func TestSendEmail_Setup_EmailMode(t *testing.T) {
	c := &SendEmail{}

	t.Run("valid configuration", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipientMode": "emails",
				"to":            "alice@example.com, bob@example.com",
				"subject":       "Test Subject",
				"body":          "Test body",
			},
			Metadata: metadataCtx,
		}

		err := c.Setup(ctx)
		require.NoError(t, err)

		metadata := metadataCtx.Metadata.(OutputMetadata)
		assert.Equal(t, "Test Subject", metadata.Subject)
		assert.ElementsMatch(t, []string{"alice@example.com", "bob@example.com"}, metadata.To)
	})

	t.Run("missing to", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipientMode": "emails",
				"to":            "",
				"subject":       "Test",
				"body":          "Body",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := c.Setup(ctx)
		assert.ErrorContains(t, err, "to is required")
	})

	t.Run("invalid email", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipientMode": "emails",
				"to":            "not-an-email",
				"subject":       "Test",
				"body":          "Body",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := c.Setup(ctx)
		assert.ErrorContains(t, err, "invalid email")
	})

	t.Run("missing subject", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipientMode": "emails",
				"to":            "alice@example.com",
				"subject":       "",
				"body":          "Body",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := c.Setup(ctx)
		assert.ErrorContains(t, err, "subject is required")
	})

	t.Run("missing body", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipientMode": "emails",
				"to":            "alice@example.com",
				"subject":       "Test",
				"body":          "",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := c.Setup(ctx)
		assert.ErrorContains(t, err, "body is required")
	})
}

func TestSendEmail_Setup_MembersMode(t *testing.T) {
	c := &SendEmail{}

	t.Run("valid user recipient", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipientMode": "members",
				"recipients": []any{
					map[string]any{"type": "user", "user": "user-123"},
				},
				"subject": "Test",
				"body":    "Body",
			},
			Metadata: metadataCtx,
		}

		err := c.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("valid role recipient", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipientMode": "members",
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
				"recipientMode": "members",
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

	t.Run("empty recipients", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipientMode": "members",
				"recipients":    []any{},
				"subject":       "Test",
				"body":          "Body",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := c.Setup(ctx)
		assert.ErrorContains(t, err, "at least one recipient")
	})

	t.Run("user recipient missing user", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"recipientMode": "members",
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
				"recipientMode": "members",
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
}

func TestSendEmail_Setup_UnknownRecipientMode(t *testing.T) {
	c := &SendEmail{}
	ctx := core.SetupContext{
		Configuration: map[string]any{
			"recipientMode": "invalid",
			"subject":       "Test",
			"body":          "Body",
		},
		Metadata: &contexts.MetadataContext{},
	}

	err := c.Setup(ctx)
	assert.ErrorContains(t, err, "unknown recipient mode")
}

func TestSendEmail_Execute_EmailMode(t *testing.T) {
	c := &SendEmail{}

	t.Run("sends email to addresses", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		notifCtx := &mockNotificationContext{available: true}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"recipientMode": "emails",
				"to":            "alice@example.com, bob@example.com",
				"subject":       "Deploy Complete",
				"body":          "The deployment finished successfully.",
				"url":           "https://example.com/deploy/123",
				"urlLabel":      "View Deployment",
			},
			ExecutionState: stateCtx,
			Notifications:  notifCtx,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, notifCtx.calls, 1)
		call := notifCtx.calls[0]
		assert.Equal(t, "Deploy Complete", call.title)
		assert.Equal(t, "The deployment finished successfully.", call.body)
		assert.Equal(t, "https://example.com/deploy/123", call.url)
		assert.Equal(t, "View Deployment", call.urlLabel)
		assert.ElementsMatch(t, []string{"alice@example.com", "bob@example.com"}, call.receivers.Emails)
		assert.Empty(t, call.receivers.Groups)
		assert.Empty(t, call.receivers.Roles)

		assert.True(t, stateCtx.Passed)
		assert.True(t, stateCtx.Finished)
		assert.Equal(t, "default", stateCtx.Channel)
		assert.Equal(t, PayloadType, stateCtx.Type)
	})

	t.Run("sends email without URL", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		notifCtx := &mockNotificationContext{available: true}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"recipientMode": "emails",
				"to":            "alice@example.com",
				"subject":       "Alert",
				"body":          "Something happened.",
			},
			ExecutionState: stateCtx,
			Notifications:  notifCtx,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, notifCtx.calls, 1)
		assert.Equal(t, "", notifCtx.calls[0].url)
		assert.Equal(t, "", notifCtx.calls[0].urlLabel)
	})
}

func TestSendEmail_Execute_MembersMode(t *testing.T) {
	c := &SendEmail{}

	t.Run("sends to users, groups, and roles", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		notifCtx := &mockNotificationContext{available: true}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"recipientMode": "members",
				"recipients": []any{
					map[string]any{"type": "user", "user": "user-123"},
					map[string]any{"type": "group", "group": "engineering"},
					map[string]any{"type": "role", "role": "admin"},
				},
				"subject": "Status Update",
				"body":    "Everything is running smoothly.",
			},
			ExecutionState: stateCtx,
			Notifications:  notifCtx,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, notifCtx.calls, 1)
		call := notifCtx.calls[0]
		assert.Equal(t, "Status Update", call.title)
		assert.Equal(t, "Everything is running smoothly.", call.body)
		assert.ElementsMatch(t, []string{"user-123"}, call.receivers.Emails)
		assert.ElementsMatch(t, []string{"engineering"}, call.receivers.Groups)
		assert.ElementsMatch(t, []string{"admin"}, call.receivers.Roles)

		assert.True(t, stateCtx.Passed)
		assert.True(t, stateCtx.Finished)
		assert.Equal(t, "default", stateCtx.Channel)
		assert.Equal(t, PayloadType, stateCtx.Type)
	})
}

func TestSendEmail_Execute_MissingNotificationContext(t *testing.T) {
	c := &SendEmail{}
	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"recipientMode": "emails",
			"to":            "alice@example.com",
			"subject":       "Test",
			"body":          "Body",
		},
		ExecutionState: stateCtx,
		Notifications:  nil,
	}

	err := c.Execute(ctx)
	assert.ErrorContains(t, err, "notification context is not available")
}

func TestSendEmail_Execute_MissingSubject(t *testing.T) {
	c := &SendEmail{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"recipientMode": "emails",
			"to":            "alice@example.com",
			"subject":       "",
			"body":          "Body",
		},
		ExecutionState: &contexts.ExecutionStateContext{},
		Notifications:  &mockNotificationContext{available: true},
	}

	err := c.Execute(ctx)
	assert.ErrorContains(t, err, "subject is required")
}

func TestSendEmail_Execute_MissingBody(t *testing.T) {
	c := &SendEmail{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"recipientMode": "emails",
			"to":            "alice@example.com",
			"subject":       "Test",
			"body":          "",
		},
		ExecutionState: &contexts.ExecutionStateContext{},
		Notifications:  &mockNotificationContext{available: true},
	}

	err := c.Execute(ctx)
	assert.ErrorContains(t, err, "body is required")
}

func TestSendEmail_Execute_EmailNotConfigured(t *testing.T) {
	c := &SendEmail{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"recipientMode": "emails",
			"to":            "alice@example.com",
			"subject":       "Test",
			"body":          "Body",
		},
		ExecutionState: &contexts.ExecutionStateContext{},
		Notifications:  &mockNotificationContext{available: false},
	}

	err := c.Execute(ctx)
	assert.ErrorContains(t, err, "email delivery is not configured")
}

func TestParseEmailList(t *testing.T) {
	t.Run("single email", func(t *testing.T) {
		emails, err := parseEmailList("alice@example.com")
		require.NoError(t, err)
		assert.Equal(t, []string{"alice@example.com"}, emails)
	})

	t.Run("multiple emails", func(t *testing.T) {
		emails, err := parseEmailList("alice@example.com, bob@example.com, carol@example.com")
		require.NoError(t, err)
		assert.Equal(t, []string{"alice@example.com", "bob@example.com", "carol@example.com"}, emails)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		emails, err := parseEmailList("  alice@example.com  ,  bob@example.com  ")
		require.NoError(t, err)
		assert.Equal(t, []string{"alice@example.com", "bob@example.com"}, emails)
	})

	t.Run("empty string", func(t *testing.T) {
		emails, err := parseEmailList("")
		require.NoError(t, err)
		assert.Empty(t, emails)
	})

	t.Run("invalid email", func(t *testing.T) {
		_, err := parseEmailList("not-valid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email address")
	})

	t.Run("skips empty entries", func(t *testing.T) {
		emails, err := parseEmailList("alice@example.com,,bob@example.com,")
		require.NoError(t, err)
		assert.Equal(t, []string{"alice@example.com", "bob@example.com"}, emails)
	})
}

func TestBuildReceivers(t *testing.T) {
	t.Run("email mode", func(t *testing.T) {
		config := Config{
			RecipientMode: RecipientModeEmails,
			To:            "alice@example.com, bob@example.com",
		}

		receivers, err := buildReceivers(config)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"alice@example.com", "bob@example.com"}, receivers.Emails)
		assert.Empty(t, receivers.Groups)
		assert.Empty(t, receivers.Roles)
	})

	t.Run("members mode with all types", func(t *testing.T) {
		config := Config{
			RecipientMode: RecipientModeMembers,
			Recipients: []Recipient{
				{Type: RecipientTypeUser, User: "user-1"},
				{Type: RecipientTypeUser, User: "user-2"},
				{Type: RecipientTypeGroup, Group: "eng"},
				{Type: RecipientTypeRole, Role: "admin"},
			},
		}

		receivers, err := buildReceivers(config)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"user-1", "user-2"}, receivers.Emails)
		assert.ElementsMatch(t, []string{"eng"}, receivers.Groups)
		assert.ElementsMatch(t, []string{"admin"}, receivers.Roles)
	})

	t.Run("members mode deduplicates", func(t *testing.T) {
		config := Config{
			RecipientMode: RecipientModeMembers,
			Recipients: []Recipient{
				{Type: RecipientTypeUser, User: "user-1"},
				{Type: RecipientTypeUser, User: "user-1"},
				{Type: RecipientTypeGroup, Group: "eng"},
				{Type: RecipientTypeGroup, Group: "eng"},
			},
		}

		receivers, err := buildReceivers(config)
		require.NoError(t, err)
		assert.Len(t, receivers.Emails, 1)
		assert.Len(t, receivers.Groups, 1)
	})

	t.Run("unknown mode", func(t *testing.T) {
		config := Config{RecipientMode: "invalid"}
		_, err := buildReceivers(config)
		assert.ErrorContains(t, err, "unknown recipient mode")
	})
}
