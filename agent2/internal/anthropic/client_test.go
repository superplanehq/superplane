package anthropic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_CreateSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("unexpected api key: %s", r.Header.Get("x-api-key"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"sesn_123","status":"idle","agent":{"id":"agent_abc"},"usage":{"input_tokens":0,"output_tokens":0}}`))
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-key",
		httpClient: http.DefaultClient,
		baseURL:    server.URL + "/v1",
	}

	session, err := client.CreateSession(context.Background(), CreateSessionRequest{
		Agent: "agent_abc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.ID != "sesn_123" {
		t.Errorf("expected session ID sesn_123, got %s", session.ID)
	}
}

func TestClient_SendMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/sesn_123/events" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-key",
		httpClient: http.DefaultClient,
		baseURL:    server.URL + "/v1",
	}

	err := client.SendMessage(context.Background(), "sesn_123", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_GetSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"sesn_123","status":"idle","usage":{"input_tokens":100,"output_tokens":50}}`))
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-key",
		httpClient: http.DefaultClient,
		baseURL:    server.URL + "/v1",
	}

	session, err := client.GetSession(context.Background(), "sesn_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.Status != "idle" {
		t.Errorf("expected idle, got %s", session.Status)
	}
	if session.Usage.OutputTokens != 50 {
		t.Errorf("expected 50 output tokens, got %d", session.Usage.OutputTokens)
	}
}
