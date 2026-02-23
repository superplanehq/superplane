package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

type LoadedPlugin struct {
	Manifest  *PluginManifest
	Dir       string
	Activated bool
}

// CallContext holds the Go-side context objects for an active RPC call.
// The Plugin Host sends callbacks with the callId so we can route
// operations like secrets, webhooks, metadata back to the right context.
type CallContext struct {
	Webhook     core.NodeWebhookContext
	HTTP        core.HTTPContext
	Metadata    core.MetadataContext
	Events      core.EventContext
	Secrets     core.SecretsContext
	Integration core.IntegrationContext
	// WebhookCtx is for the webhook provisioner (WebhookHandler.Setup/Cleanup)
	WebhookCtx core.WebhookContext
}

type Manager struct {
	pluginsDir string
	registry   *registry.Registry

	mu      sync.RWMutex
	plugins map[string]*LoadedPlugin
	host    *PluginHostProcess

	activeContexts sync.Map // map[string]*CallContext

	pluginHostPath string
	crashCount     int
	lastCrash      time.Time
}

// RPCResult represents the result returned from a Plugin Host call.
type RPCResult struct {
	Action      string `json:"action"`
	Channel     string `json:"channel"`
	PayloadType string `json:"payloadType"`
	Data        any    `json:"data"`
	Reason      string `json:"reason"`
	Message     string `json:"message"`
	Key         string `json:"key"`
	Value       string `json:"value"`
}

func NewManager(pluginsDir string, reg *registry.Registry) (*Manager, error) {
	hostPath := resolvePluginHostPath()

	m := &Manager{
		pluginsDir:     pluginsDir,
		registry:       reg,
		plugins:        make(map[string]*LoadedPlugin),
		pluginHostPath: hostPath,
	}

	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		log.Infof("Plugin directory %s does not exist, skipping .spx plugin loading", pluginsDir)
	} else {
		if err := m.loadPlugins(); err != nil {
			return nil, fmt.Errorf("loading plugins: %w", err)
		}

		if len(m.plugins) > 0 {
			if err := m.startHost(); err != nil {
				log.WithError(err).Error("Failed to start Plugin Host, plugin execution will be unavailable")
			}
		}
	}

	return m, nil
}

func resolvePluginHostPath() string {
	if p := os.Getenv("SUPERPLANE_PLUGIN_HOST_PATH"); p != "" {
		return p
	}

	candidates := []string{
		"plugin-host/dist/index.js",
		"/app/plugin-host/dist/index.js",
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return "plugin-host/dist/index.js"
}

func (m *Manager) loadPlugins() error {
	pj, err := ReadPluginsJSON(m.pluginsDir)
	if err != nil {
		return err
	}

	if len(pj.Plugins) == 0 {
		log.Info("No plugins installed")
		return nil
	}

	for _, record := range pj.Plugins {
		pluginDir := filepath.Join(m.pluginsDir, record.Name)
		manifest, err := ParseManifest(pluginDir)
		if err != nil {
			log.WithError(err).Errorf("Failed to parse plugin manifest: %s", record.Name)
			continue
		}

		if err := ValidateManifest(manifest); err != nil {
			log.WithError(err).Errorf("Invalid plugin manifest: %s", record.Name)
			continue
		}

		extensionPath := filepath.Join(pluginDir, "extension.js")
		if _, err := os.Stat(extensionPath); os.IsNotExist(err) {
			log.Errorf("Plugin %s missing extension.js", record.Name)
			continue
		}

		m.plugins[record.Name] = &LoadedPlugin{
			Manifest: manifest,
			Dir:      pluginDir,
		}

		m.registerContributions(manifest, record.Name)
		log.Infof("Plugin loaded: %s v%s", record.Name, record.Version)
	}

	log.Infof("Loaded %d plugins", len(m.plugins))
	return nil
}

func (m *Manager) registerContributions(manifest *PluginManifest, pluginName string) {
	components := make([]core.Component, 0, len(manifest.SuperPlane.Contributes.Components))
	for _, comp := range manifest.SuperPlane.Contributes.Components {
		adapter := NewPluginComponentAdapter(comp, pluginName, m)
		components = append(components, adapter)
		log.Infof("  Registered component: %s", comp.Name)
	}

	triggers := make([]core.Trigger, 0, len(manifest.SuperPlane.Contributes.Triggers))
	for _, trig := range manifest.SuperPlane.Contributes.Triggers {
		adapter := NewPluginTriggerAdapter(trig, pluginName, m)
		triggers = append(triggers, adapter)
		log.Infof("  Registered trigger: %s", trig.Name)
	}

	integrationMeta := manifest.SuperPlane.Integration
	if integrationMeta.Name == "" {
		integrationMeta.Name = pluginName
	}
	if integrationMeta.Label == "" {
		integrationMeta.Label = manifest.Name
	}

	integration := NewPluginIntegrationAdapter(integrationMeta, pluginName, m, components, triggers)
	m.registry.Integrations[integrationMeta.Name] = integration
	log.Infof("  Registered integration: %s (%s)", integrationMeta.Name, integrationMeta.Label)

	if integrationMeta.HasWebhookHandler {
		webhookHandler := NewPluginWebhookHandler(integrationMeta.Name, pluginName, m)
		m.registry.WebhookHandlers[integrationMeta.Name] = webhookHandler
		log.Infof("  Registered webhook handler: %s", integrationMeta.Name)
	}
}

func (m *Manager) unregisterContributions(manifest *PluginManifest) {
	integrationName := manifest.SuperPlane.Integration.Name
	if integrationName == "" {
		integrationName = manifest.Name
	}
	delete(m.registry.Integrations, integrationName)
	delete(m.registry.WebhookHandlers, integrationName)
}

func (m *Manager) startHost() error {
	if _, err := os.Stat(m.pluginHostPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin host not found at %s", m.pluginHostPath)
	}

	host, err := SpawnPluginHost(m.pluginHostPath, m.pluginsDir, m.handleContextCallback)
	if err != nil {
		return err
	}

	m.host = host

	go m.watchHost()

	for name, plugin := range m.plugins {
		for _, event := range plugin.Manifest.SuperPlane.ActivationEvents {
			if event == "*" {
				if err := m.activatePlugin(name); err != nil {
					log.WithError(err).Errorf("Failed to activate plugin %s", name)
				}
				break
			}
		}
	}

	return nil
}

func (m *Manager) watchHost() {
	<-m.host.Done()

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if now.Sub(m.lastCrash) < time.Minute {
		m.crashCount++
	} else {
		m.crashCount = 1
	}
	m.lastCrash = now

	if m.crashCount >= 5 {
		log.Error("Plugin Host crashed 5 times within 60 seconds, stopping restart attempts")
		return
	}

	log.Warn("Plugin Host process exited unexpectedly, restarting in 1 second...")
	time.Sleep(time.Second)

	for _, plugin := range m.plugins {
		plugin.Activated = false
	}

	if err := m.startHost(); err != nil {
		log.WithError(err).Error("Failed to restart Plugin Host")
	}
}

func (m *Manager) EnsureActivated(pluginName string) error {
	m.mu.RLock()
	plugin, ok := m.plugins[pluginName]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin %s not loaded", pluginName)
	}

	if plugin.Activated {
		return nil
	}

	return m.activatePlugin(pluginName)
}

func (m *Manager) activatePlugin(pluginName string) error {
	m.mu.RLock()
	plugin, ok := m.plugins[pluginName]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin %s not loaded", pluginName)
	}

	if m.host == nil {
		return fmt.Errorf("plugin host not running")
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultActivationTimeout)
	defer cancel()

	_, err := m.host.Call(ctx, "plugin/activate", map[string]any{
		"pluginId":   pluginName,
		"pluginPath": plugin.Dir,
	})

	if err != nil {
		return fmt.Errorf("activating plugin %s: %w", pluginName, err)
	}

	m.mu.Lock()
	plugin.Activated = true
	m.mu.Unlock()

	log.Infof("Plugin activated: %s", pluginName)
	return nil
}

// CallPluginWithContext sends an RPC call to the Plugin Host, registering the
// Go-side context objects so that callbacks (ctx/webhook.setup, ctx/secrets.getKey, etc.)
// can be routed to the correct context.
func (m *Manager) CallPluginRaw(method string, pluginName string, params map[string]any, callCtx *CallContext) (json.RawMessage, error) {
	if err := m.EnsureActivated(pluginName); err != nil {
		return nil, fmt.Errorf("activating plugin %s: %w", pluginName, err)
	}

	params["pluginId"] = pluginName

	if callCtx != nil {
		callID := uuid.New().String()
		m.activeContexts.Store(callID, callCtx)
		defer m.activeContexts.Delete(callID)

		ctxMap, ok := params["context"].(map[string]any)
		if !ok {
			ctxMap = map[string]any{}
			params["context"] = ctxMap
		}
		ctxMap["callId"] = callID
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultExecutionTimeout)
	defer cancel()

	raw, err := m.host.Call(ctx, method, params)
	if err != nil {
		return nil, fmt.Errorf("plugin error: %w", err)
	}

	return raw, nil
}

func (m *Manager) CallPluginWithContext(method string, pluginName string, params map[string]any, callCtx *CallContext) (*RPCResult, error) {
	if err := m.EnsureActivated(pluginName); err != nil {
		return nil, err
	}

	callID := uuid.New().String()

	if callCtx != nil {
		m.activeContexts.Store(callID, callCtx)
		defer m.activeContexts.Delete(callID)
	}

	params["pluginId"] = pluginName

	// Inject the callId into the context sub-map so the Plugin Host
	// passes it back in all callbacks.
	if ctxMap, ok := params["context"].(map[string]any); ok {
		ctxMap["callId"] = callID
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultExecutionTimeout)
	defer cancel()

	raw, err := m.host.Call(ctx, method, params)
	if err != nil {
		return nil, err
	}

	if raw == nil || string(raw) == "null" {
		return nil, nil
	}

	var result RPCResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, nil
	}

	return &result, nil
}

func (m *Manager) getCallContext(params json.RawMessage) (*CallContext, error) {
	var p struct {
		ContextID string `json:"contextId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	v, ok := m.activeContexts.Load(p.ContextID)
	if !ok {
		return nil, fmt.Errorf("no active context for callId %s", p.ContextID)
	}

	return v.(*CallContext), nil
}

func (m *Manager) handleContextCallback(method string, params json.RawMessage) (any, error) {
	log.Debugf("Plugin Host callback: %s", method)

	switch method {
	case "ctx/log":
		var p struct {
			Level   string `json:"level"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		log.WithField("source", "plugin").Info(p.Message)
		return nil, nil

	case "ctx/webhook.setup":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Webhook == nil {
			return nil, fmt.Errorf("webhook context not available")
		}
		url, err := callCtx.Webhook.Setup()
		if err != nil {
			return nil, err
		}
		return url, nil

	case "ctx/webhook.getSecret":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Webhook == nil {
			return nil, fmt.Errorf("webhook context not available")
		}
		secret, err := callCtx.Webhook.GetSecret()
		if err != nil {
			return nil, err
		}
		return string(secret), nil

	case "ctx/secrets.getKey":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Secrets == nil {
			return nil, fmt.Errorf("secrets context not available")
		}
		var p struct {
			SecretName string `json:"secretName"`
			KeyName    string `json:"keyName"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		val, err := callCtx.Secrets.GetKey(p.SecretName, p.KeyName)
		if err != nil {
			return nil, err
		}
		return string(val), nil

	case "ctx/metadata.get":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Metadata == nil {
			return nil, nil
		}
		return callCtx.Metadata.Get(), nil

	case "ctx/metadata.set":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Metadata == nil {
			return nil, fmt.Errorf("metadata context not available")
		}
		var p struct {
			Value any `json:"value"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, callCtx.Metadata.Set(p.Value)

	case "ctx/events.emit":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Events == nil {
			return nil, fmt.Errorf("events context not available")
		}
		var p struct {
			PayloadType string `json:"payloadType"`
			Payload     any    `json:"payload"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, callCtx.Events.Emit(p.PayloadType, p.Payload)

	case "ctx/http.request":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.HTTP == nil {
			return nil, fmt.Errorf("http context not available")
		}
		return m.handleHTTPCallback(callCtx.HTTP, params)

	case "ctx/integration.getMetadata":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Integration == nil {
			return nil, fmt.Errorf("integration context not available")
		}
		return callCtx.Integration.GetMetadata(), nil

	case "ctx/integration.setMetadata":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Integration == nil {
			return nil, fmt.Errorf("integration context not available")
		}
		var p struct {
			Value any `json:"value"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		callCtx.Integration.SetMetadata(p.Value)
		return nil, nil

	case "ctx/integration.getConfig":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Integration == nil {
			return nil, fmt.Errorf("integration context not available")
		}
		var p struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		val, err := callCtx.Integration.GetConfig(p.Name)
		if err != nil {
			return nil, err
		}
		return string(val), nil

	case "ctx/integration.setSecret":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Integration == nil {
			return nil, fmt.Errorf("integration context not available")
		}
		var p struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, callCtx.Integration.SetSecret(p.Name, []byte(p.Value))

	case "ctx/integration.getSecrets":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Integration == nil {
			return nil, fmt.Errorf("integration context not available")
		}
		secrets, err := callCtx.Integration.GetSecrets()
		if err != nil {
			return nil, err
		}
		result := make([]map[string]string, len(secrets))
		for i, s := range secrets {
			result[i] = map[string]string{"name": s.Name, "value": string(s.Value)}
		}
		return result, nil

	case "ctx/integration.newBrowserAction":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Integration == nil {
			return nil, fmt.Errorf("integration context not available")
		}
		var p struct {
			Description string            `json:"description"`
			URL         string            `json:"url"`
			Method      string            `json:"method"`
			FormFields  map[string]string `json:"formFields"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		callCtx.Integration.NewBrowserAction(core.BrowserAction{
			Description: p.Description,
			URL:         p.URL,
			Method:      p.Method,
			FormFields:  p.FormFields,
		})
		return nil, nil

	case "ctx/integration.removeBrowserAction":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Integration == nil {
			return nil, fmt.Errorf("integration context not available")
		}
		callCtx.Integration.RemoveBrowserAction()
		return nil, nil

	case "ctx/integration.ready":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Integration == nil {
			return nil, fmt.Errorf("integration context not available")
		}
		callCtx.Integration.Ready()
		return nil, nil

	case "ctx/integration.error":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Integration == nil {
			return nil, fmt.Errorf("integration context not available")
		}
		var p struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		callCtx.Integration.Error(p.Message)
		return nil, nil

	case "ctx/integration.requestWebhook":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Integration == nil {
			return nil, fmt.Errorf("integration context not available")
		}
		var p struct {
			Configuration any `json:"configuration"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, callCtx.Integration.RequestWebhook(p.Configuration)

	case "ctx/integration.id":
		callCtx, err := m.getCallContext(params)
		if err != nil {
			return nil, err
		}
		if callCtx.Integration == nil {
			return nil, fmt.Errorf("integration context not available")
		}
		return callCtx.Integration.ID().String(), nil

	default:
		return nil, fmt.Errorf("unknown callback method: %s", method)
	}
}

func (m *Manager) handleHTTPCallback(httpCtx core.HTTPContext, params json.RawMessage) (any, error) {
	var p struct {
		Method  string `json:"method"`
		URL     string `json:"url"`
		Options struct {
			Headers map[string]string `json:"headers"`
			Body    string            `json:"body"`
			Timeout int               `json:"timeout"`
		} `json:"options"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	req, err := newHTTPRequest(p.Method, p.URL, p.Options.Body, p.Options.Headers)
	if err != nil {
		return nil, err
	}

	resp, err := httpCtx.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := readResponseBody(resp)

	return map[string]any{
		"status":  resp.StatusCode,
		"headers": flattenHeaders(resp.Header),
		"body":    body,
	}, nil
}

func (m *Manager) Reload() {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Info("Reloading plugins...")

	for _, plugin := range m.plugins {
		m.unregisterContributions(plugin.Manifest)
	}

	if m.host != nil {
		m.host.Kill()
		m.host = nil
	}

	m.plugins = make(map[string]*LoadedPlugin)

	if err := m.loadPlugins(); err != nil {
		log.WithError(err).Error("Failed to reload plugins")
		return
	}

	if len(m.plugins) > 0 {
		if err := m.startHost(); err != nil {
			log.WithError(err).Error("Failed to restart Plugin Host after reload")
		}
	}

	log.Info("Plugin reload complete")
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.host != nil {
		log.Info("Shutting down Plugin Host")
		m.host.Kill()
		m.host = nil
	}
}

// LoadScriptsFromDB loads all active scripts from the database and registers
// their contributions. Call this after NewManager during server startup.
func (m *Manager) LoadScriptsFromDB() error {
	scripts, err := models.FindActiveScripts()
	if err != nil {
		return fmt.Errorf("querying active scripts: %w", err)
	}

	if len(scripts) == 0 {
		return nil
	}

	for _, script := range scripts {
		scriptID := "script." + script.Name
		manifest, err := parseScriptManifest(script.Manifest)
		if err != nil {
			log.WithError(err).Errorf("Failed to parse manifest for script %s", script.Name)
			continue
		}

		m.registerScriptContributions(scriptID, manifest)

		m.mu.Lock()
		m.plugins[scriptID] = &LoadedPlugin{
			Manifest:  manifest,
			Dir:       "",
			Activated: false,
		}
		m.mu.Unlock()

		log.Infof("Script loaded from DB: %s", scriptID)
	}

	if m.host == nil && len(m.plugins) > 0 {
		if err := m.startHost(); err != nil {
			return fmt.Errorf("starting plugin host for scripts: %w", err)
		}
	}

	for _, script := range scripts {
		scriptID := "script." + script.Name
		if err := m.activateScriptInline(scriptID, script.Source); err != nil {
			log.WithError(err).Errorf("Failed to activate script %s", scriptID)
		}
	}

	return nil
}

// ActivateScript registers and activates an in-app script.
func (m *Manager) ActivateScript(name string, source string, manifestJSON []byte) error {
	scriptID := "script." + name
	log.Infof("ActivateScript called: scriptID=%s, sourceLen=%d", scriptID, len(source))

	// Unregister any previous contributions for this script
	m.mu.RLock()
	existingPlugin, exists := m.plugins[scriptID]
	m.mu.RUnlock()
	if exists && existingPlugin.Manifest != nil {
		m.unregisterScriptContributions(scriptID, existingPlugin.Manifest)
	}

	m.mu.Lock()
	m.plugins[scriptID] = &LoadedPlugin{
		Manifest:  &PluginManifest{},
		Dir:       "",
		Activated: false,
	}
	m.mu.Unlock()

	if m.host == nil {
		log.Infof("Plugin host not running, starting it now (hostPath=%s)", m.pluginHostPath)
		if err := m.startHost(); err != nil {
			return fmt.Errorf("starting plugin host: %w", err)
		}
		log.Info("Plugin host started successfully")
	}

	return m.activateScriptInline(scriptID, source)
}

// DeactivateScript deactivates and unregisters an in-app script.
func (m *Manager) DeactivateScript(name string) error {
	scriptID := "script." + name

	m.mu.RLock()
	plugin, ok := m.plugins[scriptID]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	if m.host != nil && plugin.Activated {
		ctx, cancel := context.WithTimeout(context.Background(), DefaultActivationTimeout)
		defer cancel()
		_, _ = m.host.Call(ctx, "plugin/deactivate", map[string]any{
			"pluginId": scriptID,
		})
	}

	if plugin.Manifest != nil {
		m.unregisterScriptContributions(scriptID, plugin.Manifest)
	}

	m.mu.Lock()
	delete(m.plugins, scriptID)
	m.mu.Unlock()

	return nil
}

// UpdateScript deactivates the old script and activates the new one.
func (m *Manager) UpdateScript(name string, source string, manifestJSON []byte) error {
	if err := m.DeactivateScript(name); err != nil {
		return err
	}
	return m.ActivateScript(name, source, manifestJSON)
}

func (m *Manager) activateScriptInline(scriptID, source string) error {
	log.Infof("activateScriptInline called: scriptID=%s", scriptID)

	if m.host == nil {
		if err := m.startHost(); err != nil {
			return fmt.Errorf("starting plugin host for script activation: %w", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultActivationTimeout)
	defer cancel()

	result, err := m.host.Call(ctx, "plugin/activateInline", map[string]any{
		"pluginId": scriptID,
		"source":   source,
		"manifest": map[string]any{},
	})
	if err != nil {
		return fmt.Errorf("activating script %s: %w", scriptID, err)
	}

	var parsedResult map[string]any
	if err := json.Unmarshal(result, &parsedResult); err != nil {
		log.WithError(err).Warnf("Failed to parse activation result for %s", scriptID)
		parsedResult = nil
	}
	log.Infof("Plugin host returned result for %s: %v", scriptID, parsedResult)

	manifest := m.buildManifestFromActivationResult(parsedResult)
	log.Infof("Built manifest for %s: components=%d, triggers=%d",
		scriptID,
		len(manifest.SuperPlane.Contributes.Components),
		len(manifest.SuperPlane.Contributes.Triggers))

	m.registerScriptContributions(scriptID, manifest)

	m.mu.Lock()
	if p, ok := m.plugins[scriptID]; ok {
		p.Activated = true
		p.Manifest = manifest
	}
	m.mu.Unlock()

	log.Infof("Script activated: %s", scriptID)
	return nil
}

func (m *Manager) buildManifestFromActivationResult(result any) *PluginManifest {
	manifest := &PluginManifest{}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return manifest
	}

	if comps, ok := resultMap["components"].([]any); ok {
		for _, c := range comps {
			comp, ok := c.(map[string]any)
			if !ok {
				continue
			}

			contribution := ComponentContribution{
				Name:        stringFromMap(comp, "name"),
				Label:       stringFromMap(comp, "label"),
				Description: stringFromMap(comp, "description"),
				Icon:        stringFromMap(comp, "icon"),
				Color:       stringFromMap(comp, "color"),
			}

			if channels, ok := comp["outputChannels"].([]any); ok {
				for _, ch := range channels {
					if chMap, ok := ch.(map[string]any); ok {
						contribution.OutputChannels = append(contribution.OutputChannels, OutputChannelManifest{
							Name:  stringFromMap(chMap, "name"),
							Label: stringFromMap(chMap, "label"),
						})
					}
				}
			}

			if configs, ok := comp["configuration"].([]any); ok {
				for _, cfg := range configs {
					if cfgMap, ok := cfg.(map[string]any); ok {
						field := configuration.Field{
							Name:  stringFromMap(cfgMap, "name"),
							Label: stringFromMap(cfgMap, "label"),
							Type:  stringFromMap(cfgMap, "type"),
						}
						if req, ok := cfgMap["required"].(bool); ok {
							field.Required = req
						}
						contribution.Configuration = append(contribution.Configuration, field)
					}
				}
			}

			manifest.SuperPlane.Contributes.Components = append(manifest.SuperPlane.Contributes.Components, contribution)
		}
	}

	if trigs, ok := resultMap["triggers"].([]any); ok {
		for _, t := range trigs {
			trig, ok := t.(map[string]any)
			if !ok {
				continue
			}

			contribution := TriggerContribution{
				Name:        stringFromMap(trig, "name"),
				Label:       stringFromMap(trig, "label"),
				Description: stringFromMap(trig, "description"),
				Icon:        stringFromMap(trig, "icon"),
				Color:       stringFromMap(trig, "color"),
			}

			if configs, ok := trig["configuration"].([]any); ok {
				for _, cfg := range configs {
					if cfgMap, ok := cfg.(map[string]any); ok {
						field := configuration.Field{
							Name:  stringFromMap(cfgMap, "name"),
							Label: stringFromMap(cfgMap, "label"),
							Type:  stringFromMap(cfgMap, "type"),
						}
						if req, ok := cfgMap["required"].(bool); ok {
							field.Required = req
						}
						contribution.Configuration = append(contribution.Configuration, field)
					}
				}
			}

			manifest.SuperPlane.Contributes.Triggers = append(manifest.SuperPlane.Contributes.Triggers, contribution)
		}
	}

	return manifest
}

func stringFromMap(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func (m *Manager) registerScriptContributions(scriptID string, manifest *PluginManifest) {
	for _, comp := range manifest.SuperPlane.Contributes.Components {
		adapter := NewPluginComponentAdapter(comp, scriptID, m)
		m.registry.Components[comp.Name] = adapter
		log.Infof("  Registered script component: %s", comp.Name)
	}

	for _, trig := range manifest.SuperPlane.Contributes.Triggers {
		adapter := NewPluginTriggerAdapter(trig, scriptID, m)
		m.registry.Triggers[trig.Name] = adapter
		log.Infof("  Registered script trigger: %s", trig.Name)
	}
}

func (m *Manager) unregisterScriptContributions(scriptID string, manifest *PluginManifest) {
	for _, comp := range manifest.SuperPlane.Contributes.Components {
		delete(m.registry.Components, comp.Name)
	}
	for _, trig := range manifest.SuperPlane.Contributes.Triggers {
		delete(m.registry.Triggers, trig.Name)
	}
}

func parseScriptManifest(manifestJSON []byte) (*PluginManifest, error) {
	if len(manifestJSON) == 0 || string(manifestJSON) == "{}" {
		return &PluginManifest{}, nil
	}

	var superplaneSection SuperPlaneManifest
	if err := json.Unmarshal(manifestJSON, &superplaneSection); err != nil {
		return nil, fmt.Errorf("parsing manifest JSON: %w", err)
	}

	return &PluginManifest{
		SuperPlane: superplaneSection,
	}, nil
}
