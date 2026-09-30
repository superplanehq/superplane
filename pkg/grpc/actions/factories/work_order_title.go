package factories

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var attachmentLinkPattern = regexp.MustCompile(
	`(?i)!?\[[^\]]*]\(` + regexp.QuoteMeta(blob.FileRefScheme) + `://[^)\s]+\)`,
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
		candidate := titleFromDescriptionLine(raw)
		if candidate == "" || skippableDescriptionLine(candidate) || workOrderTitleMissing(candidate) {
			continue
		}
		return cutWorkOrderTitle(candidate)
	}
	return ""
}

func titleFromDescriptionLine(line string) string {
	line = strings.TrimSpace(line)
	line = stripLeadingHeading(line)
	line = stripLeadingListMarker(line)
	line = stripMarkdownImages(line)
	line = unwrapFullEmphasis(line)
	return collapseWhitespace(line)
}

func skippableDescriptionLine(line string) bool {
	return isAttachmentOnlyLine(line)
}

func stripMarkdownImages(line string) string {
	var builder strings.Builder
	builder.Grow(len(line))
	for index := 0; index < len(line); {
		if end, ok := markdownImageEnd(line, index); ok {
			builder.WriteByte(' ')
			index = end
			continue
		}
		builder.WriteByte(line[index])
		index++
	}
	return builder.String()
}

func markdownImageEnd(line string, start int) (int, bool) {
	if !strings.HasPrefix(line[start:], "![") {
		return 0, false
	}
	labelEnd, ok := linkLabelEnd(line, start+1)
	if !ok || labelEnd+1 >= len(line) || line[labelEnd+1] != '(' {
		return 0, false
	}
	return inlineImageClose(line, labelEnd+1)
}

func linkLabelEnd(line string, open int) (int, bool) {
	if open >= len(line) || line[open] != '[' {
		return 0, false
	}
	depth := 1
	for index := open + 1; index < len(line); {
		if escaped, ok := skipEscapedByte(line, index); ok {
			index = escaped
			continue
		}
		switch line[index] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return index, true
			}
		}
		index++
	}
	return 0, false
}

func inlineImageClose(line string, openParen int) (int, bool) {
	index := skipSpace(line, openParen+1)
	if index >= len(line) {
		return 0, false
	}
	var ok bool
	if line[index] == '<' {
		index, ok = angleDestinationEnd(line, index)
	} else {
		index, ok = bareDestinationEnd(line, index)
	}
	if !ok {
		return 0, false
	}
	index = skipSpace(line, index)
	if index < len(line) && isImageTitleStart(line[index]) {
		index, ok = imageTitleEnd(line, index)
		if !ok {
			return 0, false
		}
		index = skipSpace(line, index)
	}
	if index >= len(line) || line[index] != ')' {
		return 0, false
	}
	return index + 1, true
}

func angleDestinationEnd(line string, start int) (int, bool) {
	for index := start + 1; index < len(line); {
		if escaped, ok := skipEscapedByte(line, index); ok {
			index = escaped
			continue
		}
		if line[index] == '<' {
			return 0, false
		}
		if line[index] == '>' {
			return index + 1, true
		}
		index++
	}
	return 0, false
}

func bareDestinationEnd(line string, start int) (int, bool) {
	if start < len(line) && line[start] == ')' {
		return start, true
	}
	depth := 0
	for index := start; index < len(line); {
		if escaped, ok := skipEscapedByte(line, index); ok {
			index = escaped
			continue
		}
		switch line[index] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return index, true
			}
			depth--
		default:
			if isSpaceByte(line, index) {
				if depth != 0 {
					return 0, false
				}
				return index, true
			}
		}
		index++
	}
	return 0, false
}

func isImageTitleStart(value byte) bool {
	return value == '"' || value == '\'' || value == '('
}

func imageTitleEnd(line string, start int) (int, bool) {
	closer := line[start]
	if closer == '(' {
		closer = ')'
	}
	for index := start + 1; index < len(line); {
		if escaped, ok := skipEscapedByte(line, index); ok {
			index = escaped
			continue
		}
		if line[index] == closer {
			return index + 1, true
		}
		index++
	}
	return 0, false
}

func skipEscapedByte(line string, index int) (int, bool) {
	if index+1 >= len(line) || line[index] != '\\' {
		return 0, false
	}
	return index + 2, true
}

func skipSpace(line string, index int) int {
	for index < len(line) && isSpaceByte(line, index) {
		_, width := utf8.DecodeRuneInString(line[index:])
		index += width
	}
	return index
}

func isSpaceByte(line string, index int) bool {
	value, _ := utf8.DecodeRuneInString(line[index:])
	return unicode.IsSpace(value)
}

func isAttachmentOnlyLine(line string) bool {
	if !attachmentLinkPattern.MatchString(line) {
		return false
	}
	remaining := attachmentLinkPattern.ReplaceAllString(line, "")
	return !containsLetterOrDigit(remaining)
}

func stripLeadingListMarker(line string) string {
	rest, ok := stripBulletMarker(line)
	if !ok {
		rest, ok = stripOrderedMarker(line)
	}
	if !ok {
		return line
	}
	return stripTaskCheckbox(rest)
}

func stripBulletMarker(line string) (string, bool) {
	if line == "" {
		return line, false
	}
	switch line[0] {
	case '-', '+', '*':
	default:
		return line, false
	}
	rest := line[1:]
	if !hasLeadingSpace(rest) {
		return line, false
	}
	return strings.TrimSpace(rest), true
}

func stripOrderedMarker(line string) (string, bool) {
	digits := 0
	for digits < len(line) && digits < 9 && line[digits] >= '0' && line[digits] <= '9' {
		digits++
	}
	if digits == 0 || digits >= len(line) {
		return line, false
	}
	if line[digits] != '.' && line[digits] != ')' {
		return line, false
	}
	rest := line[digits+1:]
	if !hasLeadingSpace(rest) {
		return line, false
	}
	return strings.TrimSpace(rest), true
}

func hasLeadingSpace(value string) bool {
	first, _ := utf8.DecodeRuneInString(value)
	return unicode.IsSpace(first)
}

func stripTaskCheckbox(line string) string {
	if len(line) < 4 || line[0] != '[' || line[2] != ']' {
		return line
	}
	switch line[1] {
	case ' ', 'x', 'X':
	default:
		return line
	}
	rest := line[3:]
	space, width := utf8.DecodeRuneInString(rest)
	if width == 0 || !unicode.IsSpace(space) {
		return line
	}
	return strings.TrimSpace(rest[width:])
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
