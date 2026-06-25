package oidc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type verifyCommand struct {
	token        *string
	apiURL       *string
	orgID        *string
	canvasID     *string
	nodeID       *string
	component    *string
	projectID    *string
	pipelineFile *string
	ref          *string
	commitSha    *string
}

type verifyRequest struct {
	Token    string                 `json:"token"`
	Expected *verifyExpectedRequest `json:"expected,omitempty"`
}

type verifyExpectedRequest struct {
	OrgID        string `json:"org_id,omitempty"`
	CanvasID     string `json:"canvas_id,omitempty"`
	NodeID       string `json:"node_id,omitempty"`
	Component    string `json:"component,omitempty"`
	ProjectID    string `json:"project_id,omitempty"`
	PipelineFile string `json:"pipeline_file,omitempty"`
	Ref          string `json:"ref,omitempty"`
	CommitSha    string `json:"commit_sha,omitempty"`
}

type verifyResponse struct {
	Valid  bool `json:"valid"`
	Claims struct {
		OrgID        string `json:"org_id"`
		CanvasID     string `json:"canvas_id"`
		NodeID       string `json:"node_id"`
		ExecutionID  string `json:"execution_id"`
		Component    string `json:"component"`
		ProjectID    string `json:"project_id"`
		PipelineFile string `json:"pipeline_file"`
		Ref          string `json:"ref"`
		CommitSha    string `json:"commit_sha"`
	} `json:"claims"`
	Error string `json:"error"`
}

func (c *verifyCommand) Execute(ctx core.CommandContext) error {
	token := strings.TrimSpace(*c.token)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("SUPERPLANE_OIDC_TOKEN"))
	}
	if token == "" {
		return fmt.Errorf("token is required (use --token or SUPERPLANE_OIDC_TOKEN)")
	}

	apiURL := strings.TrimRight(strings.TrimSpace(*c.apiURL), "/")
	if apiURL == "" {
		if ctx.Config != nil {
			apiURL = strings.TrimRight(ctx.Config.GetURL(), "/")
		}
	}
	if apiURL == "" {
		apiURL = "http://localhost:8000"
	}

	expected := buildVerifyExpected(c)
	requestBody, err := json.Marshal(verifyRequest{
		Token:    token,
		Expected: expected,
	})
	if err != nil {
		return err
	}

	endpoint := apiURL + "/api/v1/oidc/verify"
	response, err := http.Post(endpoint, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("verify request failed: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read verify response: %w", err)
	}

	var result verifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse verify response: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("token verification failed")
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(result.Claims)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Token verified\n")
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(stdout, "Organization: %s\n", result.Claims.OrgID)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "Canvas: %s\n", result.Claims.CanvasID)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "Node: %s\n", result.Claims.NodeID)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "Execution: %s\n", result.Claims.ExecutionID)
		if err != nil {
			return err
		}
		if result.Claims.Component != "" {
			_, err = fmt.Fprintf(stdout, "Component: %s\n", result.Claims.Component)
			if err != nil {
				return err
			}
		}
		if result.Claims.PipelineFile != "" {
			_, err = fmt.Fprintf(stdout, "Pipeline file: %s\n", result.Claims.PipelineFile)
			if err != nil {
				return err
			}
		}
		if result.Claims.CommitSha != "" {
			_, err = fmt.Fprintf(stdout, "Commit SHA: %s\n", result.Claims.CommitSha)
		}
		return err
	})
}

func buildVerifyExpected(c *verifyCommand) *verifyExpectedRequest {
	expected := &verifyExpectedRequest{
		OrgID:        strings.TrimSpace(*c.orgID),
		CanvasID:     strings.TrimSpace(*c.canvasID),
		NodeID:       strings.TrimSpace(*c.nodeID),
		Component:    strings.TrimSpace(*c.component),
		ProjectID:    strings.TrimSpace(*c.projectID),
		PipelineFile: strings.TrimSpace(*c.pipelineFile),
		Ref:          strings.TrimSpace(*c.ref),
		CommitSha:    strings.TrimSpace(*c.commitSha),
	}

	if expected.OrgID == "" &&
		expected.CanvasID == "" &&
		expected.NodeID == "" &&
		expected.Component == "" &&
		expected.ProjectID == "" &&
		expected.PipelineFile == "" &&
		expected.Ref == "" &&
		expected.CommitSha == "" {
		return nil
	}

	return expected
}
