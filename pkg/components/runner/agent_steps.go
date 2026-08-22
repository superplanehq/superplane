package runner

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const (
	AgentStepPrompt = "prompt"
	AgentStepBash   = "bash"

	CredentialsSourceSecret      = "secret"
	CredentialsSourceIntegration = "integration"
	CredentialsSourceHosted      = "hosted"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// AgentStep is one ordered bash or prompt action for a fleet-runner agent node.
type AgentStep struct {
	Name             string  `mapstructure:"name"`
	Type             string  `mapstructure:"type"`
	Prompt           *string `mapstructure:"prompt,omitempty"`
	Command          *string `mapstructure:"command,omitempty"`
	WorkingDirectory string  `mapstructure:"workingDirectory,omitempty"`
}

func NormalizeAgentStepType(stepType string) string {
	if strings.TrimSpace(stepType) == AgentStepBash {
		return AgentStepBash
	}
	return AgentStepPrompt
}

func ValidateAgentSteps(steps []AgentStep) error {
	if len(steps) == 0 {
		return fmt.Errorf("at least one step is required")
	}

	promptCount := 0
	for i, step := range steps {
		if strings.TrimSpace(step.Name) == "" {
			return fmt.Errorf("steps[%d].name is required", i)
		}
		if err := validateStepWorkingDirectory(i, step.WorkingDirectory); err != nil {
			return err
		}
		switch NormalizeAgentStepType(step.Type) {
		case AgentStepBash:
			if step.Command == nil || strings.TrimSpace(*step.Command) == "" {
				return fmt.Errorf("steps[%d].command is required for bash steps", i)
			}
		case AgentStepPrompt:
			if step.Prompt == nil || strings.TrimSpace(*step.Prompt) == "" {
				return fmt.Errorf("steps[%d].prompt is required for prompt steps", i)
			}
			promptCount++
		}
	}
	if promptCount == 0 {
		return fmt.Errorf("at least one prompt step is required")
	}
	return nil
}

func validateStepWorkingDirectory(index int, dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}
	if strings.ContainsAny(dir, "\n\r") {
		return fmt.Errorf("steps[%d].workingDirectory must be a single path", index)
	}
	if filepath.IsAbs(dir) {
		return nil
	}
	cleaned := filepath.ToSlash(filepath.Clean(dir))
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("steps[%d].workingDirectory must not contain ..", index)
	}
	return nil
}

func AgentStepLabel(stepName, fallback string) string {
	if label := strings.TrimSpace(stepName); label != "" {
		return label
	}
	return fallback
}

func AgentStepSlug(stepNumber int, name string) string {
	slug := slugifyAgentStepName(name)
	if slug == "" {
		slug = "step"
	}
	return fmt.Sprintf("%02d-%s", stepNumber, slug)
}

func slugifyAgentStepName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range strings.ToLower(trimmed) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	slug := nonSlugChars.ReplaceAllString(b.String(), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 48 {
		slug = strings.Trim(slug[:48], "-")
	}
	return slug
}

func ShellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func DefaultAgentSteps() []map[string]any {
	return []map[string]any{
		{"name": "Prompt", "type": AgentStepPrompt, "prompt": ""},
	}
}

func IsHostedCredentials(source string) bool {
	return strings.TrimSpace(source) == CredentialsSourceHosted
}

func FundingSourceForCredentials(source string) string {
	if IsHostedCredentials(source) {
		return "hosted"
	}
	return "byok"
}
