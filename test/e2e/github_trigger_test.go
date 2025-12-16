package e2e

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"

	q "github.com/superplanehq/superplane/test/e2e/queries"
	v "github.com/superplanehq/superplane/test/e2e/vcr"
)

func TestGithubTrigger(t *testing.T) {
	steps := &GithubTriggerSteps{t: t}

	// These credentials are created specifically for E2E.
	// They are now revoked and have no access to any real repositories.
	// Used only in the recorded VCR cassettes.

	const githubOwner = "puppies-inc"
	const githubTokenValue = "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"

	v.Run(t, "addding a github trigger node", func(t *testing.T) {
		steps.start()
		steps.givenAGithubIntegrationExists(githubOwner, githubTokenValue)
		steps.givenACanvasExists()
		steps.addGithubTriggerNode()
		steps.assertGithubTriggerNodeExistsInDB()
	})

	v.Run(t, "receiving github trigger events", func(t *testing.T) {
		steps.start()
		steps.givenAGithubIntegrationExists(githubOwner, githubTokenValue)
		steps.givenACanvasWithGithubTriggerAndNoop()
		steps.saveCanvas()
		steps.waitForWebhookSetup()
		steps.simulateReceivingGithubEvent()
		steps.assertWebhookProcessed()
	})
}

type GithubTriggerSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps

	// integrationName is the name as shown in the UI dropdown
	integrationName string
}

func (s *GithubTriggerSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *GithubTriggerSteps) givenAGithubIntegrationExists(ownerSlug, token string) {
	// Use the UI flow to create the integration, as in integrations_test.go
	ownerInput := q.Locator(`input[data-testid="github-owner-input"]`)
	tokenInput := q.Locator(`input[data-testid="integration-api-token-input"]`)

	s.session.Visit("/" + s.session.OrgID.String() + "/settings/integrations")
	s.session.AssertText("Integrations")

	s.session.Click(q.Text("Add Integration"))
	s.session.AssertText("Select Integration Type")

	s.session.Click(q.Locator(`button:has-text("GitHub")`))

	s.session.FillIn(ownerInput, ownerSlug)
	s.session.FillIn(tokenInput, token)

	s.session.Click(q.TestID("create-integration-button"))

	s.integrationName = ownerSlug + "-account"

	s.session.Sleep(1000)
}

func (s *GithubTriggerSteps) givenACanvasExists() {
	s.canvas = shared.NewCanvasSteps("E2E Github Trigger", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
}

func (s *GithubTriggerSteps) addGithubTriggerNode() {
	// Open the building blocks sidebar before dragging
	openButton := q.TestID("open-sidebar-button")
	loc := openButton.Run(s.session)
	if isVisible, _ := loc.IsVisible(); isVisible {
		s.session.Click(openButton)
		s.session.Sleep(300)
	}

	source := q.TestID("building-block-github")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, 800, 200)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), "GitHub Trigger")

	// Select the GitHub integration
	integrationTrigger := q.Locator(`label:has-text("GitHub integration") + div button`)
	s.session.Click(integrationTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("` + s.integrationName + `")`))

	// Select the repository
	repositoryTrigger := q.Locator(`label:has-text("Repository") + div button`)
	s.session.Click(repositoryTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("front")`))

	// Select the event type
	eventTypeTrigger := q.Locator(`label:has-text("Event Type") + div button`)
	s.session.Click(eventTypeTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("Push")`))

	s.session.Click(q.TestID("add-node-button"))

	s.session.Sleep(1000)
}

func (s *GithubTriggerSteps) saveCanvas() {
	s.canvas.Save()
}

func (s *GithubTriggerSteps) assertGithubTriggerNodeExistsInDB() {
	node := s.canvas.GetNodeFromDB("GitHub Trigger")
	require.NotNil(s.t, node, "github trigger node not found in DB")
}

func (s *GithubTriggerSteps) givenACanvasWithGithubTriggerAndNoop() {
	s.givenACanvasExists()
	s.addGithubTriggerNode()

	s.canvas.AddNoop("Noop", models.Position{X: 1500, Y: 200})
	s.canvas.Connect("GitHub Trigger", "Noop")
}

func (s *GithubTriggerSteps) waitForWebhookSetup() {
	for i := 0; i < 10; i++ {
		node := s.canvas.GetNodeFromDB("GitHub Trigger")
		require.NotNil(s.t, node, "github trigger node not found in DB")

		webhook, err := models.FindWebhook(*node.WebhookID)
		if err == nil && webhook.State == models.WebhookStateReady {
			return
		}

		time.Sleep(1 * time.Second)
	}

	s.t.Fatal("webhook was not set up in time")
}

func (s *GithubTriggerSteps) simulateReceivingGithubEvent() {
	node := s.canvas.GetNodeFromDB("GitHub Trigger")
	require.NotNil(s.t, node, "github trigger node not found in DB")
	require.NotNil(s.t, node.WebhookID, "github trigger node does not have a webhook")

	webhook, err := models.FindWebhook(*node.WebhookID)
	require.NoError(s.t, err, "failed to load webhook for github trigger node")
	require.Equal(s.t, models.WebhookStateReady, webhook.State, "webhook is not in ready state")

	payload := map[string]any{
		"head_commit": map[string]any{
			"message": "Initial commit",
			"id":      "abc123",
			"author": map[string]any{
				"name": "E2E Tester",
			},
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(s.t, err, "failed to marshal github webhook payload")

	// NOTE: In test environments we typically run with NO_ENCRYPTION=yes,
	// so webhook.Secret contains the raw key used for signing.
	mac := hmac.New(sha256.New, webhook.Secret)
	_, err = mac.Write(body)
	require.NoError(s.t, err, "failed to compute webhook signature")

	signature := hex.EncodeToString(mac.Sum(nil))

	baseURL := os.Getenv("BASE_URL")
	require.NotEmpty(s.t, baseURL, "BASE_URL must be set for github trigger e2e tests")

	url := fmt.Sprintf("%s/api/v1/webhooks/%s", baseURL, webhook.ID.String())

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	require.NoError(s.t, err, "failed to create github webhook request")

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", "sha256="+signature)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(s.t, err, "failed to send github webhook request")
	defer resp.Body.Close()

	require.Equal(s.t, http.StatusOK, resp.StatusCode, "unexpected status code from github webhook endpoint")

	// Give the workflow_event_router and node executor some time to process the event
	time.Sleep(3 * time.Second)
}

func (s *GithubTriggerSteps) assertWebhookProcessed() {
	executions := s.canvas.GetExecutionsForNode("Noop")
	require.NotEmpty(s.t, executions, "expected at least one execution for second node")
}
