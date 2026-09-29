package checks

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

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

	maxCheckSummaryBytes      = 16 * 1024
	maxTotalCheckSummaryBytes = 96 * 1024
)

var htmlTablePattern = regexp.MustCompile(`(?is)<table\b[^>]*>.*?</table\s*>`)

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
	Key         string `json:"key"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion,omitempty"`
	Description string `json:"description,omitempty"`
	Summary     string `json:"summary,omitempty"`
	DetailsURL  string `json:"detailsUrl,omitempty"`
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
			description := ""
			summary := ""
			if output := run.GetOutput(); output != nil {
				description = strings.TrimSpace(output.GetTitle())
				summary = checkRunOutputBody(output)
			}
			latest[key] = PullRequestCheck{
				Key:         key,
				Name:        name,
				Kind:        checkKindCheckRun,
				Status:      status,
				Conclusion:  conclusion,
				Description: description,
				Summary:     summary,
				DetailsURL:  firstNonEmpty(run.GetDetailsURL(), run.GetHTMLURL()),
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
				Summary:    strings.TrimSpace(status.GetDescription()),
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
	wanted := map[string]bool{}
	for _, name := range selectedNames {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		wanted[strings.ToLower(trimmed)] = true
	}
	if len(wanted) == 0 {
		return nil
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

func storedCheckLists(evaluation waitChecksEvaluation) (checks, selected, failed []PullRequestCheck) {
	return withoutCheckSummaries(evaluation.Checks),
		limitCheckSummaries(evaluation.SelectedChecks),
		limitCheckSummaries(evaluation.FailedChecks)
}

func withoutCheckSummaries(checks []PullRequestCheck) []PullRequestCheck {
	out := make([]PullRequestCheck, len(checks))
	for i, check := range checks {
		check.Summary = ""
		out[i] = check
	}
	return out
}

func limitCheckSummaries(checks []PullRequestCheck) []PullRequestCheck {
	out := make([]PullRequestCheck, len(checks))
	remaining := maxTotalCheckSummaryBytes
	for i, check := range checks {
		check.Summary = limitCheckSummary(check.Summary, min(maxCheckSummaryBytes, remaining))
		remaining -= len(check.Summary)
		out[i] = check
	}
	return out
}

func checkRunOutputBody(output *github.CheckRunOutput) string {
	summary := strings.TrimSpace(output.GetSummary())
	text := strings.TrimSpace(output.GetText())
	body := joinDistinctCheckOutput(summary, text)
	if body == "" {
		return ""
	}
	return limitCheckSummary("\n\n"+body, maxCheckSummaryBytes)
}

func limitCheckSummary(summary string, maxBytes int) string {
	if maxBytes <= 0 || summary == "" {
		return ""
	}
	if len(summary) <= maxBytes {
		return summary
	}

	ellipsis := "\n..."
	budget := maxBytes - len(ellipsis)
	if budget < 1 {
		return truncateToBytes(summary, maxBytes)
	}

	tables := packHTMLTables(htmlTables(summary), budget)
	packed := strings.Join(tables, "\n\n")
	separator := ""
	if packed != "" {
		separator = "\n\n"
	}
	remaining := budget - len(separator) - len(packed)
	if remaining < 1 {
		return packed + ellipsis
	}

	prefix := trimSummaryCut(truncateToBytes(summary, remaining))
	missing := missingHTMLTables(prefix, tables)
	if len(missing) == 0 {
		return prefix + ellipsis
	}
	return prefix + separator + strings.Join(missing, "\n\n") + ellipsis
}

func packHTMLTables(tables []string, budget int) []string {
	var kept []string
	used := 0
	for _, table := range tables {
		need := len(table)
		if len(kept) > 0 {
			need += len("\n\n")
		}
		if used+need > budget {
			continue
		}
		kept = append(kept, table)
		used += need
	}
	return kept
}

func htmlTables(value string) []string {
	return htmlTablePattern.FindAllString(value, -1)
}

func missingHTMLTables(prefix string, tables []string) []string {
	var missing []string
	for _, table := range tables {
		if !strings.Contains(prefix, table) {
			missing = append(missing, table)
		}
	}
	return missing
}

func trimSummaryCut(value string) string {
	return strings.TrimRight(value, " \t\r\n")
}

func truncateToBytes(value string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(value) <= maxBytes {
		return value
	}
	for maxBytes > 0 && !utf8.ValidString(value[:maxBytes]) {
		maxBytes--
	}
	truncated := value[:maxBytes]
	if i := unclosedHTMLTagIndex(truncated); i >= 0 {
		truncated = truncated[:i]
	}
	return truncated
}

func unclosedHTMLTagIndex(value string) int {
	lastOpen := strings.LastIndex(value, "<")
	lastClose := strings.LastIndex(value, ">")
	if lastOpen > lastClose {
		return lastOpen
	}
	return -1
}

func joinDistinctCheckOutput(summary, text string) string {
	if text == "" || strings.Contains(summary, text) {
		return summary
	}
	if summary == "" {
		return text
	}
	return summary + "\n\n" + text
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func nextEvaluateDelay(now, timeoutAt time.Time, pollInterval time.Duration) time.Duration {
	if !now.Before(timeoutAt) {
		return 0
	}
	timeoutRemain := timeoutAt.Sub(now)
	if pollInterval < timeoutRemain {
		return pollInterval
	}
	return timeoutRemain
}
