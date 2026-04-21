package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type ConfigContext struct {
	URL            string  `json:"url" yaml:"url"`
	Organization   string  `json:"organization" yaml:"organization"`
	OrganizationID string  `json:"organizationId,omitempty" yaml:"organizationId,omitempty"`
	APIToken       string  `json:"apiToken" yaml:"apiToken"`
	Canvas         *string `json:"canvas,omitempty" yaml:"canvas,omitempty"`
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
	context ConfigContext
}

func NewCurrentContext(context ConfigContext) core.ConfigContext {
	return &CurrentContext{context: context}
}

func (c *CurrentContext) GetActiveCanvas() string {
	if c.context.Canvas == nil {
		return ""
	}

	return *c.context.Canvas
}

func (c *CurrentContext) SetActiveCanvas(canvasID string) error {
	c.context.Canvas = &canvasID
	_, err := UpsertContext(c.context)
	return err
}
