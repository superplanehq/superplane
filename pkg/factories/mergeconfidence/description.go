// Package mergeconfidence renders the run assessment owned by SuperPlane.
package mergeconfidence

import (
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/superplanehq/superplane/pkg/models/factory"
)

const (
	StartMarker = "<!-- superplane:merge-confidence:start -->"
	EndMarker   = "<!-- superplane:merge-confidence:end -->"
)

// UpdateDescription preserves every byte outside a single, ordered marker pair.
// Invalid ownership boundaries return the original description and an error.
func UpdateDescription(description, section string) (string, error) {
	starts, ends := strings.Count(description, StartMarker), strings.Count(description, EndMarker)
	if starts == 0 && ends == 0 {
		if description == "" {
			return section, nil
		}
		return description + "\n\n" + section, nil
	}
	start, end := strings.Index(description, StartMarker), strings.Index(description, EndMarker)
	if starts != 1 || ends != 1 || end < start {
		return description, fmt.Errorf("merge confidence description markers are incomplete, reversed, or duplicated")
	}
	return description[:start] + section + description[end+len(EndMarker):], nil
}

type Assessment struct {
	Passed         bool
	Enabled        []string
	Labels         map[string]string
	Checks         map[string]factory.CheckRef
	ReviewedCommit string
	TaskURL        string
}

// Render includes only enabled checks from the saved run configuration.
func Render(a Assessment) string {
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n## SuperPlane merge confidence\n\n", StartMarker)
	complete := a.Passed
	for _, id := range a.Enabled {
		if !validCheck(a.Checks[id+"-review"]) {
			complete = false
		}
	}
	if !complete {
		body.WriteString("**Incomplete**\n\n")
	} else if score, ok := a.overallScore(); ok {
		fmt.Fprintf(&body, "**Overall score: %g/5**\n\n", score)
	}
	if len(a.Enabled) == 0 {
		body.WriteString("No checks are enabled.\n\n")
	} else {
		if complete {
			if _, ok := a.overallScore(); !ok {
				body.WriteString("No built-in checks are enabled.\n\n")
			}
		}
		body.WriteString("| Check | Score | Summary |\n| --- | --- | --- |\n")
		for _, id := range a.Enabled {
			label := a.Labels[id]
			if label == "" {
				label = checkLabel(id)
			}
			check := a.Checks[id+"-review"]
			if !validCheck(check) {
				fmt.Fprintf(&body, "| %s | Missing | No result was reported. |\n", escapeText(label))
				continue
			}
			fmt.Fprintf(&body, "| %s | %g/%g | %s |\n", escapeText(label), check.Score, check.MaxScore, escapeText(check.Summary))
		}
		body.WriteString("\n")
	}
	fmt.Fprintf(&body, "Reviewed commit: %s\n\n[View task](<%s>)\n%s", escapeText(a.ReviewedCommit), escapeLink(a.TaskURL), EndMarker)
	return body.String()
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

// Encode Markdown punctuation and HTML as entities, including pipes and marker
// brackets. Collapse line breaks so generated text cannot create table rows.
func escapeText(value string) string {
	var escaped strings.Builder
	for _, r := range value {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			escaped.WriteByte(' ')
			continue
		}
		if strings.ContainsRune("&<>|\\`*_{}[]()!#~", r) {
			fmt.Fprintf(&escaped, "&#%d;", r)
			continue
		}
		escaped.WriteRune(r)
	}
	return escaped.String()
}

func escapeLink(value string) string {
	return strings.NewReplacer("<", "%3C", ">", "%3E", "\n", "%0A", "\r", "%0D", " ", "%20").Replace(value)
}
