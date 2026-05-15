package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/superplane/runner/shared/api"
	sharedmodels "github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/shared/webhook"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	taskstore "github.com/superplane/runner/task-broker/internal/store"
)

// Server implements task-broker HTTP handlers.
type Server struct {
	Store     taskstore.Store
	PublicURL string // reachable base URL fleet-manager uses to call webhookComplete
	Webhook   *webhook.Sender
	Log       *slog.Logger
	HTTP      *http.Client
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) registerFleet(w http.ResponseWriter, r *http.Request) {
	var req api.RegisterFleetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.ID = strings.TrimSpace(req.ID)
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	if req.BaseURL == "" {
		writeError(w, http.StatusBadRequest, "base_url required")
		return
	}
	f := &brokermodels.Fleet{
		ID:        req.ID,
		BaseURL:   strings.TrimRight(req.BaseURL, "/"),
		AuthToken: strings.TrimSpace(req.AuthToken),
		Labels:    taskstore.NormalizeLabels(req.Labels),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.Store.CreateFleet(r.Context(), f); err != nil {
		s.logErr("create fleet", err)
		writeError(w, http.StatusInternalServerError, "could not persist fleet")
		return
	}
	writeJSON(w, http.StatusCreated, fleetToResponse(f))
}

func (s *Server) listFleets(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListFleets(r.Context())
	if err != nil {
		s.logErr("list fleets", err)
		writeError(w, http.StatusInternalServerError, "could not list fleets")
		return
	}
	out := make([]api.FleetResponse, 0, len(list))
	for i := range list {
		out = append(out, *fleetToResponse(&list[i]))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) deleteFleet(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	if err := s.Store.DeleteFleet(r.Context(), id); err != nil {
		s.logErr("delete fleet", err)
		writeError(w, http.StatusInternalServerError, "could not delete fleet")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func fleetToResponse(f *brokermodels.Fleet) *api.FleetResponse {
	if f == nil {
		return nil
	}
	return &api.FleetResponse{
		ID:        f.ID,
		BaseURL:   f.BaseURL,
		Labels:    append([]string(nil), f.Labels...),
		CreatedAt: f.CreatedAt.Unix(),
	}
}

func (s *Server) createBrokerTask(w http.ResponseWriter, r *http.Request) {
	public := strings.TrimRight(strings.TrimSpace(s.PublicURL), "/")
	if public == "" {
		writeError(w, http.StatusInternalServerError, "broker public URL not configured")
		return
	}

	var req api.BrokerCreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.WebhookURL) == "" {
		writeError(w, http.StatusBadRequest, "webhook_url required")
		return
	}
	hasID := strings.TrimSpace(req.FleetID) != ""
	hasLabels := len(taskstore.NormalizeLabels(req.FleetLabels)) > 0
	switch {
	case hasID && hasLabels:
		writeError(w, http.StatusBadRequest, "specify either fleet_id or fleet_labels, not both")
		return
	case !hasID && !hasLabels:
		writeError(w, http.StatusBadRequest, "fleet_id or fleet_labels required")
		return
	}

	ctx := r.Context()
	var fleet *brokermodels.Fleet
	var err error
	if hasID {
		fleet, err = s.Store.GetFleet(ctx, strings.TrimSpace(req.FleetID))
		if err != nil {
			s.logErr("get fleet", err)
			writeError(w, http.StatusInternalServerError, "could not load fleet")
			return
		}
		if fleet == nil {
			writeError(w, http.StatusNotFound, "fleet not found")
			return
		}
	} else {
		fleet, err = s.Store.FindFleetByLabels(ctx, req.FleetLabels)
		if err != nil {
			s.logErr("find fleet by labels", err)
			writeError(w, http.StatusInternalServerError, "could not route fleet")
			return
		}
		if fleet == nil {
			writeError(w, http.StatusNotFound, "no fleet matches fleet_labels")
			return
		}
	}

	if msg := validateCreateTaskPayload(&req.CreateTaskRequest); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	brokerID := uuid.NewString()
	row := &brokermodels.BrokerTask{
		ID:               brokerID,
		FleetID:          fleet.ID,
		FleetTaskID:      "",
		CallerWebhookURL: strings.TrimSpace(req.WebhookURL),
		CreatedAt:        time.Now().UTC(),
	}
	if err := s.Store.InsertBrokerTask(ctx, row); err != nil {
		s.logErr("insert broker task", err)
		writeError(w, http.StatusInternalServerError, "could not create broker task")
		return
	}

	upstreamWebhook := public + "/v1/webhooks/complete/" + brokerID
	up := api.CreateTaskRequest{
		Command:                 append([]string(nil), req.Command...),
		Commands:                append([]string(nil), req.Commands...),
		WebhookURL:              upstreamWebhook,
		ExecutionMode:           req.ExecutionMode,
		DockerImage:             req.DockerImage,
		ExecutionTimeoutSeconds: cloneIntPtr(req.ExecutionTimeoutSeconds),
	}

	payload, err := json.Marshal(up)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "marshal upstream")
		return
	}

	t0 := time.Now()
	fleetTaskID, st, respBody := s.forwardCreateTask(ctx, fleet, payload)
	upstreamDur := time.Since(t0)
	if st != http.StatusCreated {
		_ = s.Store.DeleteBrokerTask(ctx, brokerID)
		s.warn("upstream create failed",
			slog.Int("status", st),
			slog.String("fleet", fleet.ID),
			slog.Duration("dur", upstreamDur),
			slog.String("upstream_host", upstreamBaseHost(fleet.BaseURL)),
			slog.String("body", strings.TrimSpace(string(respBody))))
		writeError(w, http.StatusBadGateway, "fleet-manager rejected task")
		return
	}
	if fleetTaskID == "" {
		_ = s.Store.DeleteBrokerTask(ctx, brokerID)
		writeError(w, http.StatusBadGateway, "fleet-manager missing task id")
		return
	}
	if err := s.Store.UpdateBrokerTaskFleetTaskID(ctx, brokerID, fleetTaskID); err != nil {
		s.logErr("record fleet task id", err)
		writeError(w, http.StatusInternalServerError, "could not correlate task")
		return
	}

	if s.Log != nil {
		s.Log.Info("fleet_upstream_http",
			slog.String("op", "create_task"),
			slog.Int("http_status", st),
			slog.Duration("dur", upstreamDur),
			slog.String("broker_task_id", brokerID),
			slog.String("fleet_id", fleet.ID),
			slog.String("fleet_task_id", fleetTaskID),
			slog.String("upstream_host", upstreamBaseHost(fleet.BaseURL)),
		)
	}

	writeJSON(w, http.StatusCreated, api.BrokerCreateTaskResponse{ID: brokerID})
}

func (s *Server) getBrokerTask(w http.ResponseWriter, r *http.Request) {
	brokerID := strings.TrimSpace(chi.URLParam(r, "id"))
	if brokerID == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	row, err := s.Store.GetBrokerTask(r.Context(), brokerID)
	if err != nil {
		s.logErr("get broker task", err)
		writeError(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	if row == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if row.FleetTaskID == "" {
		writeJSON(w, http.StatusOK, api.BrokerGetTaskResponse{
			TaskID: brokerID,
			Status: "pending",
		})
		return
	}
	fleet, err := s.Store.GetFleet(r.Context(), row.FleetID)
	if err != nil {
		s.logErr("get fleet for task poll", err)
		writeError(w, http.StatusInternalServerError, "could not load fleet")
		return
	}
	if fleet == nil {
		writeError(w, http.StatusInternalServerError, "fleet missing")
		return
	}
	st, upstream := s.forwardGetTask(r.Context(), fleet, row.FleetTaskID)
	if st == http.StatusNotFound {
		writeError(w, http.StatusBadGateway, "upstream task missing")
		return
	}
	if st != http.StatusOK {
		s.warn("upstream get task failed", slog.Int("status", st), slog.String("fleet", fleet.ID))
		writeError(w, http.StatusBadGateway, "fleet-manager rejected status request")
		return
	}
	up, taskLog, err := parseUpstreamTaskLog(upstream)
	if err != nil {
		writeError(w, http.StatusBadGateway, "invalid upstream response")
		return
	}
	writeJSON(w, http.StatusOK, api.BrokerGetTaskResponse{
		TaskID:                  brokerID,
		FleetTaskID:             row.FleetTaskID,
		Status:                  strings.TrimSpace(up.Status),
		ExitCode:                up.ExitCode,
		Output:                  up.Output,
		Error:                   up.Error,
		CancelRequested:         up.CancelRequested,
		CloudWatchLogGroup:      up.CloudWatchLogGroup,
		CloudWatchLogStream:     up.CloudWatchLogStream,
		TaskLog:                 taskLog,
		ExecutionTimeoutSeconds: up.ExecutionTimeoutSeconds,
	})
}

// parseUpstreamTaskLog unmarshals fleet-manager GET /v1/tasks/{id} JSON and derives task_log
// (including legacy cloudwatch_log_group / cloudwatch_log_stream fields).
func parseUpstreamTaskLog(upstream []byte) (api.TaskStatusResponse, *api.TaskLogSink, error) {
	var up api.TaskStatusResponse
	if err := json.Unmarshal(upstream, &up); err != nil {
		return up, nil, err
	}
	taskLog := up.TaskLog
	if taskLog == nil && strings.TrimSpace(up.CloudWatchLogGroup) != "" && strings.TrimSpace(up.CloudWatchLogStream) != "" {
		taskLog = api.TaskLogSinkCloudWatchFromParts(up.CloudWatchLogGroup, up.CloudWatchLogStream, "")
	}
	return up, taskLog, nil
}

func (s *Server) cancelBrokerTask(w http.ResponseWriter, r *http.Request) {
	brokerID := strings.TrimSpace(chi.URLParam(r, "id"))
	if brokerID == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	row, err := s.Store.GetBrokerTask(r.Context(), brokerID)
	if err != nil {
		s.logErr("get broker task", err)
		writeError(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	if row == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if row.FleetTaskID == "" {
		writeError(w, http.StatusConflict, "task not yet assigned upstream")
		return
	}
	fleet, err := s.Store.GetFleet(r.Context(), row.FleetID)
	if err != nil {
		s.logErr("get fleet for cancel", err)
		writeError(w, http.StatusInternalServerError, "could not load fleet")
		return
	}
	if fleet == nil {
		writeError(w, http.StatusInternalServerError, "fleet missing")
		return
	}
	st, upstream := s.forwardCancelTask(r.Context(), fleet, row.FleetTaskID)
	if st == http.StatusNotFound {
		writeError(w, http.StatusBadGateway, "upstream task missing")
		return
	}
	if st != http.StatusOK {
		s.warn("upstream cancel failed", slog.Int("status", st), slog.String("fleet", fleet.ID))
		writeError(w, http.StatusBadGateway, "fleet-manager rejected cancel")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(upstream)
}

func (s *Server) forwardCancelTask(ctx context.Context, fleet *brokermodels.Fleet, fleetTaskID string) (status int, respBody []byte) {
	c := s.HTTP
	if c == nil {
		c = http.DefaultClient
	}
	u := strings.TrimRight(fleet.BaseURL, "/") + "/v1/tasks/" + fleetTaskID + "/cancel"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return 0, nil
	}
	if t := fleet.AuthToken; t != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(t))
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, []byte(err.Error())
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	return resp.StatusCode, b
}

func (s *Server) forwardGetTask(ctx context.Context, fleet *brokermodels.Fleet, fleetTaskID string) (status int, respBody []byte) {
	c := s.HTTP
	if c == nil {
		c = http.DefaultClient
	}
	u := fleet.BaseURL + "/v1/tasks/" + fleetTaskID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, nil
	}
	if t := fleet.AuthToken; t != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(t))
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, []byte(err.Error())
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	return resp.StatusCode, b
}

func validateCreateTaskPayload(req *api.CreateTaskRequest) string {
	hasArgv := len(req.Command) > 0
	hasCmds := false
	for _, c := range req.Commands {
		if strings.TrimSpace(c) != "" {
			hasCmds = true
			break
		}
	}
	switch {
	case hasArgv && hasCmds:
		return "specify either command or commands, not both"
	case !hasArgv && !hasCmds:
		return "command or commands required"
	}
	mode := sharedmodels.ExecutionMode(strings.ToLower(strings.TrimSpace(req.ExecutionMode)))
	switch mode {
	case "", sharedmodels.ExecutionHost:
	case sharedmodels.ExecutionDocker:
		if strings.TrimSpace(req.DockerImage) == "" {
			return "docker_image required for docker execution_mode"
		}
	default:
		return "invalid execution_mode"
	}
	if msg := api.ValidateExecutionTimeoutSeconds(req.ExecutionTimeoutSeconds); msg != "" {
		return msg
	}
	return ""
}

func cloneIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func (s *Server) forwardCreateTask(ctx context.Context, fleet *brokermodels.Fleet, body []byte) (fleetTaskID string, status int, respBody []byte) {
	c := s.HTTP
	if c == nil {
		c = http.DefaultClient
	}
	u := fleet.BaseURL + "/v1/tasks"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return "", 0, nil
	}
	req.Header.Set("Content-Type", "application/json")
	if t := fleet.AuthToken; t != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(t))
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", 0, []byte(err.Error())
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode != http.StatusCreated {
		return "", resp.StatusCode, b
	}
	var parsed api.CreateTaskResponse
	if err := json.Unmarshal(b, &parsed); err != nil {
		return "", resp.StatusCode, b
	}
	return parsed.ID, resp.StatusCode, b
}

func (s *Server) webhookComplete(w http.ResponseWriter, r *http.Request) {
	brokerID := strings.TrimSpace(chi.URLParam(r, "brokerTaskID"))
	if brokerID == "" {
		writeError(w, http.StatusBadRequest, "missing broker task id")
		return
	}
	var upstream api.WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&upstream); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	row, err := s.Store.GetBrokerTask(r.Context(), brokerID)
	if err != nil {
		s.logErr("get broker task", err)
		writeError(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	if row == nil {
		writeError(w, http.StatusNotFound, "unknown broker task")
		return
	}
	if row.FleetTaskID == "" || row.FleetTaskID != upstream.TaskID {
		writeError(w, http.StatusForbidden, "task id mismatch")
		return
	}

	out := api.WebhookPayload{
		TaskID:              brokerID,
		FleetTaskID:         upstream.TaskID,
		Status:              upstream.Status,
		ExitCode:            upstream.ExitCode,
		Output:              upstream.Output,
		Error:               upstream.Error,
		CloudWatchLogGroup:  upstream.CloudWatchLogGroup,
		CloudWatchLogStream: upstream.CloudWatchLogStream,
		TaskLog:             upstream.TaskLog,
	}
	if out.TaskLog == nil && strings.TrimSpace(upstream.CloudWatchLogGroup) != "" && strings.TrimSpace(upstream.CloudWatchLogStream) != "" {
		out.TaskLog = api.TaskLogSinkCloudWatchFromParts(upstream.CloudWatchLogGroup, upstream.CloudWatchLogStream, "")
	}

	ws := s.Webhook
	if ws == nil {
		ws = webhook.DefaultSender()
	}
	if ws.Log == nil {
		ws.Log = s.Log
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := ws.Deliver(ctx, row.CallerWebhookURL, out); err != nil {
		s.warn("caller webhook delivery failed", slog.String("broker_task_id", brokerID), slog.Any("err", err))
		writeError(w, http.StatusBadGateway, "caller webhook delivery failed")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) logErr(msg string, err error) {
	if s.Log != nil {
		s.Log.Error(msg, slog.Any("err", err))
	}
}

func (s *Server) warn(msg string, attrs ...any) {
	if s.Log != nil {
		s.Log.Warn(msg, attrs...)
	}
}

func upstreamBaseHost(base string) string {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}
