package factories

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const untitledWorkOrderTitle = "Untitled task"

func fillMissingWorkOrderTitle(tx *gorm.DB, order *models.FactoryWorkOrder) error {
	if order == nil || !workOrderTitleMissing(order.Title) {
		return nil
	}

	title := titleFromWorkOrderDescription(order.Description)
	if title == "" {
		title = untitledWorkOrderTitle
	}
	return order.UpdateContent(tx, &title, nil)
}

func workOrderTitleMissing(title string) bool {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" || !containsLetterOrDigit(trimmed) {
		return true
	}
	return isPlaceholderWorkOrderTitle(trimmed)
}

func titleFromWorkOrderDescription(description string) string {
	for _, raw := range descriptionLines(description) {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "![") {
			continue
		}
		candidate := titleFromDescriptionLine(line)
		if workOrderTitleMissing(candidate) {
			continue
		}
		return cutWorkOrderTitle(candidate)
	}
	return ""
}

func titleFromDescriptionLine(line string) string {
	line = stripLeadingHeading(line)
	line = unwrapFullEmphasis(line)
	return collapseWhitespace(line)
}

func descriptionLines(description string) []string {
	normalized := strings.ReplaceAll(description, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.Split(normalized, "\n")
}

func stripLeadingHeading(line string) string {
	hashes := 0
	for hashes < len(line) && hashes < 6 && line[hashes] == '#' {
		hashes++
	}
	if hashes == 0 || hashes == len(line) {
		return line
	}
	rest := line[hashes:]
	first, _ := utf8.DecodeRuneInString(rest)
	if !unicode.IsSpace(first) {
		return line
	}
	return strings.TrimSpace(rest)
}

func unwrapFullEmphasis(line string) string {
	current := strings.TrimSpace(line)
	for {
		inner, ok := unwrapOneFullEmphasis(current)
		if !ok || len(inner) >= len(current) {
			return current
		}
		current = strings.TrimSpace(inner)
	}
}

func unwrapOneFullEmphasis(line string) (string, bool) {
	for _, delimiter := range []string{"**", "__", "*", "_"} {
		if !fullyWrapped(line, delimiter) {
			continue
		}
		inner := line[len(delimiter) : len(line)-len(delimiter)]
		if strings.TrimSpace(inner) == "" {
			continue
		}
		return inner, true
	}
	return "", false
}

func fullyWrapped(line, delimiter string) bool {
	if len(line) < len(delimiter)*2 || !strings.HasPrefix(line, delimiter) || !strings.HasSuffix(line, delimiter) {
		return false
	}
	if delimiter != "*" && delimiter != "_" {
		return true
	}
	return !strings.HasPrefix(line, delimiter+delimiter) && !strings.HasSuffix(line, delimiter+delimiter)
}

func collapseWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func cutWorkOrderTitle(title string) string {
	if utf8.RuneCountInString(title) <= workOrderTitleMaxLength {
		return title
	}
	return string([]rune(title)[:workOrderTitleMaxLength])
}

func containsLetterOrDigit(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func isPlaceholderWorkOrderTitle(title string) bool {
	switch normalizeWorkOrderTitle(title) {
	case "null", "undefined", "unknown", "untitled", "untitled task", "none", "n/a", "na", "tbd", "todo", "nil":
		return true
	default:
		return false
	}
}

func normalizeWorkOrderTitle(title string) string {
	normalized := strings.TrimSpace(title)
	normalized = unwrapWrappingQuotes(normalized)
	normalized = strings.TrimSpace(normalized)
	normalized = trimOneTrailingPunctuation(normalized)
	return strings.ToLower(strings.TrimSpace(normalized))
}

func unwrapWrappingQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	first, last := value[0], value[len(value)-1]
	if (first == '"' || first == '\'') && first == last {
		return value[1 : len(value)-1]
	}
	return value
}

func trimOneTrailingPunctuation(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	switch runes[len(runes)-1] {
	case '.', '!', '?', ',':
		return string(runes[:len(runes)-1])
	default:
		return value
	}
}
