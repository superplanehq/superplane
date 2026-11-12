package reviewrequest

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/factories/lifecycle"
	"github.com/superplanehq/superplane/pkg/models"
)

type selectedReviewer struct {
	ID     int64  `json:"id"`
	Login  string `json:"login"`
	Reason string `json:"reason"`
}

type selectionDecision struct {
	Error          *string            `json:"error,omitempty"`
	ReviewRequired *bool              `json:"reviewRequired"`
	Rationale      string             `json:"rationale"`
	Reviewers      []selectedReviewer `json:"reviewers"`
	NoCandidate    *bool              `json:"noCandidate"`
	ReportID       string             `json:"reportId"`
	SourceAppID    string             `json:"sourceAppId"`
	ReviewedCommit string             `json:"reviewedCommit"`
}

// prepareOutput validates the selection decision against the discovered
// candidates and the source report before the decision is saved.
func prepareOutput(output lifecycle.Output) ([]any, error) {
	execution, payloads := output.Execution, output.Payloads
	if execution.NodeID != selectionNodeID || !strings.HasSuffix(output.PayloadType, ".finished") {
		return payloads, nil
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(output.Tx, execution.WorkflowID)
	if err != nil {
		return nil, err
	}
	exists, err := installed(output.Tx, canvas)
	if err != nil || !exists {
		return payloads, err
	}
	if execution.State == models.CanvasNodeExecutionStateFinished || execution.State == models.CanvasNodeExecutionStateCancelling {
		return payloads, nil
	}
	if len(payloads) != 1 {
		return nil, fmt.Errorf("Review request agent must return one decision")
	}
	result, ok := payloads[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Review request agent output must be an object")
	}
	if output.Channel != "passed" {
		message, _ := result["error"].(string)
		if strings.TrimSpace(message) != "" {
			return nil, fmt.Errorf("Review request agent failed: %s", message)
		}
		return nil, fmt.Errorf("Review request agent failed")
	}
	decision, err := decodeDecision(result["result"])
	if err != nil {
		return nil, err
	}
	var candidates []string
	encoded, err := json.Marshal(execution.Metadata.Data()["reviewRequestCandidates"])
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(encoded, &candidates); err != nil {
		return nil, fmt.Errorf("Review request candidate discovery is missing: %w", err)
	}
	if execution.Metadata.Data()["reviewRequestDiscovered"] != true {
		return nil, fmt.Errorf("Review request candidate discovery did not succeed")
	}
	var identities map[string]int64
	encoded, err = json.Marshal(execution.Metadata.Data()["reviewRequestCandidateIDs"])
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
		fallback, _ := execution.Configuration.Data()["fallbackReviewer"].(string)
		if fallback != "" && slices.ContainsFunc(candidates, func(login string) bool { return strings.EqualFold(login, fallback) }) {
			decision.Reviewers = []selectedReviewer{{ID: identities[strings.ToLower(fallback)], Login: fallback, Reason: "Use the configured fallback because selection found no suitable candidate."}}
		} else {
			*decision.ReviewRequired = false
			decision.Rationale += " No eligible candidate or fallback reviewer is available."
		}
	}
	run, err := models.FindCanvasRunInTransaction(output.Tx, canvas.ID, execution.RunID)
	if err != nil {
		return nil, err
	}
	source, err := models.CheckReviewRequestAttempt(output.Tx, canvas, run)
	if err != nil {
		return nil, err
	}
	if !*decision.ReviewRequired && !*decision.NoCandidate && needsReview(source.Report) {
		return nil, fmt.Errorf("Review request requires review for concerning or uncertain findings")
	}
	decision.ReportID = source.Run.ID.String()
	decision.SourceAppID = source.App.ID.String()
	decision.ReviewedCommit, _ = source.Report["reviewedCommit"].(string)
	cloned := maps.Clone(result)
	cloned["reviewRequest"] = decision
	return []any{cloned}, nil
}

func decodeDecision(raw any) (*selectionDecision, error) {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var decision selectionDecision
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

func needsReview(report map[string]any) bool {
	checks, _ := report["checks"].([]any)
	if len(checks) == 0 {
		return true
	}
	return slices.ContainsFunc(checks, func(raw any) bool {
		check, _ := raw.(map[string]any)
		return check["level"] != "positive"
	})
}
