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
)

var environmentVariableNameRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func normalizeCommands(commands string) []string {
	lines := strings.Split(commands, "\n")
	out := make([]string, 0, len(lines))
	block := make([]string, 0)
	blockDepth := 0
	heredocs := make([]string, 0)

	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}

		if len(heredocs) > 0 {
			block = append(block, l)
			if l == heredocs[0] {
				heredocs = heredocs[1:]
			}

			if blockDepth == 0 && len(heredocs) == 0 {
				out = append(out, strings.Join(block, "\n"))
				block = block[:0]
			}

			continue
		}

		openBlocks, closeBlocks, lineHeredocs := shellLineStructure(l)
		if blockDepth == 0 && openBlocks == 0 && len(lineHeredocs) == 0 {
			out = append(out, l)
			continue
		}

		block = append(block, l)
		blockDepth += openBlocks
		blockDepth -= min(closeBlocks, blockDepth)
		heredocs = append(heredocs, lineHeredocs...)
		if blockDepth == 0 && len(heredocs) == 0 {
			out = append(out, strings.Join(block, "\n"))
			block = block[:0]
		}
	}

	if len(block) > 0 {
		out = append(out, strings.Join(block, "\n"))
	}

	return out
}

func validateCommands(commands string) error {
	lines := normalizeCommands(commands)
	if len(lines) == 0 {
		return errors.New("at least one command is required")
	}
	return nil
}

func shellLineStructure(line string) (int, int, []string) {
	words := shellWords(line)
	if len(words) == 0 {
		return 0, 0, nil
	}

	openBlocks := 0
	closeBlocks := 0
	for _, word := range words {
		if word.quoted || !word.commandPosition {
			continue
		}

		switch word.value {
		case "if", "case", "for", "select", "while", "until":
			openBlocks++
		case "{":
			openBlocks++
		case "fi", "done", "esac", "}":
			closeBlocks++
		}
	}

	return openBlocks, closeBlocks, heredocDelimiters(words)
}

type shellWord struct {
	value           string
	quoted          bool
	commandPosition bool
}

func shellWords(line string) []shellWord {
	words := []shellWord{}
	var word strings.Builder
	var quote rune
	escaped := false
	quoted := false
	commandPosition := true

	flush := func() {
		if word.Len() == 0 {
			quoted = false
			return
		}

		words = append(words, shellWord{
			value:           word.String(),
			quoted:          quoted,
			commandPosition: commandPosition,
		})
		word.Reset()
		quoted = false
		commandPosition = false
	}

	for _, r := range line {
		if escaped {
			word.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			continue
		}

		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}

			word.WriteRune(r)
			continue
		}

		switch r {
		case '\'', '"', '`':
			quote = r
			quoted = true
		case '#':
			flush()
			return words
		case ' ', '\t':
			flush()
		case ';', '&', '|', '(', ')':
			flush()
			commandPosition = true
		case '{', '}':
			flush()
			words = append(words, shellWord{
				value:           string(r),
				commandPosition: commandPosition,
			})
			commandPosition = false
		default:
			word.WriteRune(r)
		}
	}

	flush()
	return words
}

func heredocDelimiters(words []shellWord) []string {
	delimiters := []string{}
	for i, word := range words {
		delimiter := ""
		switch {
		case word.value == "<<" || word.value == "<<-":
			if i+1 < len(words) {
				delimiter = words[i+1].value
			}
		case strings.HasPrefix(word.value, "<<-"):
			delimiter = strings.TrimPrefix(word.value, "<<-")
		case strings.HasPrefix(word.value, "<<"):
			delimiter = strings.TrimPrefix(word.value, "<<")
		}

		delimiter = strings.Trim(delimiter, `"'`)
		if delimiter != "" {
			delimiters = append(delimiters, delimiter)
		}
	}

	return delimiters
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
