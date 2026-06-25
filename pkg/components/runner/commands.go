package runner

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	EnvironmentValueSourceLiteral = "literal"
	EnvironmentValueSourceSecret  = "secret"

	shellTokenIf              = "if"
	shellTokenCase            = "case"
	shellTokenFor             = "for"
	shellTokenSelect          = "select"
	shellTokenWhile           = "while"
	shellTokenUntil           = "until"
	shellTokenFunction        = "function"
	shellTokenFi              = "fi"
	shellTokenDone            = "done"
	shellTokenEsac            = "esac"
	shellTokenOpenBrace       = "{"
	shellTokenCloseBrace      = "}"
	shellTokenHeredoc         = "<<"
	shellTokenIndentedHeredoc = "<<-"
	shellTokenHereString      = "<<<"
	commandLineSeparator      = "\n"
)

var environmentVariableNameRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var shellBlockOpeners = map[string]struct{}{
	shellTokenIf:        {},
	shellTokenCase:      {},
	shellTokenFor:       {},
	shellTokenSelect:    {},
	shellTokenWhile:     {},
	shellTokenUntil:     {},
	shellTokenOpenBrace: {},
}

var shellBlockClosers = map[string]struct{}{
	shellTokenFi:         {},
	shellTokenDone:       {},
	shellTokenEsac:       {},
	shellTokenCloseBrace: {},
}

func normalizeCommands(commands string) []string {
	lines := strings.Split(commands, commandLineSeparator)
	normalizer := newCommandNormalizer(len(lines))
	for _, line := range lines {
		normalizer.add(line)
	}

	return normalizer.result()
}

func validateCommands(commands string) error {
	lines := normalizeCommands(commands)
	if len(lines) == 0 {
		return errors.New("at least one command is required")
	}
	return nil
}

type commandNormalizer struct {
	normalized        []string
	blockLines        []string
	blockDepth        int
	heredocDelimiters []string
}

func newCommandNormalizer(commandCount int) *commandNormalizer {
	return &commandNormalizer{
		normalized:        make([]string, 0, commandCount),
		blockLines:        []string{},
		heredocDelimiters: []string{},
	}
}

func (n *commandNormalizer) add(rawLine string) {
	if n.inHeredoc() {
		n.addBlockLine(rawLine)
		n.closeHeredocIfDelimiter(strings.TrimSpace(rawLine))
		n.flushBlockIfComplete()
		return
	}

	line := strings.TrimSpace(rawLine)
	if line == "" {
		return
	}

	structure := inspectShellLine(line)
	if !n.inBlock() && !structure.startsBlock() {
		n.normalized = append(n.normalized, line)
		return
	}

	n.addBlockLine(line)
	n.blockDepth += structure.openBlocks
	n.blockDepth -= min(structure.closeBlocks, n.blockDepth)
	n.heredocDelimiters = append(n.heredocDelimiters, structure.heredocDelimiters...)
	n.flushBlockIfComplete()
}

func (n *commandNormalizer) result() []string {
	if len(n.blockLines) > 0 {
		n.normalized = append(n.normalized, strings.Join(n.blockLines, commandLineSeparator))
	}

	return n.normalized
}

func (n *commandNormalizer) addBlockLine(line string) {
	n.blockLines = append(n.blockLines, line)
}

func (n *commandNormalizer) inBlock() bool {
	return n.blockDepth > 0 || n.inHeredoc()
}

func (n *commandNormalizer) inHeredoc() bool {
	return len(n.heredocDelimiters) > 0
}

func (n *commandNormalizer) closeHeredocIfDelimiter(line string) {
	if line == n.heredocDelimiters[0] {
		n.heredocDelimiters = n.heredocDelimiters[1:]
	}
}

func (n *commandNormalizer) flushBlockIfComplete() {
	if n.blockDepth > 0 || n.inHeredoc() {
		return
	}

	n.normalized = append(n.normalized, strings.Join(n.blockLines, commandLineSeparator))
	n.blockLines = n.blockLines[:0]
}

type shellLineInspection struct {
	openBlocks        int
	closeBlocks       int
	heredocDelimiters []string
}

func (s shellLineInspection) startsBlock() bool {
	return s.openBlocks > 0 || len(s.heredocDelimiters) > 0
}

func inspectShellLine(line string) shellLineInspection {
	words := shellWords(line)
	if len(words) == 0 {
		return shellLineInspection{}
	}

	inspection := shellLineInspection{}
	functionBlock := false
	for _, word := range words {
		if word.quoted || !word.commandPosition {
			continue
		}

		if word.value == shellTokenFunction {
			functionBlock = true
			continue
		}

		if _, ok := shellBlockOpeners[word.value]; ok {
			inspection.openBlocks++
			continue
		}

		if _, ok := shellBlockClosers[word.value]; ok {
			inspection.closeBlocks++
		}
	}

	if functionBlock &&
		hasUnquotedWord(words, shellTokenOpenBrace) &&
		!hasUnquotedCommandPositionWord(words, shellTokenOpenBrace) {
		inspection.openBlocks++
	}

	inspection.heredocDelimiters = heredocDelimiters(words)
	return inspection
}

func hasUnquotedWord(words []shellWord, value string) bool {
	for _, word := range words {
		if !word.quoted && word.value == value {
			return true
		}
	}

	return false
}

func hasUnquotedCommandPositionWord(words []shellWord, value string) bool {
	for _, word := range words {
		if !word.quoted && word.commandPosition && word.value == value {
			return true
		}
	}

	return false
}

type shellWord struct {
	value           string
	quoted          bool
	commandPosition bool
}

type shellLineScanner struct {
	words           []shellWord
	word            strings.Builder
	quote           rune
	escaped         bool
	quoted          bool
	commandPosition bool
}

func newShellLineScanner() *shellLineScanner {
	return &shellLineScanner{
		words:           []shellWord{},
		commandPosition: true,
	}
}

func shellWords(line string) []shellWord {
	scanner := newShellLineScanner()

	for _, r := range line {
		if scanner.scan(r) {
			break
		}
	}

	return scanner.result()
}

func (s *shellLineScanner) scan(r rune) bool {
	switch {
	case s.escaped:
		s.writeEscaped(r)
	case r == '\\':
		s.escaped = true
	case s.inQuote():
		s.scanQuoted(r)
	default:
		return s.scanUnquoted(r)
	}

	return false
}

func (s *shellLineScanner) result() []shellWord {
	s.flushWord()
	return s.words
}

func (s *shellLineScanner) writeEscaped(r rune) {
	s.word.WriteRune(r)
	s.escaped = false
}

func (s *shellLineScanner) inQuote() bool {
	return s.quote != 0
}

func (s *shellLineScanner) scanQuoted(r rune) {
	if r == s.quote {
		s.quote = 0
		return
	}

	s.word.WriteRune(r)
}

func (s *shellLineScanner) scanUnquoted(r rune) bool {
	switch {
	case isShellQuote(r):
		s.quote = r
		s.quoted = true
	case r == '#':
		s.flushWord()
		return true
	case isShellWhitespace(r):
		s.flushWord()
	case isShellCommandSeparator(r):
		s.flushWord()
		s.commandPosition = true
	case isShellBrace(r):
		s.addBraceWord(r)
	default:
		s.word.WriteRune(r)
	}

	return false
}

func (s *shellLineScanner) flushWord() {
	if s.word.Len() == 0 {
		s.quoted = false
		return
	}

	s.words = append(s.words, shellWord{
		value:           s.word.String(),
		quoted:          s.quoted,
		commandPosition: s.commandPosition,
	})
	s.word.Reset()
	s.quoted = false
	s.commandPosition = false
}

func (s *shellLineScanner) addBraceWord(r rune) {
	s.flushWord()
	s.words = append(s.words, shellWord{
		value:           string(r),
		commandPosition: s.commandPosition,
	})
	s.commandPosition = false
}

func isShellQuote(r rune) bool {
	return r == '\'' || r == '"' || r == '`'
}

func isShellWhitespace(r rune) bool {
	return r == ' ' || r == '\t'
}

func isShellCommandSeparator(r rune) bool {
	return r == ';' || r == '&' || r == '|' || r == '(' || r == ')'
}

func isShellBrace(r rune) bool {
	return r == '{' || r == '}'
}

func heredocDelimiters(words []shellWord) []string {
	delimiters := []string{}
	for i := range words {
		delimiter := heredocDelimiter(words, i)
		if delimiter != "" {
			delimiters = append(delimiters, delimiter)
		}
	}

	return delimiters
}

func heredocDelimiter(words []shellWord, index int) string {
	value := words[index].value
	switch {
	case value == shellTokenHeredoc || value == shellTokenIndentedHeredoc:
		if index+1 >= len(words) {
			return ""
		}
		return cleanHeredocDelimiter(words[index+1].value)
	case strings.HasPrefix(value, shellTokenHereString):
		return ""
	case strings.HasPrefix(value, shellTokenIndentedHeredoc):
		return cleanHeredocDelimiter(strings.TrimPrefix(value, shellTokenIndentedHeredoc))
	case strings.HasPrefix(value, shellTokenHeredoc):
		return cleanHeredocDelimiter(strings.TrimPrefix(value, shellTokenHeredoc))
	default:
		return ""
	}
}

func cleanHeredocDelimiter(delimiter string) string {
	return strings.Trim(delimiter, `"'`)
}

func validateEnvironment(environment []EnvironmentVariable) error {
	seen := make(map[string]struct{}, len(environment))

	for i, variable := range environment {
		name := strings.TrimSpace(variable.Name)
		if name == "" {
			return fmt.Errorf("environment[%d].name is required", i)
		}

		if !environmentVariableNameRegex.MatchString(name) {
			return fmt.Errorf("invalid environment variable name: %s", variable.Name)
		}

		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate environment variable name: %s", name)
		}
		seen[name] = struct{}{}

		switch strings.TrimSpace(variable.ValueSource) {
		case EnvironmentValueSourceLiteral:
			if variable.Value == nil {
				return fmt.Errorf("environment[%d].value is required for literal environment variables", i)
			}

		case EnvironmentValueSourceSecret:
			if !variable.Secret.IsSet() {
				return fmt.Errorf("environment[%d].secret.secret and environment[%d].secret.key are required", i, i)
			}

		case "":
			return fmt.Errorf("environment[%d].valueSource is required", i)

		default:
			return fmt.Errorf("invalid environment variable value source: %s", variable.ValueSource)
		}
	}

	return nil
}

func resolveEnvironment(secrets core.SecretsContext, environment []EnvironmentVariable) ([]BrokerEnvironmentVariable, error) {
	if len(environment) == 0 {
		return nil, nil
	}

	resolved := make([]BrokerEnvironmentVariable, 0, len(environment))
	for _, variable := range environment {
		name := strings.TrimSpace(variable.Name)

		switch strings.TrimSpace(variable.ValueSource) {
		case EnvironmentValueSourceLiteral:
			resolved = append(resolved, BrokerEnvironmentVariable{
				Name:  name,
				Value: *variable.Value,
			})

		case EnvironmentValueSourceSecret:
			if secrets == nil {
				return nil, fmt.Errorf("failed to resolve environment variable %s: secrets context is unavailable", name)
			}

			value, err := secrets.GetKey(variable.Secret.Secret, variable.Secret.Key)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve environment variable %s secret %s/%s: %w", name, variable.Secret.Secret, variable.Secret.Key, err)
			}

			resolved = append(resolved, BrokerEnvironmentVariable{
				Name:  name,
				Value: string(value),
			})
		}
	}

	return resolved, nil
}
