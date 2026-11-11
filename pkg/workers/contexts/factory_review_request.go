package contexts

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
)

type reviewRequestReviewer struct {
	ID     int64  `json:"id"`
	Login  string `json:"login"`
	Reason string `json:"reason"`
}

type reviewRequestDecision struct {
	Error          *string                 `json:"error,omitempty"`
	ReviewRequired *bool                   `json:"reviewRequired"`
	Rationale      string                  `json:"rationale"`
	Reviewers      []reviewRequestReviewer `json:"reviewers"`
	NoCandidate    *bool                   `json:"noCandidate"`
	ReportID       string                  `json:"reportId"`
	SourceAppID    string                  `json:"sourceAppId"`
	ReviewedCommit string                  `json:"reviewedCommit"`
}

// Only this installed automation consumes completion messages. Ordinary PR
// events and broadcasts cannot start reviewer selection.
func (s *EventContext) reviewRequestEvent(payloadType string, payload any) (any, string, bool, error) {
	template, _ := s.node.Metadata.Data()[models.FactoryAppTemplateMetadataKey].(map[string]any)
	if template["id"] != "review-request" {
		return payload, "", false, nil
	}
	if s.run != nil || s.node.ComponentName() != "onBroadcast" || payloadType != "app.broadcast" {
		return nil, "", true, nil
	}
	if s.node.Configuration.Data()["enabled"] == false {
		return nil, "", true, nil
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(s.tx, s.node.WorkflowID)
	if err != nil {
		return nil, "", false, err
	}
	source, err := models.ValidateReviewRequestAdmission(s.tx, canvas, s.node, payload)
	if err != nil {
		if errors.Is(err, models.ErrReviewRequestSourceInvalid) || errors.Is(err, models.ErrReviewRequestSuperseded) {
			return nil, "", true, nil
		}
		return nil, "", false, err
	}
	message := maps.Clone(source.Message)
	root, err := source.Run.RootEvent(s.tx)
	if err != nil {
		return nil, "", false, err
	}
	producer, _ := models.RootEventSourcePayload(root.Data.Data()).(map[string]any)
	for _, key := range []string{"repository", "pull_request", "workOrder"} {
		if value, ok := producer[key]; ok {
			message[key] = value
		}
	}
	message["reviewRequestSource"] = message["payload"]
	message["pullRequest"] = map[string]any{"id": source.PullRequest.ID.String()}
	return message, "review-request:" + source.App.ID.String() + ":" + source.Run.ID.String(), false, nil
}

func (s *ExecutionStateContext) prepareReviewRequestOutput(channel, payloadType string, payloads []any) ([]any, error) {
	if s.execution.NodeID != "select-reviewers" || !strings.HasSuffix(payloadType, ".finished") {
		return payloads, nil
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(s.tx, s.execution.WorkflowID)
	if err != nil {
		return nil, err
	}
	installed, err := models.CanvasHasFactoryAppTemplate(s.tx, canvas.ID, "review-request")
	if err != nil || !installed {
		return payloads, err
	}
	if s.IsFinished() || s.IsCancelling() {
		return payloads, nil
	}
	if len(payloads) != 1 {
		return nil, fmt.Errorf("Review request agent must return one decision")
	}
	output, ok := payloads[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Review request agent output must be an object")
	}
	if channel != "passed" {
		message, _ := output["error"].(string)
		if strings.TrimSpace(message) != "" {
			return nil, fmt.Errorf("Review request agent failed: %s", message)
		}
		return nil, fmt.Errorf("Review request agent failed")
	}
	decision, err := decodeReviewRequestDecision(output["result"])
	if err != nil {
		return nil, err
	}
	var candidates []string
	encoded, err := json.Marshal(s.execution.Metadata.Data()["reviewRequestCandidates"])
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(encoded, &candidates); err != nil {
		return nil, fmt.Errorf("Review request candidate discovery is missing: %w", err)
	}
	if s.execution.Metadata.Data()["reviewRequestDiscovered"] != true {
		return nil, fmt.Errorf("Review request candidate discovery did not succeed")
	}
	var identities map[string]int64
	encoded, err = json.Marshal(s.execution.Metadata.Data()["reviewRequestCandidateIDs"])
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(encoded, &identities); err != nil {
		return nil, err
	}
	for i, reviewer := range decision.Reviewers {
		if !slices.ContainsFunc(candidates, func(login string) bool { return strings.EqualFold(login, reviewer.Login) }) {
			return nil, fmt.Errorf("Review request selected an ineligible reviewer: %s", reviewer.Login)
		}
		decision.Reviewers[i].ID = identities[strings.ToLower(reviewer.Login)]
	}
	if *decision.NoCandidate {
		fallback, _ := s.execution.Configuration.Data()["fallbackReviewer"].(string)
		if fallback != "" && slices.ContainsFunc(candidates, func(login string) bool { return strings.EqualFold(login, fallback) }) {
			decision.Reviewers = []reviewRequestReviewer{{ID: identities[strings.ToLower(fallback)], Login: fallback, Reason: "Use the configured fallback because selection found no suitable candidate."}}
		} else {
			*decision.ReviewRequired = false
			decision.Rationale += " No eligible candidate or fallback reviewer is available."
		}
	}
	run, err := models.FindCanvasRunInTransaction(s.tx, canvas.ID, s.execution.RunID)
	if err != nil {
		return nil, err
	}
	source, err := models.CheckReviewRequestAttempt(s.tx, canvas, run)
	if err != nil {
		return nil, err
	}
	if !*decision.ReviewRequired && !*decision.NoCandidate && reviewRequestNeedsReview(source.Report) {
		return nil, fmt.Errorf("Review request requires review for concerning or uncertain findings")
	}
	decision.ReportID = source.Run.ID.String()
	decision.SourceAppID = source.App.ID.String()
	decision.ReviewedCommit, _ = source.Report["reviewedCommit"].(string)
	cloned := maps.Clone(output)
	cloned["reviewRequest"] = decision
	return []any{cloned}, nil
}

func decodeReviewRequestDecision(raw any) (*reviewRequestDecision, error) {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var decision reviewRequestDecision
	if err := json.Unmarshal(encoded, &decision); err != nil {
		return nil, fmt.Errorf("Invalid Review request decision: %w", err)
	}
	if decision.Error != nil {
		return nil, fmt.Errorf("Review request selection failed: %s", *decision.Error)
	}
	if decision.ReviewRequired == nil || decision.NoCandidate == nil || strings.TrimSpace(decision.Rationale) == "" || decision.Reviewers == nil {
		return nil, fmt.Errorf("Review request decision requires reviewRequired, noCandidate, rationale, and reviewers")
	}
	if (!*decision.ReviewRequired && (len(decision.Reviewers) != 0 || *decision.NoCandidate)) || (*decision.NoCandidate && len(decision.Reviewers) != 0) || (*decision.ReviewRequired && !*decision.NoCandidate && len(decision.Reviewers) == 0) {
		return nil, fmt.Errorf("Review request decision contradicts its reviewer selection")
	}
	seen := map[string]bool{}
	for i := range decision.Reviewers {
		reviewer := &decision.Reviewers[i]
		reviewer.Login = strings.TrimSpace(reviewer.Login)
		login := strings.ToLower(reviewer.Login)
		if login == "" || strings.TrimSpace(reviewer.Reason) == "" || seen[login] {
			return nil, fmt.Errorf("Review request reviewers require distinct logins and reasons")
		}
		seen[login] = true
	}
	return &decision, nil
}

func reviewRequestNeedsReview(report map[string]any) bool {
	checks, _ := report["checks"].([]any)
	if len(checks) == 0 {
		return true
	}
	return slices.ContainsFunc(checks, func(raw any) bool {
		check, _ := raw.(map[string]any)
		return check["level"] != "positive"
	})
}
