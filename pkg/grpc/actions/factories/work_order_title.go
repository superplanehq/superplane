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
	if !strings.Contains(line, "![") {
		return line
	}
	closes := markdownLabelCloses(line)
	lastParen := strings.LastIndexByte(line, ')')
	var bareEnds []int
	if lastParen >= 0 {
		bareEnds = bareDestinationEnds(line)
	}
	var builder strings.Builder
	builder.Grow(len(line))
	index := 0
	for index < len(line) {
		next := strings.Index(line[index:], "![")
		if next < 0 {
			builder.WriteString(line[index:])
			break
		}
		next += index
		builder.WriteString(line[index:next])
		if end, ok := markdownImageEnd(line, next, closes, lastParen, bareEnds); ok {
			builder.WriteByte(' ')
			index = end
			continue
		}
		builder.WriteByte(line[next])
		index = next + 1
	}
	return builder.String()
}

func markdownLabelCloses(line string) []int {
	closes := make([]int, len(line))
	for index := range closes {
		closes[index] = -1
	}
	var stack []int
	for index := 0; index < len(line); {
		if len(stack) > 0 {
			if escaped, ok := skipEscapedByte(line, index); ok {
				index = escaped
				continue
			}
		}
		switch line[index] {
		case '[':
			stack = append(stack, index)
		case ']':
			if len(stack) > 0 {
				open := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				closes[open] = index
			}
		}
		index++
	}
	return closes
}

func markdownImageEnd(line string, start int, closes []int, lastParen int, bareEnds []int) (int, bool) {
	if start+1 >= len(line) || line[start] != '!' || line[start+1] != '[' {
		return 0, false
	}
	labelEnd := closes[start+1]
	if labelEnd < 0 || labelEnd+1 >= len(line) || line[labelEnd+1] != '(' {
		return 0, false
	}
	if labelEnd+1 > lastParen {
		return 0, false
	}
	return inlineImageClose(line, labelEnd+1, bareEnds)
}

func bareDestinationEnds(line string) []int {
	escaped := escapedBytes(line)
	matchOpen, matchClose := parenthesisMatches(line, escaped)
	nextSpace := nextUnescapedSpace(line, escaped)
	nextClose := nextDepthZeroClose(line, escaped, matchOpen)
	outside := outsideOpenCounts(line, escaped, matchClose, nextSpace)
	ends := make([]int, len(line))
	for index := range ends {
		ends[index] = -1
	}
	for start := 0; start < len(line); start++ {
		if line[start] == ')' {
			ends[start] = start
			continue
		}
		space := nextSpace[start]
		closeAt := nextClose[start]
		if closeAt >= 0 && closeAt < space {
			ends[start] = closeAt
			continue
		}
		if space >= len(line) {
			continue
		}
		if outside[space]-outside[start] > 0 {
			continue
		}
		ends[start] = space
	}
	return ends
}

func escapedBytes(line string) []bool {
	escaped := make([]bool, len(line))
	for index := 0; index < len(line); {
		if index+1 < len(line) && line[index] == '\\' {
			escaped[index+1] = true
			index += 2
			continue
		}
		index++
	}
	return escaped
}

func parenthesisMatches(line string, escaped []bool) ([]int, []int) {
	matchOpen := make([]int, len(line))
	matchClose := make([]int, len(line))
	for index := range matchOpen {
		matchOpen[index] = -2
		matchClose[index] = -2
	}
	var stack []int
	for index := 0; index < len(line); index++ {
		if escaped[index] {
			continue
		}
		switch line[index] {
		case '(':
			stack = append(stack, index)
		case ')':
			if len(stack) == 0 {
				matchOpen[index] = -1
				continue
			}
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			matchOpen[index] = open
			matchClose[open] = index
		}
	}
	return matchOpen, matchClose
}

func nextUnescapedSpace(line string, escaped []bool) []int {
	nextSpace := make([]int, len(line)+1)
	next := len(line)
	nextSpace[len(line)] = len(line)
	for index := len(line) - 1; index >= 0; index-- {
		if !escaped[index] && isSpaceByte(line, index) {
			next = index
		}
		nextSpace[index] = next
	}
	return nextSpace
}

func nextDepthZeroClose(line string, escaped []bool, matchOpen []int) []int {
	nextClose := make([]int, len(line))
	for index := range nextClose {
		nextClose[index] = -1
	}
	var closes []int
	for index := len(line) - 1; index >= 0; index-- {
		if index+1 < len(line) && !escaped[index+1] && line[index+1] == ')' {
			closes = append(closes, index+1)
		}
		for len(closes) > 0 {
			top := closes[len(closes)-1]
			open := matchOpen[top]
			if open >= index && open != -1 {
				closes = closes[:len(closes)-1]
				continue
			}
			break
		}
		if len(closes) > 0 {
			nextClose[index] = closes[len(closes)-1]
		}
	}
	return nextClose
}

func outsideOpenCounts(line string, escaped []bool, matchClose []int, nextSpace []int) []int {
	outside := make([]int, len(line)+1)
	for index := 0; index < len(line); index++ {
		outside[index+1] = outside[index]
		if escaped[index] || line[index] != '(' {
			continue
		}
		match := matchClose[index]
		if match < 0 || match >= nextSpace[index] {
			outside[index+1]++
		}
	}
	return outside
}

func inlineImageClose(line string, openParen int, bareEnds []int) (int, bool) {
	index := skipSpace(line, openParen+1)
	if index >= len(line) {
		return 0, false
	}
	var ok bool
	if line[index] == '<' {
		index, ok = angleDestinationEnd(line, index)
	} else if index < len(bareEnds) && bareEnds[index] >= 0 {
		index, ok = bareEnds[index], true
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
