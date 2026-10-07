package linear

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestParseAppEvent(t *testing.T) {
	headers := http.Header{}
	headers.Set(EventHeader, IssueResourceType)
	body := []byte(`{
		"action": "create",
		"type": "Issue",
		"organizationId": "org-1",
		"url": "https://linear.app/acme/issue/ENG-1",
		"data": {"id": "issue-1", "teamId": "team-1", "team": {"id": "team-1", "key": "ENG"}}
	}`)

	event, err := ParseAppEvent(headers, body)
	require.NoError(t, err)
	assert.Equal(t, AppEvent{
		OrganizationID: "org-1",
		WorkspaceKey:   "acme",
		ResourceType:   IssueResourceType,
		TeamID:         "team-1",
		TeamKey:        "ENG",
		IssueID:        "issue-1",
	}, event)
}

func TestParseAppEvent__CommentUsesParentIssueTeam(t *testing.T) {
	body := []byte(`{
		"type": "Comment",
		"organizationId": "org-1",
		"url": "https://linear.app/acme/issue/ENG-1#comment",
		"data": {
			"id": "comment-1",
			"issue": {"id": "issue-1", "team": {"id": "team-9", "key": "ENG"}}
		}
	}`)

	event, err := ParseAppEvent(nil, body)
	require.NoError(t, err)
	assert.Equal(t, CommentResourceType, event.ResourceType)
	assert.Equal(t, "team-9", event.TeamID)
	assert.Equal(t, "ENG", event.TeamKey)
}

func TestSubscriptionMatches(t *testing.T) {
	teams := []Team{{ID: "team-1", Key: "ENG"}}
	issue := AppEvent{ResourceType: IssueResourceType, TeamID: "team-1", TeamKey: "ENG"}

	assert.True(t, SubscriptionMatches(WebhookConfiguration{
		TeamID:       "team-1",
		ResourceType: IssueResourceType,
	}, issue, teams))

	assert.False(t, SubscriptionMatches(WebhookConfiguration{
		TeamID:       "team-2",
		ResourceType: IssueResourceType,
	}, issue, teams))

	assert.False(t, SubscriptionMatches(WebhookConfiguration{
		TeamID:       "team-1",
		ResourceType: CommentResourceType,
	}, issue, teams))

	assert.True(t, SubscriptionMatches(WebhookConfiguration{
		ResourceType: IssueResourceType,
	}, issue, teams))

	byKey := AppEvent{ResourceType: IssueResourceType, TeamKey: "ENG"}
	assert.True(t, SubscriptionMatches(WebhookConfiguration{
		TeamID:       "team-1",
		ResourceType: IssueResourceType,
	}, byKey, teams))

	assert.False(t, SubscriptionMatches(WebhookConfiguration{
		TeamID:       "team-1",
		ResourceType: IssueResourceType,
	}, AppEvent{ResourceType: IssueResourceType}, teams))
}

func TestParseAppEvent__AttachmentUsesParentIssueTeam(t *testing.T) {
	raw, err := os.ReadFile("example_data_on_issue_attachment.json")
	require.NoError(t, err)

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &envelope))

	event, err := ParseAppEvent(nil, envelope.Data)
	require.NoError(t, err)
	assert.Equal(t, AttachmentResourceType, event.ResourceType)
	assert.Equal(t, "ENG", event.TeamKey)
	assert.Empty(t, event.TeamID)
	assert.NotEmpty(t, event.IssueID)

	teams := []Team{{ID: "team-1", Key: "ENG", Name: "Engineering"}}
	assert.True(t, SubscriptionMatches(WebhookConfiguration{
		TeamID:       "team-1",
		ResourceType: AttachmentResourceType,
	}, event, teams))
	assert.False(t, SubscriptionMatches(WebhookConfiguration{
		TeamID:       "team-2",
		ResourceType: AttachmentResourceType,
	}, event, teams))
}

func TestResolveEventTeam__LooksUpIssueWhenThePayloadHasNoTeam(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"issue":{"id":"issue-1","identifier":"ENG-9","team":{"id":"team-9","key":"ENG","name":"Engineering"}}}}`),
		},
	}

	event, err := ResolveEventTeam(httpContext, newAuthorizedIntegration(), AppEvent{
		ResourceType: AttachmentResourceType,
		IssueID:      "issue-1",
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, "team-9", event.TeamID)
	assert.Equal(t, "ENG", event.TeamKey)
	require.Len(t, httpContext.Requests, 1)
}

func TestResolveEventTeam__UsesTheKnownTeamKeyWithoutARequest(t *testing.T) {
	httpContext := &contexts.HTTPContext{}
	event, err := ResolveEventTeam(httpContext, newAuthorizedIntegration(), AppEvent{
		ResourceType: AttachmentResourceType,
		TeamKey:      "ENG",
		IssueID:      "issue-1",
	}, []Team{{ID: "team-1", Key: "ENG"}})

	require.NoError(t, err)
	assert.Equal(t, "team-1", event.TeamID)
	assert.Empty(t, httpContext.Requests)
}

func TestSignatureMatches(t *testing.T) {
	body := []byte(`{"action":"create"}`)
	mac := hmac.New(sha256.New, []byte("app-secret"))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	assert.True(t, SignatureMatches(signature, body, []byte("app-secret")))
	assert.False(t, SignatureMatches(signature, body, []byte("other-secret")))
	assert.False(t, SignatureMatches("", body, []byte("app-secret")))
}

func TestIsAppLevelWebhook(t *testing.T) {
	assert.True(t, IsAppLevelWebhook(map[string]any{"appLevel": true}))
	assert.False(t, IsAppLevelWebhook(map[string]any{"id": "w1"}))
}
