// Package mergeconfidence builds run assessment data owned by SuperPlane.
package mergeconfidence

import (
	"math"
	"strings"

	"github.com/superplanehq/superplane/pkg/models/factory"
)

type Assessment struct {
	Passed         bool
	Enabled        []string
	Labels         map[string]string
	Checks         map[string]factory.CheckRef
	ReviewedCommit string
	TaskURL        string
}

type ReportData struct {
	PullRequestID  string        `json:"pullRequestId"`
	Complete       bool          `json:"complete"`
	OverallScore   *float64      `json:"overallScore"`
	Checks         []ReportCheck `json:"checks"`
	ReviewedCommit string        `json:"reviewedCommit"`
	TaskURL        string        `json:"taskUrl"`
}

type ReportCheck struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	Score    float64 `json:"score"`
	MaxScore float64 `json:"maxScore"`
	Summary  string  `json:"summary"`
	Missing  bool    `json:"missing"`
}

// ReportData includes only checks selected in the saved run configuration.
func (a Assessment) ReportData() ReportData {
	report := ReportData{Complete: a.Passed, Checks: []ReportCheck{}, ReviewedCommit: a.ReviewedCommit, TaskURL: a.TaskURL}
	for _, id := range a.Enabled {
		label := a.Labels[id]
		if label == "" {
			label = checkLabel(id)
		}
		check := a.Checks[id+"-review"]
		missing := !validCheck(check)
		if missing {
			report.Complete = false
		}
		report.Checks = append(report.Checks, ReportCheck{ID: id, Label: label, Score: check.Score, MaxScore: check.MaxScore, Summary: check.Summary, Missing: missing})
	}
	if report.Complete {
		if score, ok := a.overallScore(); ok {
			report.OverallScore = &score
		}
	}
	return report
}

// Match the UI headline: invert risk and drift, then take the weakest built-in
// score. Custom checks are visible but do not contribute to the aggregate.
func (a Assessment) overallScore() (float64, bool) {
	weakest, found := 1.0, false
	for _, id := range a.Enabled {
		switch id {
		case "risk", "performance", "security", "drift", "reversibility":
		default:
			continue
		}
		check := a.Checks[id+"-review"]
		if !validCheck(check) {
			continue
		}
		points := check.Score
		if id == "risk" || id == "drift" {
			points = check.MaxScore + 1 - points
		}
		weakest = math.Min(weakest, math.Max(0, math.Min(check.MaxScore, points))/check.MaxScore)
		found = true
	}
	return math.Round(weakest * 5), found
}

func validCheck(check factory.CheckRef) bool {
	return check.MaxScore == 5 && check.Score >= 1 && check.Score <= 5 && math.Trunc(check.Score) == check.Score && strings.TrimSpace(check.Summary) != ""
}

func checkLabel(id string) string {
	switch id {
	case "risk":
		return "Blast radius"
	case "performance":
		return "Performance"
	case "security":
		return "Security"
	case "drift":
		return "Drift from Specification"
	case "reversibility":
		return "Reversibility"
	default:
		return id
	}
}
