package public

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runner "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunnerListMembers(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	ctx, _, _ := mustTaskLinkedAgentRun(t, r)
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60, nil))
	member := mustTaskOwnerMember(t, r, "members-dev")

	rec := runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/members", "")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct {
		Members []struct {
			UserID      string   `json:"user_id"`
			Name        string   `json:"name"`
			GitHubLogin string   `json:"github_login"`
			Roles       []string `json:"roles"`
			OpenTasks   int      `json:"open_tasks"`
		} `json:"members"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	names := []string{}
	var found bool
	for _, listed := range body.Members {
		names = append(names, listed.Name)
		if listed.UserID == member.ID.String() {
			found = true
			assert.Equal(t, "members-dev", listed.GitHubLogin)
		}
	}
	assert.True(t, found, rec.Body.String())
	assert.IsNonDecreasing(t, lowerAll(names))

	rec = runnerRequest(t, server, http.MethodGet, runnerTokenWithScopes(t, token, runner.ScopeOwnerRead), "/api/v1/runner/members", "")
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}

func TestRunnerChecks(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	ctx, factory, order := mustTaskLinkedAgentRun(t, r)
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60, nil))
	db := database.DB(t.Context())
	_, err := order.ReportCheck(db, models.FactoryWorkOrderCheckParams{Key: "coverage", Name: "Coverage", Score: 5, MaxScore: 5})
	require.NoError(t, err)
	check := func(key string) string {
		return `{"key":"` + key + `","name":"API latency","score":2,"max_score":5,"level":"caution","reason":"p99 grew."}`
	}

	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/checks", `{"checks":[`+check("api-latency")+`]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"checks":[{"key":"api-latency","name":"API latency","score":2,"max_score":5,"level":"caution","reason":"p99 grew."}]}`, rec.Body.String())

	rec = runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/checks?task="+url.QueryEscape(factory.WorkOrderKey(order.Number)), "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.ElementsMatch(t, []string{"coverage", "api-latency"}, runnerCheckKeys(t, rec.Body.Bytes()))

	for _, key := range []string{"coverage", models.PlanningClarityCheckKey, "risk-review"} {
		rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/checks", `{"checks":[`+check(key)+`]}`)
		assert.Equal(t, http.StatusConflict, rec.Code, key)
		assert.Contains(t, rec.Body.String(), "check key belongs to SuperPlane")
	}
	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/checks", `{"checks":[{"key":"x","name":"X","score":1,"max_score":5,"level":"warning","reason":"r"}]}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	rec = runnerRequest(t, server, http.MethodDelete, token, "/api/v1/runner/task/checks", `{"keys":["api-latency","coverage"]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"removed":["api-latency"]}`, rec.Body.String())
	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, "coverage", checks[0].Key)

	rec = runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/checks?task=missing", "")
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	readOnly := runnerTokenWithScopes(t, token, runner.ScopeChecksRead)
	rec = runnerRequest(t, server, http.MethodPost, readOnly, "/api/v1/runner/task/checks", `{"checks":[`+check("other")+`]}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}

func TestRunnerArtifactsByKey(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	ctx, _, order := mustTaskLinkedAgentRun(t, r)
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60, nil))

	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/links", `{"url":"https://preview.example","title":"Preview","key":"preview"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var added struct {
		ArtifactID string `json:"artifact_id"`
		Type       string `json:"type"`
		Key        string `json:"key"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &added))
	assert.Equal(t, "link", added.Type)
	assert.Equal(t, "preview", added.Key)
	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/markdown", `{"body":"# Notes","title":"Notes"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	rec = runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/task/artifacts?type=link", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var listed struct {
		Artifacts []runnerArtifactView `json:"artifacts"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	require.Len(t, listed.Artifacts, 1)
	assert.Equal(t, added.ArtifactID, listed.Artifacts[0].ID)
	assert.Equal(t, "preview", listed.Artifacts[0].Key)
	assert.Equal(t, "https://preview.example", listed.Artifacts[0].URL)

	for _, query := range []string{"key=preview", "id=" + added.ArtifactID} {
		rec = runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/task/artifact?"+query, "")
		require.Equal(t, http.StatusOK, rec.Code, query)
		assert.Contains(t, rec.Body.String(), `"title":"Preview"`)
	}
	rec = runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/task/artifacts?type=video", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	rec = runnerRequest(t, server, http.MethodDelete, token, "/api/v1/runner/task/artifacts", `{"key":"preview"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	artifacts, err := order.ListArtifacts(database.DB(t.Context()))
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, models.FactoryWorkOrderArtifactTypeMarkdown, artifacts[0].Type)
}

func TestRunnerPlanningRunAttachesArtifactsToTheDraft(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	server, session, factoryModel, planningToken := mustPlanningRunnerSession(t, r)
	token := mustPlanningGlobalRunnerToken(t, r, session, planningToken)

	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/links", `{"url":"https://design.example","title":"Design","key":"design"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/checks",
		`{"checks":[{"key":"scope","name":"Scope","score":3,"max_score":5,"level":"caution","reason":"Two services change."}]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	db := database.DB(t.Context())
	require.NotNil(t, session.DraftWorkOrderID)
	draft, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
	require.NoError(t, err)
	_, err = findRunnerArtifactByKey(db, draft, "design")
	require.NoError(t, err)
	checks, err := draft.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, models.FactoryWorkOrderCheckSourceAgent, checks[0].Source)
}

func runnerTokenWithScopes(t *testing.T, token string, scopes ...runner.RunnerToolScope) string {
	t.Helper()
	scope, err := runner.ParseRunnerToken(jwt.NewSigner("test"), token)
	require.NoError(t, err)
	scope.Scopes = scopes
	limited, err := runner.MintRunnerToken(jwt.NewSigner("test"), *scope, time.Minute)
	require.NoError(t, err)
	return limited
}

func runnerCheckKeys(t *testing.T, body []byte) []string {
	t.Helper()
	var response struct {
		Checks []runnerCheckView `json:"checks"`
	}
	require.NoError(t, json.Unmarshal(body, &response))
	keys := []string{}
	for _, check := range response.Checks {
		keys = append(keys, check.Key)
	}
	return keys
}

func lowerAll(values []string) []string {
	lowered := make([]string, 0, len(values))
	for _, value := range values {
		lowered = append(lowered, strings.ToLower(value))
	}
	return lowered
}
