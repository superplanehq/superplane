package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/viper"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

const (
	EnvURL   = "SUPERPLANE_URL"
	EnvToken = "SUPERPLANE_TOKEN"
)

type ConfigContext struct {
	URL            string  `json:"url" yaml:"url"`
	Organization   string  `json:"organization" yaml:"organization"`
	OrganizationID string  `json:"organizationId,omitempty" yaml:"organizationId,omitempty"`
	APIToken       string  `json:"apiToken" yaml:"apiToken"`
	App            *string `json:"app,omitempty" yaml:"app,omitempty"`
	Canvas         *string `json:"canvas,omitempty" yaml:"canvas,omitempty"` // deprecated: use app
}

func activeAppID(context ConfigContext) string {
	if context.App != nil && strings.TrimSpace(*context.App) != "" {
		return strings.TrimSpace(*context.App)
	}
	if context.Canvas != nil {
		return strings.TrimSpace(*context.Canvas)
	}
	return ""
}

func normalizeBaseURL(raw string) string {
	baseURL := strings.TrimSpace(raw)
	return strings.TrimRight(baseURL, "/")
}

func normalizeContext(context ConfigContext) ConfigContext {
	context.URL = normalizeBaseURL(context.URL)
	context.Organization = strings.TrimSpace(context.Organization)
	context.OrganizationID = strings.TrimSpace(context.OrganizationID)
	context.APIToken = strings.TrimSpace(context.APIToken)
	return context
}

func ContextSelector(context ConfigContext) string {
	context = normalizeContext(context)
	id := context.OrganizationID
	if id == "" {
		id = context.Organization
	}
	return fmt.Sprintf("%s/%s", context.URL, id)
}

func normalizeContextSelector(raw string) string {
	selector := strings.TrimSpace(raw)
	selector = strings.TrimRight(selector, "/")

	splitIndex := strings.LastIndex(selector, "/")
	if splitIndex <= 0 || splitIndex == len(selector)-1 {
		return selector
	}

	baseURL := normalizeBaseURL(selector[:splitIndex])
	organization := strings.TrimSpace(selector[splitIndex+1:])
	return fmt.Sprintf("%s/%s", baseURL, organization)
}

func GetContexts() []ConfigContext {
	var contexts []ConfigContext
	if err := viper.UnmarshalKey(ConfigKeyContexts, &contexts); err != nil {
		return nil
	}

	normalized := make([]ConfigContext, 0, len(contexts))
	for _, context := range contexts {
		context = normalizeContext(context)
		if context.URL == "" || context.APIToken == "" {
			continue
		}
		normalized = append(normalized, context)
	}

	return normalized
}

func GetEnvironmentContext() (ConfigContext, bool) {
	context, ok, err := environmentContext()
	if err != nil {
		return ConfigContext{}, false
	}

	return context, ok
}

func ValidateEnvironmentContext() error {
	_, _, err := environmentContext()
	return err
}

func environmentContext() (ConfigContext, bool, error) {
	rawURL, urlSet := os.LookupEnv(EnvURL)
	rawToken, tokenSet := os.LookupEnv(EnvToken)
	if !urlSet && !tokenSet {
		return ConfigContext{}, false, nil
	}

	url := normalizeBaseURL(rawURL)
	token := strings.TrimSpace(rawToken)
	if url == "" || token == "" {
		return ConfigContext{}, false, fmt.Errorf("%s and %s must both be set to use environment CLI authentication", EnvURL, EnvToken)
	}

	return ConfigContext{
		URL:          url,
		Organization: "environment",
		APIToken:     token,
	}, true, nil
}

func GetCurrentContext() (ConfigContext, bool) {
	contexts := GetContexts()
	if len(contexts) == 0 {
		return ConfigContext{}, false
	}

	currentSelector := normalizeContextSelector(viper.GetString(ConfigKeyCurrentContext))
	if currentSelector == "" {
		return ConfigContext{}, false
	}

	for _, context := range contexts {
		if ContextSelector(context) == currentSelector {
			return context, true
		}
	}

	return ConfigContext{}, false
}

func WriteConfig() error {
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write configuration: %w", err)
	}

	return nil
}

func SaveContexts(contexts []ConfigContext) error {
	viper.Set(ConfigKeyContexts, contexts)
	return WriteConfig()
}

// SwitchContext makes the context identified by (baseURL, org) current. The
// org argument matches on organization ID first and then on organization name.
func SwitchContext(baseURL, org string) (*ConfigContext, error) {
	contexts := GetContexts()
	if len(contexts) == 0 {
		return nil, fmt.Errorf("no contexts configured")
	}

	url := normalizeBaseURL(baseURL)
	name := strings.TrimSpace(org)
	if url == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if name == "" {
		return nil, fmt.Errorf("organization is required")
	}

	for i, c := range contexts {
		if c.URL == url && c.OrganizationID != "" && c.OrganizationID == name {
			return saveCurrent(contexts[i])
		}
	}
	for i, c := range contexts {
		if c.URL == url && c.Organization == name {
			return saveCurrent(contexts[i])
		}
	}

	return nil, fmt.Errorf("no context found for %s %q", url, name)
}

func contextMatchesOrgArg(c ConfigContext, orgOrID string) bool {
	if c.OrganizationID != "" && c.OrganizationID == orgOrID {
		return true
	}
	return c.Organization == orgOrID
}

func distinctBaseURLs(contexts []ConfigContext) []string {
	seen := make(map[string]struct{}, len(contexts))
	for _, c := range contexts {
		if c.URL != "" {
			seen[c.URL] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for u := range seen {
		out = append(out, u)
	}
	sort.Strings(out)
	return out
}

func formatURLBulletList(urls []string) string {
	var b strings.Builder
	for _, u := range urls {
		b.WriteString("- ")
		b.WriteString(u)
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// SwitchContextByOrganization sets the current context using organization id or
// name across saved installations. When urlFilter is non-empty, only contexts
// at that base URL are considered (so multiple matches are always same-installation
// duplicates, never cross-installation ambiguity). Multiple matches on different
// base URLs without urlFilter returns an error that lists installations and suggests --url.
func SwitchContextByOrganization(orgOrID, urlFilter string) (*ConfigContext, error) {
	orgOrID = strings.TrimSpace(orgOrID)
	if orgOrID == "" {
		return nil, fmt.Errorf("organization id or name is required")
	}

	contexts := GetContexts()
	if len(contexts) == 0 {
		return nil, fmt.Errorf("no contexts configured")
	}

	wantURL := normalizeBaseURL(urlFilter)
	var candidates []ConfigContext
	for _, c := range contexts {
		if !contextMatchesOrgArg(c, orgOrID) {
			continue
		}
		if wantURL != "" && c.URL != wantURL {
			continue
		}
		candidates = append(candidates, c)
	}

	if len(candidates) == 0 {
		if wantURL != "" {
			return nil, fmt.Errorf("no context found for %q at %s", orgOrID, wantURL)
		}
		return nil, fmt.Errorf("no context found for organization %q", orgOrID)
	}

	if len(candidates) == 1 {
		return saveCurrent(candidates[0])
	}

	urls := distinctBaseURLs(candidates)
	if len(urls) == 1 {
		return nil, fmt.Errorf(
			"multiple saved contexts match %q at %s; remove duplicate entries from your config",
			orgOrID,
			urls[0],
		)
	}

	return nil, fmt.Errorf(
		"ambiguous organization %q matches multiple installations:\n%s\n\nUse: superplane context %q --url <base-url>",
		orgOrID,
		formatURLBulletList(urls),
		orgOrID,
	)
}

func saveCurrent(selected ConfigContext) (*ConfigContext, error) {
	viper.Set(ConfigKeyCurrentContext, ContextSelector(selected))
	if err := WriteConfig(); err != nil {
		return nil, err
	}
	return &selected, nil
}

func UpsertContext(context ConfigContext) (ConfigContext, error) {
	context = normalizeContext(context)
	if context.URL == "" {
		return ConfigContext{}, fmt.Errorf("organization URL is required")
	}
	if context.APIToken == "" {
		return ConfigContext{}, fmt.Errorf("API token is required")
	}

	contexts := GetContexts()
	existingIndex := findMatchingContextIndex(contexts, context)

	if existingIndex >= 0 {
		contexts[existingIndex] = context
	} else {
		contexts = append(contexts, context)
	}

	viper.Set(ConfigKeyContexts, contexts)
	viper.Set(ConfigKeyCurrentContext, ContextSelector(context))
	if err := WriteConfig(); err != nil {
		return ConfigContext{}, err
	}

	return context, nil
}

func findMatchingContextIndex(contexts []ConfigContext, context ConfigContext) int {
	selector := ContextSelector(context)
	for i, existing := range contexts {
		if ContextSelector(existing) == selector {
			return i
		}
	}

	if context.OrganizationID == "" || context.Organization == "" {
		return -1
	}

	for i, existing := range contexts {
		if existing.OrganizationID != "" {
			continue
		}
		if existing.URL == context.URL && existing.Organization == context.Organization {
			return i
		}
	}

	return -1
}

/*
 * Implementation of the core.ConfigContext interface,
 * which uses the current context as the source for operations..
 */
type CurrentContext struct {
	context  ConfigContext
	readOnly bool
}

func NewCurrentContext(context ConfigContext) core.ConfigContext {
	return &CurrentContext{context: context}
}

func NewEnvironmentContext(context ConfigContext) core.ConfigContext {
	return &CurrentContext{context: context, readOnly: true}
}

func (c *CurrentContext) GetActiveApp() string {
	return activeAppID(c.context)
}

func (c *CurrentContext) SetActiveApp(appID string) error {
	if c.readOnly {
		return fmt.Errorf("cannot set active app when using %s and %s; pass --app-id instead", EnvURL, EnvToken)
	}

	appID = strings.TrimSpace(appID)
	c.context.App = &appID
	c.context.Canvas = nil
	_, err := UpsertContext(c.context)
	return err
}

func (c *CurrentContext) GetURL() string {
	return c.context.URL
}
