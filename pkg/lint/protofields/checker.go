// Package protofields lints .proto message definitions for field-number gaps.
//
// The house style is that message field numbers stay sequential and clean:
// within each message, the used field numbers must be contiguous from the
// lowest number to the highest with no holes. Wire-level compatibility is not a
// concern because the protos are used for JSON conversion, so removing a field
// should be followed by renumbering the remaining fields rather than leaving a
// gap (or a `reserved` marker, which this linter also treats as a gap).
package protofields

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// DefaultRootDir is the directory whose top-level .proto files are linted.
// Nested directories such as protos/include/ are intentionally excluded.
const DefaultRootDir = "protos"

// Guidance explains how to resolve a field-number gap.
const Guidance = `Proto message field numbers must be contiguous with no gaps.

Why:
- Protos here are used for JSON conversion, not wire compatibility, so field
  numbers are cosmetic and the house style keeps them sequential and clean.
- Gaps usually mean a field was removed without renumbering the ones after it.

How to fix:
- Renumber the remaining fields so the numbers run contiguously (e.g. 1..N).
- Do not use "reserved" to paper over a hole; renumber instead.
- Regenerate the protobuf code with "make pb.gen".

See AGENTS.md "Build, Test, and Development Commands".`

// Issue describes a single message whose field numbers are not contiguous.
type Issue struct {
	// Path is the .proto file that declares the message.
	Path string
	// Message is the message name, qualified with its parents when nested
	// (e.g. "CanvasRepository.Metadata").
	Message string
	// Missing lists the field numbers between the lowest and highest used
	// numbers that are not assigned to any field, sorted ascending.
	Missing []int
	// Have lists the field numbers that are used, sorted ascending.
	Have []int
}

// Key returns a stable identifier for the issue.
func (i Issue) Key() string {
	return fmt.Sprintf("%s:%s", i.Path, i.Message)
}

func (i Issue) String() string {
	return fmt.Sprintf(
		"%s: %s: field numbers have gaps: missing %s (have %s)",
		i.Path, i.Message, formatList(i.Missing), formatRanges(i.Have),
	)
}

// Run lints every top-level .proto file in rootDir and returns one issue per
// message with a field-number gap.
func Run(rootDir string) ([]Issue, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("read proto dir %q: %w", rootDir, err)
	}

	var issues []Issue
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".proto") {
			continue
		}

		path := filepath.Join(rootDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		issues = append(issues, parse(path, string(content))...)
	}

	sort.Slice(issues, func(a, b int) bool {
		if issues[a].Path != issues[b].Path {
			return issues[a].Path < issues[b].Path
		}
		return issues[a].Message < issues[b].Message
	})

	return issues, nil
}

// messageFrame accumulates the field numbers declared directly in one message
// (fields inside its oneof blocks count; fields of nested messages do not).
type messageFrame struct {
	name    string
	numbers map[int]bool
}

// parse tokenizes a .proto file and collects field numbers per message.
func parse(path, src string) []Issue {
	tokens := tokenize(src)

	var (
		issues       []Issue
		blockStack   []string        // kind of each open "{ }" block
		nameStack    []string        // enclosing message names, for qualified naming
		messageStack []*messageFrame // one frame per open message block
		stmtTokens   []string        // leading tokens of the current statement
		bracketDepth int             // depth of "[ ]" (inline field options)
	)

	for idx := 0; idx < len(tokens); idx++ {
		switch tok := tokens[idx]; tok {
		case "[":
			bracketDepth++
		case "]":
			if bracketDepth > 0 {
				bracketDepth--
			}
		case "{":
			kind := blockKind(stmtTokens)
			blockStack = append(blockStack, kind)
			if kind == "message" {
				nameStack = append(nameStack, messageName(stmtTokens))
				messageStack = append(messageStack, &messageFrame{
					name:    strings.Join(nameStack, "."),
					numbers: map[int]bool{},
				})
			}
			stmtTokens = nil
		case "}":
			if len(blockStack) > 0 {
				kind := blockStack[len(blockStack)-1]
				blockStack = blockStack[:len(blockStack)-1]
				if kind == "message" {
					frame := messageStack[len(messageStack)-1]
					messageStack = messageStack[:len(messageStack)-1]
					nameStack = nameStack[:len(nameStack)-1]
					if issue, ok := gapIssue(path, frame); ok {
						issues = append(issues, issue)
					}
				}
			}
			stmtTokens = nil
		case ";":
			stmtTokens = nil
		case "=":
			if bracketDepth == 0 && len(messageStack) > 0 && isFieldStatement(blockStack, stmtTokens) {
				if idx+1 < len(tokens) {
					if num, err := strconv.Atoi(tokens[idx+1]); err == nil {
						messageStack[len(messageStack)-1].numbers[num] = true
					}
				}
			}
		default:
			stmtTokens = append(stmtTokens, tok)
		}
	}

	return issues
}

// isFieldStatement reports whether the current statement declares a field whose
// number should be counted: it must sit directly inside a message or oneof
// block and must not be an option/reserved/block-opening statement.
func isFieldStatement(blockStack, stmtTokens []string) bool {
	if len(blockStack) == 0 || len(stmtTokens) == 0 {
		return false
	}

	switch blockStack[len(blockStack)-1] {
	case "message", "oneof":
	default:
		return false
	}

	switch stmtTokens[0] {
	case "option", "reserved", "extensions", "message", "enum", "oneof", "rpc", "service", "extend":
		return false
	}

	return true
}

// blockKind classifies a "{" by the keyword that opened the statement.
func blockKind(stmtTokens []string) string {
	if len(stmtTokens) == 0 {
		return "block"
	}

	switch stmtTokens[0] {
	case "message", "enum", "oneof", "service", "rpc", "extend":
		return stmtTokens[0]
	default:
		return "block"
	}
}

func messageName(stmtTokens []string) string {
	if len(stmtTokens) >= 2 {
		return stmtTokens[1]
	}
	return "?"
}

func gapIssue(path string, frame *messageFrame) (Issue, bool) {
	if len(frame.numbers) == 0 {
		return Issue{}, false
	}

	have := make([]int, 0, len(frame.numbers))
	for num := range frame.numbers {
		have = append(have, num)
	}
	sort.Ints(have)

	var missing []int
	for num := have[0]; num <= have[len(have)-1]; num++ {
		if !frame.numbers[num] {
			missing = append(missing, num)
		}
	}
	if len(missing) == 0 {
		return Issue{}, false
	}

	return Issue{Path: path, Message: frame.name, Missing: missing, Have: have}, true
}

// tokenize splits proto source into tokens, discarding whitespace and comments.
// Identifiers keep embedded dots (e.g. "google.protobuf.Timestamp") and string
// literals are kept whole so their contents never look like statements.
func tokenize(src string) []string {
	var tokens []string
	for i, n := 0, len(src); i < n; {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '/' && i+1 < n && src[i+1] == '/':
			for i < n && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < n && src[i+1] == '*':
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i += 2
		case c == '"' || c == '\'':
			j := i + 1
			for j < n {
				if src[j] == '\\' && j+1 < n {
					j += 2
					continue
				}
				if src[j] == c {
					j++
					break
				}
				j++
			}
			tokens = append(tokens, src[i:j])
			i = j
		case isIdentStart(c):
			j := i + 1
			for j < n && isIdentPart(src[j]) {
				j++
			}
			tokens = append(tokens, src[i:j])
			i = j
		case c >= '0' && c <= '9':
			j := i + 1
			for j < n && src[j] >= '0' && src[j] <= '9' {
				j++
			}
			tokens = append(tokens, src[i:j])
			i = j
		default:
			tokens = append(tokens, string(c))
			i++
		}
	}

	return tokens
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9') || c == '.'
}

// formatList renders numbers as "4, 11, 13".
func formatList(nums []int) string {
	parts := make([]string, len(nums))
	for i, num := range nums {
		parts[i] = strconv.Itoa(num)
	}
	return strings.Join(parts, ", ")
}

// formatRanges renders a sorted list as compact ranges, e.g. "1–3, 5–10, 12, 14".
func formatRanges(nums []int) string {
	if len(nums) == 0 {
		return ""
	}

	var parts []string
	start, prev := nums[0], nums[0]
	flush := func() {
		if start == prev {
			parts = append(parts, strconv.Itoa(start))
		} else {
			parts = append(parts, fmt.Sprintf("%d–%d", start, prev))
		}
	}

	for _, num := range nums[1:] {
		if num == prev+1 {
			prev = num
			continue
		}
		flush()
		start, prev = num, num
	}
	flush()

	return strings.Join(parts, ", ")
}
