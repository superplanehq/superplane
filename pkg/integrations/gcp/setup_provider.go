package gcp

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/artifactregistry"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/cloudbuild"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/clouddns"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/cloudfunctions"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/compute"
	gcppubsub "github.com/superplanehq/superplane/pkg/integrations/gcp/pubsub"
	"github.com/superplanehq/superplane/pkg/oidc"
)

const (
	SetupStepCapabilitySelection    = "capabilitySelection"
	SetupStepSelectConnectionMethod = "selectConnectionMethod"
	SetupStepServiceAccountKey      = "enterServiceAccountKey"
	SetupStepWIFProvider            = "enterWIFProvider"
	SetupStepWIFServiceAccount      = "enterWIFServiceAccount"

	PropertyConnectionMethod    = "connectionMethod"
	PropertyProjectID           = "projectId"
	PropertyClientEmail         = "clientEmail"
	PropertyWIFProvider         = "workloadIdentityProvider"
	PropertyServiceAccountEmail = "serviceAccountEmail"
)

//go:embed templates/sa-key-instructions.tpl
var saKeyInstructionsTemplate []byte

//go:embed templates/wif-provider-instructions.tpl
var wifProviderInstructionsTemplate []byte

//go:embed templates/wif-service-account-instructions.tpl
var wifServiceAccountInstructionsTemplate []byte

//go:embed templates/setup-complete-sak.tpl
var setupCompleteSAKTemplate []byte

//go:embed templates/setup-complete-wif.tpl
var setupCompleteWIFTemplate []byte

type SetupProvider struct{}

func (s *SetupProvider) CapabilityGroups() []core.CapabilityGroup {
	return []core.CapabilityGroup{
		{
			Label: "Compute Engine",
			Capabilities: genCapabilities(
				[]core.Action{&compute.CreateVM{}},
				[]core.Trigger{&compute.OnVMInstance{}},
			),
		},
		{
			Label: "Cloud Build",
			Capabilities: genCapabilities(
				[]core.Action{&cloudbuild.CreateBuild{}, &cloudbuild.GetBuild{}, &cloudbuild.RunTrigger{}},
				[]core.Trigger{&cloudbuild.OnBuildComplete{}},
			),
		},
		{
			Label: "Artifact Registry",
			Capabilities: genCapabilities(
				[]core.Action{&artifactregistry.GetArtifact{}, &artifactregistry.GetArtifactAnalysis{}},
				[]core.Trigger{&artifactregistry.OnArtifactPush{}, &artifactregistry.OnArtifactAnalysis{}},
			),
		},
		{
			Label: "Pub/Sub",
			Capabilities: genCapabilities(
				[]core.Action{
					&gcppubsub.PublishMessage{},
					&gcppubsub.CreateTopicComponent{},
					&gcppubsub.DeleteTopicComponent{},
					&gcppubsub.CreateSubscriptionComponent{},
					&gcppubsub.DeleteSubscriptionComponent{},
				},
				[]core.Trigger{&gcppubsub.OnMessage{}},
			),
		},
		{
			Label: "Cloud Functions",
			Capabilities: genCapabilities(
				[]core.Action{&cloudfunctions.InvokeFunction{}},
				nil,
			),
		},
		{
			Label: "Cloud DNS",
			Capabilities: genCapabilities(
				[]core.Action{&clouddns.CreateRecord{}, &clouddns.DeleteRecord{}, &clouddns.UpdateRecord{}},
				nil,
			),
		},
	}
}

func genCapabilities(actions []core.Action, triggers []core.Trigger) []core.Capability {
	caps := []core.Capability{}
	for _, a := range actions {
		caps = append(caps, core.Capability{
			Type:           core.IntegrationCapabilityTypeAction,
			Name:           a.Name(),
			Label:          a.Label(),
			Description:    a.Description(),
			Configuration:  a.Configuration(),
			OutputChannels: a.OutputChannels(nil),
		})
	}
	for _, t := range triggers {
		caps = append(caps, core.Capability{
			Type:          core.IntegrationCapabilityTypeTrigger,
			Name:          t.Name(),
			Label:         t.Label(),
			Description:   t.Description(),
			Configuration: t.Configuration(),
		})
	}
	return caps
}

func (s *SetupProvider) allCapabilityNames() []string {
	var names []string
	for _, group := range s.CapabilityGroups() {
		for _, cap := range group.Capabilities {
			names = append(names, cap.Name)
		}
	}
	return names
}

func (s *SetupProvider) capabilityDiff(selected []string) []string {
	var diff []string
	for _, name := range s.allCapabilityNames() {
		if !slices.Contains(selected, name) {
			diff = append(diff, name)
		}
	}
	return diff
}

func (s *SetupProvider) FirstStep(_ core.SetupStepContext) core.SetupStep {
	return core.SetupStep{
		Type:         core.SetupStepTypeCapabilitySelection,
		Name:         SetupStepCapabilitySelection,
		Label:        "Select capabilities",
		Capabilities: s.allCapabilityNames(),
	}
}

func (s *SetupProvider) OnStepSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	switch ctx.Step.Name {
	case SetupStepCapabilitySelection:
		return s.onCapabilitySelectionSubmit(ctx)
	case SetupStepSelectConnectionMethod:
		return s.onSelectConnectionMethodSubmit(ctx)
	case SetupStepServiceAccountKey:
		return s.onEnterServiceAccountKeySubmit(ctx)
	case SetupStepWIFProvider:
		return s.onEnterWIFProviderSubmit(ctx)
	case SetupStepWIFServiceAccount:
		return s.onEnterWIFServiceAccountSubmit(ctx)
	}
	return nil, errors.New("unknown step")
}

func (s *SetupProvider) OnStepRevert(ctx core.SetupStepContext) error {
	switch ctx.Step.Name {
	case SetupStepCapabilitySelection:
		ctx.Capabilities.Clear()
		return nil
	case SetupStepSelectConnectionMethod:
		return ctx.Properties.Delete(PropertyConnectionMethod)
	case SetupStepServiceAccountKey:
		_ = ctx.Secrets.Delete(gcpcommon.SecretNameServiceAccountKey)
		return ctx.Properties.Delete(PropertyProjectID, PropertyClientEmail)
	case SetupStepWIFProvider:
		return ctx.Properties.Delete(PropertyWIFProvider, PropertyProjectID)
	case SetupStepWIFServiceAccount:
		return ctx.Properties.Delete(PropertyServiceAccountEmail)
	}
	return errors.New("unknown step")
}

func (s *SetupProvider) OnPropertyUpdate(_ core.PropertyUpdateContext) (*core.SetupStep, error) {
	return nil, fmt.Errorf("property updates are not supported for GCP")
}

func (s *SetupProvider) OnSecretUpdate(ctx core.SecretUpdateContext) (*core.SetupStep, error) {
	if ctx.SecretName != gcpcommon.SecretNameServiceAccountKey {
		return nil, fmt.Errorf("unknown secret: %s", ctx.SecretName)
	}

	keyStr := strings.TrimSpace(ctx.Value)
	if keyStr == "" {
		return nil, fmt.Errorf("service account key is required")
	}

	metadata, err := validateAndParseServiceAccountKey([]byte(keyStr))
	if err != nil {
		return nil, fmt.Errorf("invalid service account key: %w", err)
	}

	storedProjectID, _ := ctx.Properties.GetString(PropertyProjectID)
	if storedProjectID != "" && storedProjectID != metadata.ProjectID {
		return nil, fmt.Errorf("key is for project %q but integration is connected to %q", metadata.ProjectID, storedProjectID)
	}

	client, err := gcpcommon.NewClientFromKeyJSON(ctx.HTTP, []byte(keyStr), metadata.ProjectID)
	if err != nil {
		return nil, err
	}

	crmURL := fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/projects/%s", metadata.ProjectID)
	if _, err := client.GetURL(context.Background(), crmURL); err != nil {
		return nil, fmt.Errorf("connection failed. Ensure the 'Cloud Resource Manager API' is enabled and the service account has 'Viewer' permissions: %w", err)
	}

	return nil, ctx.Secrets.Update(gcpcommon.SecretNameServiceAccountKey, keyStr)
}

func (s *SetupProvider) OnCapabilityUpdate(ctx core.CapabilityUpdateContext) (*core.SetupStep, error) {
	requested, ok := ctx.Changes[core.IntegrationCapabilityStateRequested]
	if !ok {
		return nil, errors.New("no requested capabilities")
	}
	ctx.Capabilities.Enable(requested...)
	return nil, nil
}

// --- step handlers ---

func (s *SetupProvider) onCapabilitySelectionSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	ctx.Capabilities.Request(ctx.Step.Capabilities...)
	ctx.Capabilities.Available(s.capabilityDiff(ctx.Step.Capabilities)...)

	return &core.SetupStep{
		Type:  core.SetupStepTypeInputs,
		Name:  SetupStepSelectConnectionMethod,
		Label: "How do you want to connect to Google Cloud?",
		Inputs: []configuration.Field{
			{
				Name:        PropertyConnectionMethod,
				Label:       "Connection method",
				Type:        configuration.FieldTypeSelect,
				Required:    true,
				Description: "Authenticate with a Service Account JSON key or Workload Identity Federation (keyless).",
				Default:     ConnectionMethodServiceAccountKey,
				TypeOptions: &configuration.TypeOptions{
					Select: &configuration.SelectTypeOptions{
						Options: []configuration.FieldOption{
							{Label: "Service Account Key", Value: ConnectionMethodServiceAccountKey},
							{Label: "Workload Identity Federation", Value: ConnectionMethodWIF},
						},
					},
				},
			},
		},
	}, nil
}

func (s *SetupProvider) onSelectConnectionMethodSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := ctx.Step.Inputs.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	method, ok := m[PropertyConnectionMethod].(string)
	if !ok || strings.TrimSpace(method) == "" {
		return nil, errors.New("connection method is required")
	}

	if method != ConnectionMethodServiceAccountKey && method != ConnectionMethodWIF {
		return nil, fmt.Errorf("unknown connection method: %s", method)
	}

	if err := ctx.Properties.Create(core.IntegrationPropertyDefinition{
		Name:        PropertyConnectionMethod,
		Label:       "Connection Method",
		Description: "Authentication method used to connect to Google Cloud",
		Type:        core.IntegrationPropertyTypeString,
		Value:       method,
		Editable:    false,
	}); err != nil {
		return nil, fmt.Errorf("error storing connection method: %w", err)
	}

	if method == ConnectionMethodWIF {
		wifInstructions, err := renderTemplate("wifProvider", wifProviderInstructionsTemplate, map[string]any{
			"IssuerURL": integrationOIDCIssuerURL(ctx),
		})
		if err != nil {
			return nil, err
		}
		return &core.SetupStep{
			Type:         core.SetupStepTypeInputs,
			Name:         SetupStepWIFProvider,
			Label:        "Enter Workload Identity Federation details",
			Instructions: wifInstructions,
			Inputs: []configuration.Field{
				{
					Name:        PropertyWIFProvider,
					Label:       "Pool Provider Resource Name",
					Type:        configuration.FieldTypeString,
					Required:    true,
					Description: "Full resource name of the OIDC provider.",
					Placeholder: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/superplane",
				},
				{
					Name:        PropertyProjectID,
					Label:       "Project ID",
					Type:        configuration.FieldTypeString,
					Required:    true,
					Description: "GCP project ID (e.g. my-project)",
					Placeholder: "e.g. my-project",
				},
			},
		}, nil
	}

	return &core.SetupStep{
		Type:         core.SetupStepTypeInputs,
		Name:         SetupStepServiceAccountKey,
		Label:        "Enter Service Account key",
		Instructions: string(saKeyInstructionsTemplate),
		Inputs: []configuration.Field{
			{
				Name:      gcpcommon.SecretNameServiceAccountKey,
				Label:     "Service Account Key (JSON)",
				Type:      configuration.FieldTypeString,
				Required:  true,
				Sensitive: true,
			},
		},
	}, nil
}

func (s *SetupProvider) onEnterServiceAccountKeySubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := ctx.Step.Inputs.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	keyStr, ok := m[gcpcommon.SecretNameServiceAccountKey].(string)
	if !ok {
		return nil, errors.New("invalid service account key")
	}
	keyStr = strings.TrimSpace(keyStr)
	if keyStr == "" {
		return nil, errors.New("service account key is required")
	}

	metadata, err := validateAndParseServiceAccountKey([]byte(keyStr))
	if err != nil {
		return nil, fmt.Errorf("invalid service account key: %w", err)
	}

	client, err := gcpcommon.NewClientFromKeyJSON(ctx.HTTP, []byte(keyStr), metadata.ProjectID)
	if err != nil {
		return nil, err
	}

	crmURL := fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/projects/%s", metadata.ProjectID)
	if _, err := client.GetURL(context.Background(), crmURL); err != nil {
		return nil, fmt.Errorf("connection failed. Ensure the 'Cloud Resource Manager API' is enabled and the service account has 'Viewer' permissions: %w", err)
	}

	if err := ctx.Secrets.Create(core.IntegrationSecretDefinition{
		Name:        gcpcommon.SecretNameServiceAccountKey,
		Label:       "Service Account Key",
		Description: "GCP service account JSON key",
		Value:       keyStr,
		Editable:    true,
	}); err != nil {
		return nil, fmt.Errorf("error storing service account key: %w", err)
	}

	if err := ctx.Properties.CreateMany([]core.IntegrationPropertyDefinition{
		{Name: PropertyProjectID, Label: "Project ID", Type: core.IntegrationPropertyTypeString, Value: metadata.ProjectID, Editable: false},
		{Name: PropertyClientEmail, Label: "Service Account", Type: core.IntegrationPropertyTypeString, Value: metadata.ClientEmail, Editable: false},
	}); err != nil {
		return nil, fmt.Errorf("error storing properties: %w", err)
	}

	ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)

	instructions, err := renderTemplate("setupCompleteSAK", setupCompleteSAKTemplate, map[string]any{
		"ProjectID":   metadata.ProjectID,
		"ClientEmail": metadata.ClientEmail,
	})
	if err != nil {
		return nil, err
	}

	return &core.SetupStep{
		Type:         core.SetupStepTypeDone,
		Name:         "done",
		Label:        "Setup complete",
		Instructions: instructions,
	}, nil
}

func (s *SetupProvider) onEnterWIFProviderSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := ctx.Step.Inputs.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	provider, ok := m[PropertyWIFProvider].(string)
	if !ok || strings.TrimSpace(provider) == "" {
		return nil, errors.New("pool provider resource name is required")
	}

	projectID, ok := m[PropertyProjectID].(string)
	if !ok || strings.TrimSpace(projectID) == "" {
		return nil, errors.New("project ID is required")
	}

	provider = strings.TrimSpace(provider)
	projectID = strings.TrimSpace(projectID)

	if err := ctx.Properties.CreateMany([]core.IntegrationPropertyDefinition{
		{Name: PropertyWIFProvider, Label: "WIF Pool Provider", Type: core.IntegrationPropertyTypeString, Value: provider, Editable: false},
		{Name: PropertyProjectID, Label: "Project ID", Type: core.IntegrationPropertyTypeString, Value: projectID, Editable: false},
	}); err != nil {
		return nil, fmt.Errorf("error storing WIF properties: %w", err)
	}

	principal := wifPrincipal(provider, ctx.IntegrationID.String())
	instructions, err := renderTemplate("wifServiceAccount", wifServiceAccountInstructionsTemplate, map[string]any{
		"Principal":     principal,
		"IntegrationID": ctx.IntegrationID.String(),
	})
	if err != nil {
		return nil, err
	}

	return &core.SetupStep{
		Type:         core.SetupStepTypeInputs,
		Name:         SetupStepWIFServiceAccount,
		Label:        "Enter service account email",
		Instructions: instructions,
		Inputs: []configuration.Field{
			{
				Name:        PropertyServiceAccountEmail,
				Label:       "Service Account Email",
				Type:        configuration.FieldTypeString,
				Required:    true,
				Description: "Email of the service account SuperPlane will impersonate.",
				Placeholder: "e.g. superplane@my-project.iam.gserviceaccount.com",
			},
		},
	}, nil
}

func (s *SetupProvider) onEnterWIFServiceAccountSubmit(ctx core.SetupStepContext) (*core.SetupStep, error) {
	m, ok := ctx.Step.Inputs.(map[string]any)
	if !ok {
		return nil, errors.New("invalid input")
	}

	saEmail, ok := m[PropertyServiceAccountEmail].(string)
	if !ok || strings.TrimSpace(saEmail) == "" {
		return nil, errors.New("service account email is required")
	}
	saEmail = strings.TrimSpace(saEmail)

	if err := ctx.Properties.Create(core.IntegrationPropertyDefinition{
		Name:        PropertyServiceAccountEmail,
		Label:       "Service Account",
		Description: "GCP service account SuperPlane impersonates via WIF",
		Type:        core.IntegrationPropertyTypeString,
		Value:       saEmail,
		Editable:    false,
	}); err != nil {
		return nil, fmt.Errorf("error storing service account email: %w", err)
	}

	ctx.Capabilities.Enable(ctx.Capabilities.Requested()...)

	projectID, _ := ctx.Properties.GetString(PropertyProjectID)
	instructions, err := renderTemplate("setupCompleteWIF", setupCompleteWIFTemplate, map[string]any{
		"ProjectID":           projectID,
		"ServiceAccountEmail": saEmail,
		"IntegrationID":       ctx.IntegrationID.String(),
	})
	if err != nil {
		return nil, err
	}

	return &core.SetupStep{
		Type:         core.SetupStepTypeDone,
		Name:         "done",
		Label:        "Setup complete",
		Instructions: instructions,
	}, nil
}

// wifPrincipal derives the full WIF principal from the provider resource name and integration ID.
// Provider format: //iam.googleapis.com/projects/N/locations/global/workloadIdentityPools/P/providers/R
// Principal format: principal://iam.googleapis.com/projects/N/locations/global/workloadIdentityPools/P/subject/app-installation:ID
func wifPrincipal(provider, integrationID string) string {
	path := strings.TrimPrefix(provider, "//iam.googleapis.com/")
	if idx := strings.LastIndex(path, "/providers/"); idx >= 0 {
		path = path[:idx]
	}
	return fmt.Sprintf("principal://iam.googleapis.com/%s/subject/app-installation:%s", path, integrationID)
}

// integrationOIDCIssuerURL is the origin served in /.well-known/openid-configuration (see public.Server.handleOIDCConfiguration).
func integrationOIDCIssuerURL(ctx core.SetupStepContext) string {
	base := strings.TrimSpace(ctx.WebhooksBaseURL)
	if base == "" {
		base = strings.TrimSpace(ctx.BaseURL)
	}
	return oidc.CanonicalIssuerURL(base)
}

func renderTemplate(name string, tplBytes []byte, data map[string]any) (string, error) {
	tmpl, err := template.New(name).Parse(string(tplBytes))
	if err != nil {
		return "", fmt.Errorf("error parsing template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template %s: %w", name, err)
	}
	return buf.String(), nil
}
