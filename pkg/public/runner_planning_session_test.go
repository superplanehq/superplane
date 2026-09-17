package public

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestRunnerPlanningSessionDraftRouteIsRemoved(t *testing.T) {
	r := support.Setup(t)
	server, _, _, token := mustPlanningRunnerSession(t, r)

	draft := httptest.NewRequest(http.MethodPost, "/api/v1/runner/planning-sessions/drafts", bytes.NewReader([]byte(
		`{"title":"Retry refunds","description":"Stop double charges."}`,
	)))
	draft.Header.Set("Authorization", "Bearer "+token)
	draftRec := httptest.NewRecorder()
	server.Router.ServeHTTP(draftRec, draft)
	require.Equal(t, http.StatusNotFound, draftRec.Code, draftRec.Body.String())
}

func TestRunnerPlanningSessionSpecAndConfidence(t *testing.T) {
	r := support.Setup(t)
	server, session, factoryModel, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	require.NotNil(t, session.DraftWorkOrderID)
	order, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
	require.NoError(t, err)

	spec := httptest.NewRequest(http.MethodPost, "/api/v1/runner/planning-sessions/specs", bytes.NewReader([]byte(
		`{"body":"# Retry refunds\n\n## Executive summary\n\nStop double charges.\n"}`,
	)))
	spec.Header.Set("Authorization", "Bearer "+token)
	specRec := httptest.NewRecorder()
	server.Router.ServeHTTP(specRec, spec)
	require.Equal(t, http.StatusOK, specRec.Code, specRec.Body.String())

	confidence := httptest.NewRequest(http.MethodPost, "/api/v1/runner/planning-sessions/confidence", bytes.NewReader([]byte(
		`{"score":4,"summary":"This issue is a good fit for an agent."}`,
	)))
	confidence.Header.Set("Authorization", "Bearer "+token)
	confidenceRec := httptest.NewRecorder()
	server.Router.ServeHTTP(confidenceRec, confidence)
	require.Equal(t, http.StatusOK, confidenceRec.Code, confidenceRec.Body.String())

	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, models.PlanningSpecArtifactKey+":"+order.ID.String(), *artifacts[0].Key)
	assert.Contains(t, string(artifacts[0].Data), "Stop double charges.")

	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, 4.0, checks[0].Score)
}

func TestRunnerPlanningSessionConfidenceWithoutSpec(t *testing.T) {
	r := support.Setup(t)
	server, session, factoryModel, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	require.NotNil(t, session.DraftWorkOrderID)
	order, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runner/planning-sessions/confidence", bytes.NewReader([]byte(
		`{"score":2,"summary":"The request is still missing the failing path."}`,
	)))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	assert.Empty(t, artifacts)

	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, 2.0, checks[0].Score)
	assert.Equal(t, "The request is still missing the failing path.", checks[0].Summary)
}

func TestRunnerPlanningSessionSurvey(t *testing.T) {
	r := support.Setup(t)
	server, session, _, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runner/planning-sessions/surveys", bytes.NewReader([]byte(
		`{"questions":[{"prompt":"What is the priority?","options":["High","Low"]}]}`,
	)))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	reloaded, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	require.Len(t, reloaded.CurrentSurvey().Questions, 1)
	assert.Equal(t, "What is the priority?", reloaded.CurrentSurvey().Questions[0].Prompt)
	assert.Equal(t, []string{"High", "Low"}, reloaded.CurrentSurvey().Questions[0].Options)
}

func TestRunnerPlanningSessionRecordsAgentMessage(t *testing.T) {
	r := support.Setup(t)
	server, session, _, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runner/planning-sessions/agent-messages", bytes.NewReader([]byte(
		`{"text":"I found the retry seam in billing/retry.go."}`,
	)))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	messages, err := models.ListPlanningSessionMessages(db, session.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, models.PlanningSessionMessageRoleAgent, messages[0].Role)
	assert.Equal(t, "I found the retry seam in billing/retry.go.", messages[0].Text)
	assert.True(t, messages[0].Delivered)
}

func TestRunnerPlanningSessionStoresActivityAndLinksAgentMessage(t *testing.T) {
	r := support.Setup(t)
	server, session, _, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	activityID := uuid.New()
	startedAt := time.Now().Add(-time.Second).UnixMilli()

	body := fmt.Sprintf(`{
		"schema_version":2,
		"activity_id":%q,
		"provider":"codex",
		"turn":1,
		"sequence":4,
		"status":"passed",
		"started_at":%d,
		"completed_at":%d,
		"items":[{"type":"tool","id":"tool-1","kind":"bash","input":"go test ./pkg/models","status":"passed"}]
	}`, activityID.String(), startedAt, time.Now().UnixMilli())
	req := httptest.NewRequest(http.MethodPut, "/api/v1/runner/planning-sessions/activities/"+activityID.String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	messageBody := fmt.Sprintf(`{"text":"Ready.","activity_id":%q}`, activityID.String())
	req = httptest.NewRequest(http.MethodPost, "/api/v1/runner/planning-sessions/agent-messages", bytes.NewReader([]byte(messageBody)))
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	activities, err := models.ListPlanningSessionActivities(db, session.ID)
	require.NoError(t, err)
	require.Len(t, activities, 1)
	assert.Equal(t, int64(4), activities[0].LastSequence)
	assert.Equal(t, "go test ./pkg/models", activities[0].Snapshot.Data().Items[0].Input)
	messages, err := models.ListPlanningSessionMessages(db, session.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.NotNil(t, messages[0].ActivityID)
	assert.Equal(t, activityID, *messages[0].ActivityID)
}

func TestRunnerPlanningSessionIgnoresStaleActivitySnapshot(t *testing.T) {
	r := support.Setup(t)
	server, session, _, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	activityID := uuid.New()
	startedAt := time.Now().UnixMilli()

	put := func(sequence int, provider string) {
		body := fmt.Sprintf(`{"schema_version":2,"activity_id":%q,"provider":%q,"turn":1,"sequence":%d,"status":"running","started_at":%d,"items":[]}`,
			activityID.String(), provider, sequence, startedAt)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/runner/planning-sessions/activities/"+activityID.String(), bytes.NewReader([]byte(body)))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	}

	put(5, "claude")
	put(4, "stale")
	activities, err := models.ListPlanningSessionActivities(db, session.ID)
	require.NoError(t, err)
	require.Len(t, activities, 1)
	assert.Equal(t, int64(5), activities[0].LastSequence)
	assert.Equal(t, "claude", activities[0].Provider)
}

func TestRunnerPlanningSessionRejectsOtherToken(t *testing.T) {
	r := support.Setup(t)
	server, signer := mustRunnerLiveLogServer(t, r)
	token, err := signer.GenerateWithClaims(time.Hour, map[string]string{
		"purpose":       "other",
		"org_id":        r.Organization.ID.String(),
		"factory_id":    uuid.New().String(),
		"session_id":    uuid.New().String(),
		"canvas_run_id": uuid.New().String(),
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/planning-sessions/wait", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRunnerPlanningSessionRejectsTaskCreationKind(t *testing.T) {
	r := support.Setup(t)
	server, signer := mustRunnerLiveLogServer(t, r)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "planning", "start")
	session, err := factoryModel.StartPlanningSession(db, models.StartPlanningSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		Entrypoint:      entrypoint,
	})
	require.NoError(t, err)
	token := mustPlanningRunnerToken(t, signer, session)

	requests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "wait", method: http.MethodGet, path: "/api/v1/runner/planning-sessions/wait?hold_seconds=1"},
		{name: "spec", method: http.MethodPost, path: "/api/v1/runner/planning-sessions/specs", body: `{"body":"# Plan"}`},
		{name: "confidence", method: http.MethodPost, path: "/api/v1/runner/planning-sessions/confidence", body: `{"score":4,"summary":"Clear"}`},
		{name: "survey", method: http.MethodPost, path: "/api/v1/runner/planning-sessions/surveys", body: `{"questions":[{"prompt":"Priority?","options":["High","Low"]}]}`},
		{name: "agent message", method: http.MethodPost, path: "/api/v1/runner/planning-sessions/agent-messages", body: `{"text":"Ready."}`},
		{name: "activity", method: http.MethodPut, path: "/api/v1/runner/planning-sessions/activities/" + uuid.New().String(), body: `{}`},
	}
	for _, request := range requests {
		t.Run(request.name, func(t *testing.T) {
			req := httptest.NewRequest(request.method, request.path, bytes.NewBufferString(request.body))
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			server.Router.ServeHTTP(rec, req)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		})
	}
}

func TestRunnerPlanningSessionRejectsRunMismatch(t *testing.T) {
	r := support.Setup(t)
	server, signer := mustRunnerLiveLogServer(t, r)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "planning", "start")
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)
	token, err := runneraction.MintPlanningSessionToken(signer, runneraction.PlanningSessionScope{
		OrganizationID: session.OrganizationID,
		FactoryID:      session.FactoryID,
		SessionID:      session.ID,
		CanvasRunID:    uuid.New(),
	}, time.Hour)
	require.NoError(t, err)

	requests := []struct {
		path string
		body string
	}{
		{path: "/api/v1/runner/planning-sessions/specs", body: `{"body":"# Retry refunds"}`},
		{path: "/api/v1/runner/planning-sessions/agent-messages", body: `{"text":"Ready."}`},
	}
	for _, request := range requests {
		req := httptest.NewRequest(http.MethodPost, request.path, bytes.NewBufferString(request.body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestConsumeResolvedWaitTreatsDoubleConsumeAsMiss(t *testing.T) {
	r := support.Setup(t)
	_, session, _, _ := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	requireResolvedMessageWait(t, db, session)

	winner, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	loser, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	require.Equal(t, models.PlanningWaitResolved, winner.WaitState)
	require.Equal(t, models.PlanningWaitResolved, loser.WaitState)

	result, consumed, err := consumeResolvedWait(winner, db)
	require.NoError(t, err)
	assert.True(t, consumed)
	assert.Equal(t, models.PlanningWaitKindMessage, result.Kind)

	_, consumed, err = consumeResolvedWait(loser, db)
	require.NoError(t, err)
	assert.False(t, consumed)
}

func TestRunnerPlanningWaitDoubleConsumeReturnsPending(t *testing.T) {
	r := support.Setup(t)
	server, session, _, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	requireResolvedMessageWait(t, db, session)

	const waiters = 2
	recs := make([]*httptest.ResponseRecorder, waiters)
	var wg sync.WaitGroup
	wg.Add(waiters)
	for i := range recs {
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/planning-sessions/wait?hold_seconds=1", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			server.Router.ServeHTTP(rec, req)
			recs[i] = rec
		}(i)
	}
	wg.Wait()

	deliveredCount := 0
	for _, rec := range recs {
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		status, _ := body["status"].(string)
		require.Contains(t, []string{models.PlanningWaitKindMessage, "pending"}, status)
		if status == models.PlanningWaitKindMessage {
			deliveredCount++
		}
	}
	assert.Equal(t, 1, deliveredCount)
}

func TestRunnerPlanningWaitContextCancelReturnsPending(t *testing.T) {
	r := support.Setup(t)
	server, session, _, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/planning-sessions/wait?hold_seconds=60", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		server.Router.ServeHTTP(rec, req)
	}()

	requirePlanningWaitPending(t, db, session)
	cancel()
	<-done

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "pending", body["status"])
}

func TestRunnerPlanningWaitCancelDoesNotConsumeUserMessage(t *testing.T) {
	r := support.Setup(t)
	server, session, _, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	require.NoError(t, session.BeginWait(db))
	require.NoError(t, session.SendUserMessage(db, "hello", uuid.Nil))
	require.Equal(t, models.PlanningWaitResolved, session.WaitState)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/planning-sessions/wait?hold_seconds=60", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "pending", body["status"])

	held, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	require.Equal(t, models.PlanningWaitResolved, held.WaitState)

	live := httptest.NewRequest(http.MethodGet, "/api/v1/runner/planning-sessions/wait?hold_seconds=1", nil)
	live.Header.Set("Authorization", "Bearer "+token)
	liveRec := httptest.NewRecorder()
	server.Router.ServeHTTP(liveRec, live)
	require.Equal(t, http.StatusOK, liveRec.Code, liveRec.Body.String())
	var delivered map[string]any
	require.NoError(t, json.Unmarshal(liveRec.Body.Bytes(), &delivered))
	assert.Equal(t, models.PlanningWaitKindMessage, delivered["status"])
	assert.Equal(t, "hello", delivered["text"])
}

func TestRunnerPlanningWaitFailedWriteRestoresUserMessage(t *testing.T) {
	r := support.Setup(t)
	server, session, _, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	require.NoError(t, session.BeginWait(db))
	require.NoError(t, session.SendUserMessage(db, "hello", uuid.Nil))
	require.Equal(t, models.PlanningWaitResolved, session.WaitState)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/planning-sessions/wait?hold_seconds=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(failingResponseWriter{ResponseWriter: rec}, req)

	held, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	require.Equal(t, models.PlanningWaitResolved, held.WaitState)

	live := httptest.NewRequest(http.MethodGet, "/api/v1/runner/planning-sessions/wait?hold_seconds=1", nil)
	live.Header.Set("Authorization", "Bearer "+token)
	liveRec := httptest.NewRecorder()
	server.Router.ServeHTTP(liveRec, live)
	require.Equal(t, http.StatusOK, liveRec.Code, liveRec.Body.String())
	var delivered map[string]any
	require.NoError(t, json.Unmarshal(liveRec.Body.Bytes(), &delivered))
	assert.Equal(t, models.PlanningWaitKindMessage, delivered["status"])
	assert.Equal(t, "hello", delivered["text"])
}

func TestRunnerPlanningWaitMintsFileRefsInDeliveredText(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	server, session, _, token := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	require.NotNil(t, session.DraftWorkOrderID)

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: session.OrganizationID,
		FactoryID:      session.FactoryID,
		WorkOrderID:    *session.DraftWorkOrderID,
		Filename:       "shot.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(t.Context(), db, store, file, bytes.NewReader([]byte("png-bytes"))))

	message := "See ![shot.png](" + blob.FileRef(file.ID) + ")"
	require.NoError(t, session.BeginWait(db))
	require.NoError(t, session.SendUserMessage(db, message, uuid.Nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runner/planning-sessions/wait?hold_seconds=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var delivered map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &delivered))
	assert.Equal(t, models.PlanningWaitKindMessage, delivered["status"])
	text, _ := delivered["text"].(string)
	assert.Contains(t, text, "sp_file=1")
	assert.Contains(t, text, "/api/v1/public/files/"+file.ID.String())
	assert.NotContains(t, text, blob.FileRef(file.ID))

	messages, err := models.ListPlanningSessionMessages(db, session.ID)
	require.NoError(t, err)
	require.NotEmpty(t, messages)
	assert.Equal(t, message, messages[len(messages)-1].Text)
}

type failingResponseWriter struct {
	http.ResponseWriter
}

func (w failingResponseWriter) Write(p []byte) (int, error) {
	return 0, errors.New("client gone")
}

func mustPlanningRunnerSession(t *testing.T, r *support.ResourceRegistry) (*Server, *models.FactoryPlanningSession, *models.Factory, string) {
	t.Helper()
	server, signer := mustRunnerLiveLogServer(t, r)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "planning", "start")
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)
	return server, session, factoryModel, mustPlanningRunnerToken(t, signer, session)
}

func mustPlanningRunnerToken(t *testing.T, signer *jwt.Signer, session *models.FactoryPlanningSession) string {
	t.Helper()
	require.NotNil(t, session.CanvasRunID)
	token, err := runneraction.MintPlanningSessionToken(signer, runneraction.PlanningSessionScope{
		OrganizationID: session.OrganizationID,
		FactoryID:      session.FactoryID,
		SessionID:      session.ID,
		CanvasRunID:    *session.CanvasRunID,
	}, time.Hour)
	require.NoError(t, err)
	return token
}

func TestBeginPlanningWaitAndNotify_PublishesAfterWaitCommits(t *testing.T) {
	r := support.Setup(t)
	_, session, _, _ := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	require.NoError(t, session.ProposeSurvey(db, models.PlanningSessionSurvey{
		Questions: []models.PlanningSessionSurveyQuestion{
			{Prompt: "Which service owns retries?", Options: []string{"Payments", "Billing"}},
		},
	}))

	published := []messages.FactoryWorkOrderNotificationMessage{}
	restore := messages.SetWorkOrderNotificationPublisherForTest(func(message messages.FactoryWorkOrderNotificationMessage) error {
		reloaded, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
		require.NoError(t, err)
		assert.Equal(t, models.PlanningWaitPending, reloaded.WaitState)
		published = append(published, message)
		return nil
	})
	defer restore()

	require.NoError(t, beginPlanningWaitAndNotify(db, session))
	require.Len(t, published, 1)
	assert.Equal(t, factoryevents.EventTypeOrderAgentQuestion, published[0].EventType)
	assert.Equal(t, session.DraftWorkOrderID.String(), published[0].OrderID)
	assert.Equal(t, "Which service owns retries?", published[0].QuestionPrompt)
	assert.Empty(t, published[0].ActorUserID)

	require.NoError(t, beginPlanningWaitAndNotify(db, session))
	require.Len(t, published, 1)
}

func TestBeginPlanningWaitAndNotify_SkipsSessionWithoutTask(t *testing.T) {
	r := support.Setup(t)
	_, session, _, _ := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	require.NoError(t, session.ProposeSurvey(db, models.PlanningSessionSurvey{
		Questions: []models.PlanningSessionSurveyQuestion{
			{Prompt: "Which service owns retries?", Options: []string{"Payments", "Billing"}},
		},
	}))
	session.DraftWorkOrderID = nil
	session.WaitWorkOrderID = nil
	require.NoError(t, db.Model(session).Updates(map[string]any{
		"draft_work_order_id": nil,
		"wait_work_order_id":  nil,
	}).Error)

	published := []messages.FactoryWorkOrderNotificationMessage{}
	restore := messages.SetWorkOrderNotificationPublisherForTest(func(message messages.FactoryWorkOrderNotificationMessage) error {
		published = append(published, message)
		return nil
	})
	defer restore()

	require.NoError(t, beginPlanningWaitAndNotify(db, session))
	assert.Empty(t, published)
}

func TestBeginPlanningWaitAndNotify_SkipsFollowUpWaitWithoutQuestion(t *testing.T) {
	r := support.Setup(t)
	_, session, _, _ := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())

	published := []messages.FactoryWorkOrderNotificationMessage{}
	restore := messages.SetWorkOrderNotificationPublisherForTest(func(message messages.FactoryWorkOrderNotificationMessage) error {
		published = append(published, message)
		return nil
	})
	defer restore()

	require.NoError(t, beginPlanningWaitAndNotify(db, session))
	assert.Empty(t, published)
	assert.Equal(t, models.PlanningWaitPending, session.WaitState)
}

func TestProposePlanningSpecAndNotify_PublishesOnceWhenPlanBecomesReady(t *testing.T) {
	r := support.Setup(t)
	_, session, factoryModel, _ := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	require.NotNil(t, session.DraftWorkOrderID)
	orderID := *session.DraftWorkOrderID

	published := []messages.FactoryWorkOrderNotificationMessage{}
	restore := messages.SetWorkOrderNotificationPublisherForTest(func(message messages.FactoryWorkOrderNotificationMessage) error {
		published = append(published, message)
		return nil
	})
	defer restore()

	require.NoError(t, proposePlanningSpecAndNotify(db, session, "# Retry refunds\n\nStop double charges.\n"))
	require.Len(t, published, 1)
	assert.Equal(t, factoryevents.EventTypeOrderPlanReady, published[0].EventType)
	assert.Equal(t, orderID.String(), published[0].OrderID)
	assert.Equal(t, factoryModel.ID.String(), published[0].FactoryID)
	assert.Empty(t, published[0].ActorUserID)

	require.NoError(t, proposePlanningSpecAndNotify(db, session, "# Retry refunds\n\nStop double charges again.\n"))
	require.Len(t, published, 1)

	require.NoError(t, session.End(db))
	require.Len(t, published, 1)
}

func TestProposePlanningSpecAndNotify_SilentWithoutTask(t *testing.T) {
	r := support.Setup(t)
	_, session, _, _ := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	session.DraftWorkOrderID = nil
	session.WaitWorkOrderID = nil
	require.NoError(t, db.Model(session).Updates(map[string]any{
		"draft_work_order_id": nil,
		"wait_work_order_id":  nil,
	}).Error)

	published := []messages.FactoryWorkOrderNotificationMessage{}
	restore := messages.SetWorkOrderNotificationPublisherForTest(func(message messages.FactoryWorkOrderNotificationMessage) error {
		published = append(published, message)
		return nil
	})
	defer restore()

	err := proposePlanningSpecAndNotify(db, session, "# Retry refunds\n\nStop double charges.\n")
	require.Error(t, err)
	assert.Empty(t, published)
}

func requireResolvedMessageWait(t *testing.T, db *gorm.DB, session *models.FactoryPlanningSession) {
	t.Helper()
	require.NoError(t, session.BeginWait(db))
	require.NoError(t, session.SendUserMessage(db, "Add refund retries.", uuid.Nil))
	require.Equal(t, models.PlanningWaitResolved, session.WaitState)
}

func requirePlanningWaitPending(t *testing.T, db *gorm.DB, session *models.FactoryPlanningSession) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		reloaded, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
		require.NoError(t, err)
		if reloaded.WaitState == models.PlanningWaitPending {
			return
		}
		require.True(t, time.Now().Before(deadline), "planning wait did not become pending")
		time.Sleep(20 * time.Millisecond)
	}
}
