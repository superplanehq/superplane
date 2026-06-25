package oidc

import (
	"fmt"
	"time"
)

const (
	ExecutionTokenAudience = "superplane-ci"
	ExecutionTokenDuration = time.Hour

	ClaimOrgID        = "org_id"
	ClaimCanvasID     = "canvas_id"
	ClaimNodeID       = "node_id"
	ClaimExecutionID  = "execution_id"
	ClaimComponent    = "component"
	ClaimProjectID    = "project_id"
	ClaimPipelineFile = "pipeline_file"
	ClaimRef          = "ref"
	ClaimCommitSha    = "commit_sha"
)

type ExecutionTokenInput struct {
	OrganizationID string
	CanvasID       string
	NodeID         string
	ExecutionID    string
	Component      string
	ProjectID      string
	PipelineFile   string
	Ref            string
	CommitSha      string
}

type ExecutionTokenClaims struct {
	Subject      string
	Audience     string
	OrgID        string
	CanvasID     string
	NodeID       string
	ExecutionID  string
	Component    string
	ProjectID    string
	PipelineFile string
	Ref          string
	CommitSha    string
	IssuedAt     int64
	ExpiresAt    int64
}

type ExecutionTokenExpected struct {
	OrgID        string
	CanvasID     string
	NodeID       string
	Component    string
	ProjectID    string
	PipelineFile string
	Ref          string
	CommitSha    string
}

func SignExecutionToken(provider Provider, input ExecutionTokenInput) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("OIDC provider is not configured")
	}

	subject := fmt.Sprintf("execution:%s", input.ExecutionID)
	claims := map[string]any{
		ClaimOrgID:       input.OrganizationID,
		ClaimCanvasID:    input.CanvasID,
		ClaimNodeID:      input.NodeID,
		ClaimExecutionID: input.ExecutionID,
		ClaimComponent:   input.Component,
	}

	if input.ProjectID != "" {
		claims[ClaimProjectID] = input.ProjectID
	}
	if input.PipelineFile != "" {
		claims[ClaimPipelineFile] = input.PipelineFile
	}
	if input.Ref != "" {
		claims[ClaimRef] = input.Ref
	}
	if input.CommitSha != "" {
		claims[ClaimCommitSha] = input.CommitSha
	}

	return provider.Sign(subject, ExecutionTokenDuration, ExecutionTokenAudience, claims)
}

func ParseExecutionTokenClaims(raw map[string]any) (ExecutionTokenClaims, error) {
	claims := ExecutionTokenClaims{
		Subject:      stringClaim(raw, "sub"),
		Audience:     audienceClaim(raw),
		OrgID:        stringClaim(raw, ClaimOrgID),
		CanvasID:     stringClaim(raw, ClaimCanvasID),
		NodeID:       stringClaim(raw, ClaimNodeID),
		ExecutionID:  stringClaim(raw, ClaimExecutionID),
		Component:    stringClaim(raw, ClaimComponent),
		ProjectID:    stringClaim(raw, ClaimProjectID),
		PipelineFile: stringClaim(raw, ClaimPipelineFile),
		Ref:          stringClaim(raw, ClaimRef),
		CommitSha:    stringClaim(raw, ClaimCommitSha),
		IssuedAt:     int64Claim(raw, "iat"),
		ExpiresAt:    int64Claim(raw, "exp"),
	}

	if claims.OrgID == "" || claims.CanvasID == "" || claims.NodeID == "" || claims.ExecutionID == "" {
		return ExecutionTokenClaims{}, fmt.Errorf("token is missing required execution claims")
	}

	if claims.Audience != ExecutionTokenAudience {
		return ExecutionTokenClaims{}, fmt.Errorf("token audience must be %q", ExecutionTokenAudience)
	}

	return claims, nil
}

func (expected ExecutionTokenExpected) Matches(claims ExecutionTokenClaims) error {
	if expected.OrgID != "" && expected.OrgID != claims.OrgID {
		return fmt.Errorf("org_id mismatch")
	}
	if expected.CanvasID != "" && expected.CanvasID != claims.CanvasID {
		return fmt.Errorf("canvas_id mismatch")
	}
	if expected.NodeID != "" && expected.NodeID != claims.NodeID {
		return fmt.Errorf("node_id mismatch")
	}
	if expected.Component != "" && expected.Component != claims.Component {
		return fmt.Errorf("component mismatch")
	}
	if expected.ProjectID != "" && expected.ProjectID != claims.ProjectID {
		return fmt.Errorf("project_id mismatch")
	}
	if expected.PipelineFile != "" && expected.PipelineFile != claims.PipelineFile {
		return fmt.Errorf("pipeline_file mismatch")
	}
	if expected.Ref != "" && expected.Ref != claims.Ref {
		return fmt.Errorf("ref mismatch")
	}
	if expected.CommitSha != "" && expected.CommitSha != claims.CommitSha {
		return fmt.Errorf("commit_sha mismatch")
	}

	return nil
}

func stringClaim(raw map[string]any, key string) string {
	value, ok := raw[key]
	if !ok || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func int64Claim(raw map[string]any, key string) int64 {
	value, ok := raw[key]
	if !ok || value == nil {
		return 0
	}

	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		return 0
	}
}

func audienceClaim(raw map[string]any) string {
	value, ok := raw["aud"]
	if !ok || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		if len(typed) == 0 {
			return ""
		}
		return fmt.Sprint(typed[0])
	default:
		return fmt.Sprint(typed)
	}
}
