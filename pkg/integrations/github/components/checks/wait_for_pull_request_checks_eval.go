package checks

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
)

const (
	checkKindCheckRun = "check_run"
	checkKindStatus   = "status"

	checkStatusPending   = "pending"
	checkStatusCompleted = "completed"

	waitChecksOutcomePassed   = "passed"
	waitChecksOutcomeFailed   = "failed"
	waitChecksOutcomeTimedOut = "timedOut"
	waitChecksOutcomePending  = "pending"
)

var nonFailingConclusions = map[string]bool{
	"success":   true,
	"neutral":   true,
	"skipped":   true,
	"cancelled": true,
}

var failingConclusions = map[string]bool{
	"failure":         true,
	"error":           true,
	"timed_out":       true,
	"action_required": true,
}

type PullRequestCheck struct {
	Key        string `json:"key"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion,omitempty"`
	DetailsURL string `json:"detailsUrl,omitempty"`
}

type waitChecksEvaluation struct {
	Outcome         string
	AllTerminal     bool
	Fingerprint     string
	Checks          []PullRequestCheck
	SelectedChecks  []PullRequestCheck
	FailedChecks    []PullRequestCheck
	MissingSelected []string
}

func normalizePullRequestChecks(checkRuns *github.ListCheckRunsResults, combined *github.CombinedStatus) []PullRequestCheck {
	latest := map[string]PullRequestCheck{}

	if checkRuns != nil {
		for _, run := range checkRuns.CheckRuns {
			if run == nil {
				continue
			}
			name := strings.TrimSpace(run.GetName())
			if name == "" {
				continue
			}
			appSlug := ""
			if run.GetApp() != nil {
				appSlug = strings.TrimSpace(run.GetApp().GetSlug())
			}
			key := fmt.Sprintf("check-run:%s:%s", appSlug, name)
			status := strings.ToLower(strings.TrimSpace(run.GetStatus()))
			conclusion := strings.ToLower(strings.TrimSpace(run.GetConclusion()))
			if status != checkStatusCompleted {
				status = checkStatusPending
				conclusion = ""
			}
			latest[key] = PullRequestCheck{
				Key:        key,
				Name:       name,
				Kind:       checkKindCheckRun,
				Status:     status,
				Conclusion: conclusion,
				DetailsURL: firstNonEmpty(run.GetDetailsURL(), run.GetHTMLURL()),
			}
		}
	}

	if combined != nil {
		for _, status := range combined.Statuses {
			if status == nil {
				continue
			}
			contextName := strings.TrimSpace(status.GetContext())
			if contextName == "" {
				continue
			}
			key := "status:" + contextName
			state := strings.ToLower(strings.TrimSpace(status.GetState()))
			normalizedStatus := checkStatusPending
			conclusion := ""
			if state != "pending" && state != "" {
				normalizedStatus = checkStatusCompleted
				conclusion = state
			}
			latest[key] = PullRequestCheck{
				Key:        key,
				Name:       contextName,
				Kind:       checkKindStatus,
				Status:     normalizedStatus,
				Conclusion: conclusion,
				DetailsURL: status.GetTargetURL(),
			}
		}
	}

	checks := make([]PullRequestCheck, 0, len(latest))
	for _, check := range latest {
		checks = append(checks, check)
	}
	sort.Slice(checks, func(i, j int) bool {
		return checks[i].Key < checks[j].Key
	})
	return checks
}

func evaluatePullRequestChecks(checks []PullRequestCheck, selectedNames []string, timedOut bool) waitChecksEvaluation {
	selected := selectedChecks(checks, selectedNames)
	failed := failedChecks(selected)
	missing := missingSelectedNames(checks, selectedNames)
	fingerprint := checkFingerprint(checks)

	evaluation := waitChecksEvaluation{
		Checks:          checks,
		SelectedChecks:  selected,
		FailedChecks:    failed,
		MissingSelected: missing,
		Fingerprint:     fingerprint,
	}

	if hasPending(selected) || len(missing) > 0 {
		evaluation.AllTerminal = false
		if timedOut {
			evaluation.Outcome = waitChecksOutcomeTimedOut
			return evaluation
		}
		evaluation.Outcome = waitChecksOutcomePending
		return evaluation
	}

	evaluation.AllTerminal = true
	if timedOut {
		evaluation.Outcome = waitChecksOutcomeTimedOut
		return evaluation
	}
	if len(failed) > 0 {
		evaluation.Outcome = waitChecksOutcomeFailed
		return evaluation
	}
	evaluation.Outcome = waitChecksOutcomePassed
	return evaluation
}

func selectedChecks(checks []PullRequestCheck, selectedNames []string) []PullRequestCheck {
	if len(selectedNames) == 0 {
		return checks
	}

	wanted := map[string]bool{}
	for _, name := range selectedNames {
		wanted[strings.ToLower(strings.TrimSpace(name))] = true
	}

	selected := make([]PullRequestCheck, 0, len(checks))
	for _, check := range checks {
		if wanted[strings.ToLower(check.Name)] {
			selected = append(selected, check)
		}
	}
	return selected
}

func missingSelectedNames(checks []PullRequestCheck, selectedNames []string) []string {
	if len(selectedNames) == 0 {
		return nil
	}

	seen := map[string]bool{}
	for _, check := range checks {
		seen[strings.ToLower(check.Name)] = true
	}

	var missing []string
	for _, name := range selectedNames {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		if !seen[strings.ToLower(trimmed)] {
			missing = append(missing, trimmed)
		}
	}
	return missing
}

func failedChecks(checks []PullRequestCheck) []PullRequestCheck {
	var failed []PullRequestCheck
	for _, check := range checks {
		if check.Status != checkStatusCompleted {
			continue
		}
		if failingConclusions[check.Conclusion] {
			failed = append(failed, check)
		}
	}
	return failed
}

func hasPending(checks []PullRequestCheck) bool {
	for _, check := range checks {
		if check.Status != checkStatusCompleted {
			return true
		}
		if check.Conclusion != "" && !nonFailingConclusions[check.Conclusion] && !failingConclusions[check.Conclusion] {
			return true
		}
	}
	return false
}

func checkFingerprint(checks []PullRequestCheck) string {
	payload, err := json.Marshal(checks)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func nextEvaluateDelay(now, lastChange, timeoutAt time.Time, allTerminal bool, quietPeriod, pollInterval time.Duration) time.Duration {
	if !now.Before(timeoutAt) {
		return 0
	}
	timeoutRemain := timeoutAt.Sub(now)
	if allTerminal {
		quietUntil := lastChange.Add(quietPeriod)
		if !now.Before(quietUntil) {
			return 0
		}
		quietRemain := quietUntil.Sub(now)
		if quietRemain < timeoutRemain {
			return quietRemain
		}
		return timeoutRemain
	}
	if pollInterval < timeoutRemain {
		return pollInterval
	}
	return timeoutRemain
}
