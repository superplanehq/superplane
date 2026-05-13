package agents

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestAnthropicIntegration is a live integration test against the Anthropic API.
// Run with: go test -run TestAnthropicIntegration -v ./pkg/agents/ -tags=integration
// Requires: ANTHROPIC_API_KEY, ANTHROPIC_AGENT_ID, ANTHROPIC_ENVIRONMENT_ID
func TestAnthropicIntegration(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	agentID := os.Getenv("ANTHROPIC_AGENT_ID")
	envID := os.Getenv("ANTHROPIC_ENVIRONMENT_ID")

	if apiKey == "" || agentID == "" || envID == "" {
		t.Skip("Skipping: ANTHROPIC_API_KEY, ANTHROPIC_AGENT_ID, ANTHROPIC_ENVIRONMENT_ID required")
	}

	client := NewClient(apiKey, agentID, envID)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. Create session
	t.Log("Creating session...")
	session, err := client.CreateSession(ctx)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	t.Logf("Session created: ID=%s, Status=%s", session.ID, session.Status)

	if session.ID == "" {
		t.Fatal("Session ID is empty")
	}

	// 2. Get session
	t.Log("Getting session...")
	got, err := client.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	t.Logf("Session status: %s", got.Status)

	// 3. Send message
	t.Log("Sending message...")
	err = client.SendMessage(ctx, session.ID, "List all canvases in this organization")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	t.Log("Message sent successfully")

	// 4. Poll for completion
	t.Log("Polling for completion...")
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)

		s, err := client.GetSession(ctx, session.ID)
		if err != nil {
			t.Fatalf("GetSession (poll %d) failed: %v", i, err)
		}
		t.Logf("Poll %d: status=%s, output_tokens=%d", i, s.Status, s.Usage.OutputTokens)

		if s.Status == "idle" && s.Usage.OutputTokens > 0 {
			t.Log("Session completed!")

			// 5. List events
			events, err := client.ListEvents(ctx, session.ID, 50)
			if err != nil {
				t.Fatalf("ListEvents failed: %v", err)
			}
			t.Logf("Got %d events", len(events.Data))
			for _, ev := range events.Data {
				t.Logf("  Event: type=%s, id=%s", ev.Type, ev.ID)
			}
			return
		}

		if s.Status == "failed" {
			t.Fatal("Session failed")
		}
	}

	t.Fatal("Timed out waiting for session to complete")
}
