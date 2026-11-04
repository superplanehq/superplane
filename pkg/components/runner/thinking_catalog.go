package runner

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/llm/modelid"
	"github.com/superplanehq/superplane/pkg/models"
)

// ModelThinkingLevels returns the thinking levels that the agent CLI for a
// direct provider can send to a model, in display order. Anthropic runs
// Claude Code and OpenAI runs Codex. OpenRouter efforts come from its stored
// provider catalog. An empty result means that the model has no thinking control.
//
// Keep this catalog in sync with the models that the CLIs accept. A model
// that is not in the catalog has no thinking levels, so the UI hides the
// control instead of sending a value that does nothing.
func ModelThinkingLevels(provider, model string) []string {
	id := strings.ToLower(modelid.CatalogModelID(model))
	if id == "" {
		return nil
	}
	return slices.Clone(modelThinkingLevels(strings.TrimSpace(provider), id))
}

func modelThinkingLevels(provider, id string) []string {
	switch provider {
	case models.UsageProviderAnthropic:
		return claudeCodeModelThinkingLevels(id)
	case models.UsageProviderOpenAI:
		return openAIModelThinkingLevels(modelLeaf(id))
	default:
		return nil
	}
}

// ProviderCLIThinkingLevels returns the levels that the agent CLI for a
// provider accepts, whatever the model.
func ProviderCLIThinkingLevels(provider string) []string {
	switch strings.TrimSpace(provider) {
	case models.UsageProviderAnthropic:
		return ClaudeCodeThinkingLevels
	case models.UsageProviderOpenAI:
		return CodexThinkingLevels
	case models.UsageProviderOpenRouter, models.UsageProviderCustom:
		return OpenCodeThinkingLevels
	default:
		return CommonThinkingLevels()
	}
}

// CommonThinkingLevels returns the levels that every agent CLI accepts.
func CommonThinkingLevels() []string {
	var out []string
	for _, level := range allThinkingLevels {
		if slices.Contains(ClaudeCodeThinkingLevels, level) &&
			slices.Contains(CodexThinkingLevels, level) &&
			slices.Contains(OpenCodeThinkingLevels, level) {
			out = append(out, level)
		}
	}
	return out
}

// SupportedThinkingLevels keeps the known levels from a provider catalog,
// such as the Codex model list, in display order.
func SupportedThinkingLevels(levels []string) []string {
	out := make([]string, 0, len(levels))
	for _, level := range allThinkingLevels {
		if slices.ContainsFunc(levels, func(candidate string) bool {
			return strings.EqualFold(strings.TrimSpace(candidate), level)
		}) {
			out = append(out, level)
		}
	}
	return out
}

var (
	thinkingLowToHigh      = []string{ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh}
	thinkingLowToXHigh     = []string{ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh, ThinkingLevelXHigh}
	thinkingNoneToHigh     = []string{ThinkingLevelNone, ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh}
	thinkingNoneToXHigh    = []string{ThinkingLevelNone, ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh, ThinkingLevelXHigh}
	thinkingMinimalToHigh  = []string{ThinkingLevelMinimal, ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh}
	thinkingClaudeAdaptive = []string{ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh, ThinkingLevelMax}
)

var claudeModelPattern = regexp.MustCompile(`claude-(opus|sonnet|haiku|fable)-(\d+)(?:[.-](\d{1,2}))?(?:[.@-]|$)`)

type claudeModel struct {
	family string
	major  int
	minor  int
}

func (m claudeModel) atLeast(major, minor int) bool {
	return m.major > major || m.major == major && m.minor >= minor
}

func parseClaudeModel(id string) (claudeModel, bool) {
	match := claudeModelPattern.FindStringSubmatch(id)
	if match == nil {
		return claudeModel{}, false
	}
	major, _ := strconv.Atoi(match[2])
	minor, _ := strconv.Atoi(match[3])
	return claudeModel{family: match[1], major: major, minor: minor}, true
}

// claudeCodeModelThinkingLevels follows the Claude effort parameter:
// Opus 4.5 has low to high, the 4.6 models add max, and 4.7 and later add xhigh.
func claudeCodeModelThinkingLevels(id string) []string {
	model, ok := parseClaudeModel(id)
	if !ok || model.family == "haiku" {
		return nil
	}
	switch {
	case model.atLeast(4, 7):
		return ClaudeCodeThinkingLevels
	case model.atLeast(4, 6):
		return thinkingClaudeAdaptive
	case model.family == "opus" && model.atLeast(4, 5):
		return thinkingLowToHigh
	default:
		return nil
	}
}

var (
	gpt5Pattern      = regexp.MustCompile(`^gpt-5(?:[.-]|$)`)
	gpt5MinorPattern = regexp.MustCompile(`^gpt-5[.-](\d{1,2})(?:[.-]|$)`)
	gpt5ProPattern   = regexp.MustCompile(`^gpt-5[.-]?pro(?:[.-]|$)`)
	gpt5xProPattern  = regexp.MustCompile(`^gpt-5[.-]\d{1,2}[.-]pro(?:[.-]|$)`)
	oSeriesPattern   = regexp.MustCompile(`^o\d(?:[.-]|$)`)
)

// openAIModelThinkingLevels follows the OpenAI reasoning effort values for
// each model family. Models without reasoning, such as gpt-4o, have none.
func openAIModelThinkingLevels(id string) []string {
	if oSeriesPattern.MatchString(id) {
		return thinkingLowToHigh
	}
	if !gpt5Pattern.MatchString(id) {
		return nil
	}

	minor, hasMinor := gpt5Minor(id)
	switch {
	case strings.Contains(id, "deep-research"):
		return []string{ThinkingLevelMedium}
	case strings.Contains(id, "-chat"):
		if hasMinor {
			return []string{ThinkingLevelMedium}
		}
		return nil
	case gpt5ProPattern.MatchString(id):
		return []string{ThinkingLevelHigh}
	case strings.Contains(id, "codex"):
		return codexModelThinkingLevels(id, minor, hasMinor)
	case gpt5xProPattern.MatchString(id):
		return []string{ThinkingLevelMedium, ThinkingLevelHigh, ThinkingLevelXHigh}
	case !hasMinor:
		return thinkingMinimalToHigh
	case minor == 1:
		return thinkingNoneToHigh
	default:
		return thinkingNoneToXHigh
	}
}

func codexModelThinkingLevels(id string, minor int, hasMinor bool) []string {
	switch {
	case hasMinor && minor >= 3:
		return thinkingNoneToXHigh
	case strings.Contains(id, "codex-max") || hasMinor && minor >= 2:
		return thinkingLowToXHigh
	default:
		return thinkingLowToHigh
	}
}

func gpt5Minor(id string) (int, bool) {
	match := gpt5MinorPattern.FindStringSubmatch(id)
	if match == nil {
		return 0, false
	}
	minor, err := strconv.Atoi(match[1])
	return minor, err == nil && minor > 0
}

func modelLeaf(id string) string {
	if slash := strings.LastIndex(id, "/"); slash >= 0 {
		return id[slash+1:]
	}
	return id
}
