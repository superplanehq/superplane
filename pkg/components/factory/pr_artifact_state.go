package factory

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
)

var validPrArtifactStates = map[string]bool{
	"open":   true,
	"draft":  true,
	"closed": true,
	"merged": true,
}

type prArtifactLifecycleFieldOptions struct {
	Visibility     []configuration.VisibilityCondition
	StateTogglable bool
	StateDefault   any
}

// prArtifactLifecycleFields is the shared state / merged / draft trio used
// by addPullRequest and updatePullRequest so the two configs cannot drift.
func prArtifactLifecycleFields(opts prArtifactLifecycleFieldOptions) []configuration.Field {
	return []configuration.Field{
		{
			Name:                 "state",
			Label:                "State",
			Description:          "Pull request state — drives the artifact chip's icon/color. One of `open`, `draft`, `closed`, `merged`. Accepts an expression (e.g. `{{ root().data.pull_request.state }}`).",
			Placeholder:          "open",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			Default:              opts.StateDefault,
			Togglable:            opts.StateTogglable,
			VisibilityConditions: opts.Visibility,
		},
		{
			Name:                 "merged",
			Label:                "Merged",
			Description:          "GitHub-native `pull_request.merged` flag. When it resolves to a truthy value, the chip renders as merged even if `state` is blank or still says `open`. Accepts an expression (e.g. `{{ root().data.pull_request.merged }}`).",
			Placeholder:          "{{ root().data.pull_request.merged }}",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			Togglable:            true,
			VisibilityConditions: opts.Visibility,
		},
		{
			Name:                 "draft",
			Label:                "Draft",
			Description:          "GitHub-native `pull_request.draft` flag. When it resolves to a truthy value and the PR is not merged, the chip renders as draft. Accepts an expression (e.g. `{{ root().data.pull_request.draft }}`).",
			Placeholder:          "{{ root().data.pull_request.draft }}",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			Togglable:            true,
			VisibilityConditions: opts.Visibility,
		},
	}
}

// resolvePrArtifactState folds the caller-supplied SuperPlane `state`
// together with GitHub-native `merged` / `draft` flags into the single
// value the UI reads. Mirrors the frontend's `extractPrArtifactState`
// precedence so a chip renders the same before and after a page reload:
//
//  1. `merged` truthy → "merged" (GitHub reports merged PRs as
//     `{ state: "closed", merged: true }`, so this must win over
//     `state: closed` and any leftover `state: open`).
//  2. Explicit non-"open" SuperPlane `state` (draft/closed/merged).
//  3. `draft` truthy → "draft" (GitHub draft PRs stay `state: "open"`).
//  4. Explicit "open" `state`, otherwise empty (defer to defaults).
//
// The model still validates the resolved value against the known
// state vocabulary, so an unknown `state` value fails the write.
func resolvePrArtifactState(state, merged, draft any) string {
	if isTruthyConfigValue(merged) {
		return "merged"
	}

	explicit := normalizePrArtifactStateValue(state)
	// A flag-only update can send `merged: false` (or `draft: false`)
	// while a leftover SuperPlane `state` still says merged/draft.
	// The flag is the newer signal — do not persist the stale state.
	if explicit == "merged" && isPresentConfigValue(merged) && !isTruthyConfigValue(merged) {
		explicit = ""
	}
	if explicit == "draft" && isPresentConfigValue(draft) && !isTruthyConfigValue(draft) {
		explicit = ""
	}

	if explicit != "" && explicit != "open" {
		return explicit
	}

	if isTruthyConfigValue(draft) {
		return "draft"
	}

	return explicit
}

// prArtifactStateUpdates builds the data keys to persist after resolve.
// When a SuperPlane state is produced, `merged` and `draft` are rewritten
// to match it so a leftover GitHub flag cannot outrank a later `state`
// on the chip. When only a flag is supplied, that flag is written as a
// bool so `merged: false` can clear a stale `merged: true`. A vetoed
// `state: merged` (flag is false) is not written back — the chip then
// ignores leftover `state: merged` when `merged` is false.
func prArtifactStateUpdates(state, merged, draft any) (map[string]any, error) {
	resolved := resolvePrArtifactState(state, merged, draft)
	if err := validatePrArtifactState(resolved); err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if resolved != "" {
		updates["state"] = resolved
		updates["merged"] = resolved == "merged"
		updates["draft"] = resolved == "draft"
		return updates, nil
	}

	if isPresentConfigValue(merged) {
		updates["merged"] = isTruthyConfigValue(merged)
	}
	if isPresentConfigValue(draft) {
		updates["draft"] = isTruthyConfigValue(draft)
	}
	return updates, nil
}

func validatePrArtifactState(state string) error {
	if state == "" || validPrArtifactStates[state] {
		return nil
	}
	return fmt.Errorf("invalid pull request state %q (want one of open, draft, closed, merged)", state)
}

// normalizePrArtifactStateValue trims and lower-cases a caller-supplied
// state value. Non-string types (e.g. a resolved bool) are treated as
// absent so a stray `state: {{ pr.merged }}` doesn't overwrite a real
// value with "true".
func normalizePrArtifactStateValue(value any) string {
	raw, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(raw))
}

// isTruthyConfigValue accepts both real booleans (from CEL / native
// resolution) and their string representations ("true"/"false") because
// templated flow inputs almost always arrive as strings.
func isTruthyConfigValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	}
	return false
}

func isPresentConfigValue(value any) bool {
	if value == nil {
		return false
	}
	if raw, ok := value.(string); ok && strings.TrimSpace(raw) == "" {
		return false
	}
	return true
}
