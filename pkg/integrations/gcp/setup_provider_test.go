package gcp

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/logger"
)

var testIntegrationID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func newSetupCtx(intCtx *contexts.IntegrationContext, httpCtx *contexts.HTTPContext, caps *contexts.CapabilityContext) core.SetupStepContext {
	return core.SetupStepContext{
		IntegrationID: testIntegrationID,
		Logger:        logger.DiscardLogger(),
		HTTP:          httpCtx,
		Properties:    contexts.NewIntegrationPropertyStorage(intCtx),
		Secrets:       intCtx.Secrets(),
		Capabilities:  caps,
	}
}

// Test_GCP_SetupProvider_OnCapabilityUpdate -------------------------------------------------------

func Test_GCP_SetupProvider_OnCapabilityUpdate(t *testing.T) {
	s := &SetupProvider{}

	t.Run("returns error when no requested entry", func(t *testing.T) {
		_, err := s.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger:       logger.DiscardLogger(),
			Changes:      map[core.IntegrationCapabilityState][]string{},
			Capabilities: &contexts.CapabilityContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no requested capabilities")
	})

	t.Run("enables requested capabilities", func(t *testing.T) {
		caps := &contexts.CapabilityContext{}
		_, err := s.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: logger.DiscardLogger(),
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"gcp.createVM", "gcp.cloudbuild.createBuild"},
			},
			Capabilities: caps,
		})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"gcp.createVM", "gcp.cloudbuild.createBuild"}, caps.EnabledCapabilities)
	})
}

// Test_GCP_SetupProvider_OnStepSubmit -------------------------------------------------------------

func Test_GCP_SetupProvider_OnStepSubmit(t *testing.T) {
	s := &SetupProvider{}

	t.Run("unknown step returns error", func(t *testing.T) {
		ctx := newSetupCtx(&contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: "nope"}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step")
	})

	// --- capabilitySelection ---

	t.Run("capabilitySelection moves capabilities to requested/available and returns WIF provider step", func(t *testing.T) {
		caps := &contexts.CapabilityContext{}
		ctx := newSetupCtx(&contexts.IntegrationContext{}, &contexts.HTTPContext{}, caps)
		ctx.WebhooksBaseURL = "https://superplane.example"
		ctx.Step = core.StepInfo{
			Name:         SetupStepCapabilitySelection,
			Capabilities: []string{"gcp.createVM"},
		}
		next, err := s.OnStepSubmit(ctx)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepWIFProvider, next.Name)
		assert.Equal(t, core.SetupStepTypeInputs, next.Type)
		assert.Contains(t, next.Instructions, "https://superplane.example")
		assert.Equal(t, []string{"gcp.createVM"}, caps.RequestedCapabilties)
		assert.NotEmpty(t, caps.AvailableCapabilities)
	})

	// --- enterWIFProvider ---

	t.Run("enterWIFProvider with invalid input returns error", func(t *testing.T) {
		ctx := newSetupCtx(&contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFProvider, Inputs: 123}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")
	})

	t.Run("enterWIFProvider with empty provider returns error", func(t *testing.T) {
		ctx := newSetupCtx(&contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFProvider, Inputs: map[string]any{PropertyWIFProvider: "", PropertyProjectID: "my-project"}}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pool provider resource name is required")
	})

	t.Run("enterWIFProvider with empty project ID returns error", func(t *testing.T) {
		ctx := newSetupCtx(&contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFProvider, Inputs: map[string]any{
			PropertyWIFProvider: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers/sp",
			PropertyProjectID:   "",
		}}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project ID is required")
	})

	t.Run("enterWIFProvider success stores properties and returns enterWIFServiceAccount", func(t *testing.T) {
		provider := "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/superplane"
		intCtx := &contexts.IntegrationContext{}
		ctx := newSetupCtx(intCtx, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFProvider, Inputs: map[string]any{
			PropertyWIFProvider: provider,
			PropertyProjectID:   "my-project",
		}}

		next, err := s.OnStepSubmit(ctx)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepWIFServiceAccount, next.Name)
		assert.Equal(t, core.SetupStepTypeInputs, next.Type)
		assert.Contains(t, next.Instructions, testIntegrationID.String())
		assert.Contains(t, next.Instructions, "app-installation:"+testIntegrationID.String())

		storedProvider, _ := ctx.Properties.GetString(PropertyWIFProvider)
		assert.Equal(t, provider, storedProvider)
		storedProject, _ := ctx.Properties.GetString(PropertyProjectID)
		assert.Equal(t, "my-project", storedProject)
	})

	t.Run("enterWIFProvider normalizes IAM REST URL to canonical resource name", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		ctx := newSetupCtx(intCtx, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFProvider, Inputs: map[string]any{
			PropertyWIFProvider: "https://iam.googleapis.com/v1/projects/999/locations/global/workloadIdentityPools/w-pool/providers/w-prov",
			PropertyProjectID:   "my-project",
		}}

		next, err := s.OnStepSubmit(ctx)
		require.NoError(t, err)
		require.NotNil(t, next)
		storedProvider, _ := ctx.Properties.GetString(PropertyWIFProvider)
		assert.Equal(t, "//iam.googleapis.com/projects/999/locations/global/workloadIdentityPools/w-pool/providers/w-prov", storedProvider)
		assert.Equal(t, SetupStepWIFServiceAccount, next.Name)
	})

	// --- enterWIFServiceAccount ---

	t.Run("enterWIFServiceAccount with invalid input returns error", func(t *testing.T) {
		ctx := newSetupCtx(&contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFServiceAccount, Inputs: "bad"}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")
	})

	t.Run("enterWIFServiceAccount with empty email returns error", func(t *testing.T) {
		ctx := newSetupCtx(&contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFServiceAccount, Inputs: map[string]any{PropertyServiceAccountEmail: ""}}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service account email is required")
	})

	t.Run("enterWIFServiceAccount success stores email, enables capabilities, returns done", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		caps := &contexts.CapabilityContext{RequestedCapabilties: []string{"gcp.cloudbuild.onBuildComplete"}}
		ctx := newSetupCtx(intCtx, &contexts.HTTPContext{}, caps)
		require.NoError(t, ctx.Properties.Create(core.IntegrationPropertyDefinition{Name: PropertyProjectID, Value: "my-project"}))
		ctx.Step = core.StepInfo{Name: SetupStepWIFServiceAccount, Inputs: map[string]any{PropertyServiceAccountEmail: "sp@my-project.iam.gserviceaccount.com"}}

		next, err := s.OnStepSubmit(ctx)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, core.SetupStepTypeDone, next.Type)
		assert.Equal(t, "done", next.Name)
		assert.Contains(t, next.Instructions, "my-project")
		assert.Contains(t, next.Instructions, "sp@my-project.iam.gserviceaccount.com")

		storedEmail, _ := ctx.Properties.GetString(PropertyServiceAccountEmail)
		assert.Equal(t, "sp@my-project.iam.gserviceaccount.com", storedEmail)
		assert.Equal(t, []string{"gcp.cloudbuild.onBuildComplete"}, caps.EnabledCapabilities)
	})
}

// Test_GCP_SetupProvider_OnStepRevert -------------------------------------------------------------

func Test_GCP_SetupProvider_OnStepRevert(t *testing.T) {
	s := &SetupProvider{}

	t.Run("unknown step returns error", func(t *testing.T) {
		ctx := newSetupCtx(&contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: "nope"}
		err := s.OnStepRevert(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step")
	})

	t.Run("capabilitySelection clears capabilities", func(t *testing.T) {
		caps := &contexts.CapabilityContext{
			RequestedCapabilties:  []string{"gcp.createVM"},
			AvailableCapabilities: []string{"gcp.cloudbuild.createBuild"},
		}
		ctx := newSetupCtx(&contexts.IntegrationContext{}, &contexts.HTTPContext{}, caps)
		ctx.Step = core.StepInfo{Name: SetupStepCapabilitySelection}
		require.NoError(t, s.OnStepRevert(ctx))
		assert.Empty(t, caps.RequestedCapabilties)
		assert.Empty(t, caps.AvailableCapabilities)
	})

	t.Run("enterWIFProvider deletes provider and projectId properties", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		ctx := newSetupCtx(intCtx, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		require.NoError(t, ctx.Properties.Create(core.IntegrationPropertyDefinition{Name: PropertyWIFProvider, Value: "//iam.googleapis.com/projects/123/..."}))
		require.NoError(t, ctx.Properties.Create(core.IntegrationPropertyDefinition{Name: PropertyProjectID, Value: "my-project"}))

		ctx.Step = core.StepInfo{Name: SetupStepWIFProvider}
		require.NoError(t, s.OnStepRevert(ctx))

		_, err := ctx.Properties.GetString(PropertyWIFProvider)
		require.Error(t, err)
		_, err = ctx.Properties.GetString(PropertyProjectID)
		require.Error(t, err)
	})

	t.Run("enterWIFServiceAccount deletes service account email", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		ctx := newSetupCtx(intCtx, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		require.NoError(t, ctx.Properties.Create(core.IntegrationPropertyDefinition{Name: PropertyServiceAccountEmail, Value: "sa@proj.iam.gserviceaccount.com"}))

		ctx.Step = core.StepInfo{Name: SetupStepWIFServiceAccount}
		require.NoError(t, s.OnStepRevert(ctx))

		_, err := ctx.Properties.GetString(PropertyServiceAccountEmail)
		require.Error(t, err)
	})
}

// Test_GCP_SetupProvider_OnSecretUpdate -----------------------------------------------------------

func Test_GCP_SetupProvider_OnSecretUpdate(t *testing.T) {
	s := &SetupProvider{}

	t.Run("any secret update is rejected (WIF-only setup has no editable secrets)", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger.DiscardLogger(),
			SecretName: "anything",
			Value:      "x",
			HTTP:       &contexts.HTTPContext{},
			Properties: contexts.NewIntegrationPropertyStorage(intCtx),
			Secrets:    intCtx.Secrets(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown secret")
	})
}

// Test_GCP_SetupProvider_wifPrincipal -------------------------------------------------------------

func Test_GCP_SetupProvider_wifPrincipal(t *testing.T) {
	provider := "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/superplane"
	id := "abc-def"
	principal := wifPrincipal(provider, id)
	assert.Equal(t,
		"principal://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/subject/app-installation:abc-def",
		principal,
	)
}

// Test_GCP_SetupProvider_CapabilityGroups ---------------------------------------------------------

func Test_GCP_SetupProvider_CapabilityGroups(t *testing.T) {
	s := &SetupProvider{}
	groups := s.CapabilityGroups()
	require.NotEmpty(t, groups)

	var allNames []string
	for _, g := range groups {
		for _, c := range g.Capabilities {
			allNames = append(allNames, c.Name)
			assert.NotEmpty(t, c.Label, "capability %s has no label", c.Name)
		}
	}

	assert.Contains(t, allNames, "gcp.createVM")
	assert.Contains(t, allNames, "gcp.compute.onVMInstance")
	assert.Contains(t, allNames, "gcp.cloudbuild.createBuild")
	assert.Contains(t, allNames, "gcp.cloudbuild.onBuildComplete")
	assert.Contains(t, allNames, "gcp.artifactregistry.getArtifact")
	assert.Contains(t, allNames, "gcp.artifactregistry.onArtifactPush")
	assert.Contains(t, allNames, "gcp.pubsub.publishMessage")
	assert.Contains(t, allNames, "gcp.pubsub.onMessage")
	assert.Contains(t, allNames, "gcp.cloudfunctions.invokeFunction")
	assert.Contains(t, allNames, "gcp.clouddns.createRecord")
}
