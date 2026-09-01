package public

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestCreateRunnerWorkOrderSurvey(t *testing.T) {
	r := support.Setup(t)
	server, signer := mustRunnerLiveLogServer(t, r)
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	scope := runneraction.WorkOrderSurveyScope{
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		CanvasRunID:    uuid.New(),
	}
	token, err := runneraction.MintWorkOrderSurveyToken(signer, scope, time.Hour)
	require.NoError(t, err)

	body := []byte(`{"timeout_seconds":120,"questions":[{"id":"scope","prompt":"Where?","options":["A","B"]}]}`)
	rec := postRunnerWorkOrderSurvey(t, server, token, body)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var created runnerWorkOrderSurveyResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, models.FactoryWorkOrderSurveyPending, created.Status)
	require.NotEmpty(t, created.ID)

	again := postRunnerWorkOrderSurvey(t, server, token, []byte(`{"questions":[{"id":"other","prompt":"Ignored"}]}`))
	require.Equal(t, http.StatusOK, again.Code, again.Body.String())
	var same runnerWorkOrderSurveyResponse
	require.NoError(t, json.Unmarshal(again.Body.Bytes(), &same))
	assert.Equal(t, created.ID, same.ID)
}

func TestCreateRunnerWorkOrderSurveyRejectsOtherRunPending(t *testing.T) {
	r := support.Setup(t)
	server, signer := mustRunnerLiveLogServer(t, r)
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	_, _, err = order.CreateSurvey(db, models.FactoryWorkOrderSurveyParams{
		CanvasRunID: uuid.New(),
		Questions:   []models.WorkOrderSurveyQuestion{{ID: "scope", Prompt: "Where?"}},
	})
	require.NoError(t, err)

	token, err := runneraction.MintWorkOrderSurveyToken(signer, runneraction.WorkOrderSurveyScope{
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		CanvasRunID:    uuid.New(),
	}, time.Hour)
	require.NoError(t, err)

	rec := postRunnerWorkOrderSurvey(t, server, token, []byte(`{"questions":[{"id":"x","prompt":"Nope"}]}`))
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestCreateRunnerWorkOrderSurveyRejectsMissingToken(t *testing.T) {
	r := support.Setup(t)
	server, _ := mustRunnerLiveLogServer(t, r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runner/work-order-surveys", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWaitRunnerWorkOrderSurveyReturnsOnAnswer(t *testing.T) {
	r := support.Setup(t)
	server, signer := mustRunnerLiveLogServer(t, r)
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	scope := runneraction.WorkOrderSurveyScope{
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		CanvasRunID:    uuid.New(),
	}
	survey, _, err := order.CreateSurvey(db, models.FactoryWorkOrderSurveyParams{
		CanvasRunID: scope.CanvasRunID,
		Questions:   []models.WorkOrderSurveyQuestion{{ID: "scope", Prompt: "Where?"}},
	})
	require.NoError(t, err)

	token, err := runneraction.MintWorkOrderSurveyToken(signer, scope, time.Hour)
	require.NoError(t, err)

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		done <- waitRunnerWorkOrderSurvey(t, server, token, survey.ID, 20)
	}()

	time.Sleep(50 * time.Millisecond)
	require.NoError(t, survey.Answer(db, r.User, []models.WorkOrderSurveyAnswer{{ID: "scope", Value: "A"}}))
	runneraction.NotifyWorkOrderSurvey(survey.ID)

	select {
	case rec := <-done:
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var got runnerWorkOrderSurveyResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, models.FactoryWorkOrderSurveyAnswered, got.Status)
		require.Len(t, got.Answers, 1)
		assert.Equal(t, "A", got.Answers[0].Value)
	case <-time.After(3 * time.Second):
		t.Fatal("wait did not return after the survey was answered")
	}
}

func postRunnerWorkOrderSurvey(t *testing.T, server *Server, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runner/work-order-surveys", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	return rec
}

func waitRunnerWorkOrderSurvey(
	t *testing.T,
	server *Server,
	token string,
	surveyID uuid.UUID,
	hold int,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/v1/runner/work-order-surveys/%s/wait?hold_seconds=%d", surveyID, hold),
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	return rec
}

func TestMintWorkOrderSurveyTokenIsNotAUserSession(t *testing.T) {
	r := support.Setup(t)
	_, signer := mustRunnerLiveLogServer(t, r)

	token, err := runneraction.MintWorkOrderSurveyToken(signer, runneraction.WorkOrderSurveyScope{
		OrganizationID: r.Organization.ID,
		FactoryID:      uuid.New(),
		WorkOrderID:    uuid.New(),
		CanvasRunID:    uuid.New(),
	}, time.Hour)
	require.NoError(t, err)

	scope, err := runneraction.ParseWorkOrderSurveyToken(signer, token)
	require.NoError(t, err)
	assert.Equal(t, r.Organization.ID, scope.OrganizationID)
}
