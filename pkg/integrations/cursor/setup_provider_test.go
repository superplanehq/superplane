package cursor

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/logger"
)

func Test__Cursor__SetupProvider__OnCapabilityUpdate(t *testing.T) {
	s := &SetupProvider{}
	log := logger.DiscardLogger()

	t.Run("returns error when no requested capabilities entry", func(t *testing.T) {
		_, err := s.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger:       log,
			Changes:      map[core.IntegrationCapabilityState][]string{},
			Capabilities: &contexts.CapabilityContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no requested capabilities")
	})

	t.Run("returns enterAdminKey when admin capability requested but admin secret missing", func(t *testing.T) {
		cap := &contexts.CapabilityContext{}
		step, err := s.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: log,
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"cursor.getDailyUsageData"},
			},
			Capabilities: cap,
			Secrets:      (&contexts.IntegrationContext{CurrentSecrets: map[string]core.IntegrationSecret{}}).Secrets(),
			HTTP:         &contexts.HTTPContext{},
		})
		require.NoError(t, err)
		require.NotNil(t, step)
		assert.Equal(t, SetupStepEnterAdminKey, step.Name)
		assert.Equal(t, core.SetupStepTypeInputs, step.Type)
		require.Len(t, step.Inputs, 1)
		assert.Equal(t, SecretAdminKey, step.Inputs[0].Name)
	})

	t.Run("returns enterAdminKey when launch key exists but admin missing", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		intCtx := &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				SecretLaunchAgentKey: {Name: SecretLaunchAgentKey, Value: []byte("existing-launch")},
			},
		}

		cap := &contexts.CapabilityContext{}
		step, err := s.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: log,
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"cursor.getDailyUsageData"},
			},
			Capabilities: cap,
			Secrets:      intCtx.Secrets(),
			HTTP:         httpCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, step)
		require.Len(t, step.Inputs, 1)
		assert.Equal(t, SecretAdminKey, step.Inputs[0].Name)
	})

	t.Run("enables when admin key already present and API succeeds", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
			},
		}
		intCtx := &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				SecretAdminKey: {Name: SecretAdminKey, Value: []byte("valid-admin")},
			},
		}
		cap := &contexts.CapabilityContext{}
		step, err := s.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: log,
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"cursor.getDailyUsageData"},
			},
			Capabilities: cap,
			Secrets:      intCtx.Secrets(),
			HTTP:         httpCtx,
		})
		require.NoError(t, err)
		assert.Nil(t, step)
		assert.Contains(t, cap.EnabledCapabilities, "cursor.getDailyUsageData")
	})
}

func Test__Cursor__SetupProvider__OnSecretUpdate(t *testing.T) {
	s := &SetupProvider{}
	log := logger.DiscardLogger()

	t.Run("unknown secret", func(t *testing.T) {
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:       log,
			SecretName:   "other",
			Value:        "x",
			HTTP:         &contexts.HTTPContext{},
			Secrets:      (&contexts.IntegrationContext{CurrentSecrets: map[string]core.IntegrationSecret{}}).Secrets(),
			Capabilities: &contexts.CapabilityContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown secret")
	})

	t.Run("value is required when trimmed empty", func(t *testing.T) {
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     log,
			SecretName: SecretLaunchAgentKey,
			Value:      "   ",
			HTTP:       &contexts.HTTPContext{},
			Secrets: (&contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					SecretAdminKey: {Name: SecretAdminKey, Value: []byte("adm")},
				},
			}).Secrets(),
			Capabilities: &contexts.CapabilityContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "value is required")
	})

	t.Run("success updates launch secret", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"agents":[]}`)),
				},
			},
		}
		intCtx := &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				SecretLaunchAgentKey: {Name: SecretLaunchAgentKey, Value: []byte("old")},
			},
		}
		secrets := intCtx.Secrets()
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:       log,
			SecretName:   SecretLaunchAgentKey,
			Value:        "new-launch",
			HTTP:         httpCtx,
			Secrets:      secrets,
			Capabilities: &contexts.CapabilityContext{},
		})
		require.NoError(t, err)
		v, getErr := secrets.Get(SecretLaunchAgentKey)
		require.NoError(t, getErr)
		assert.Equal(t, "new-launch", v)
	})

	t.Run("updating launch key does not verify admin key", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"agents":[]}`)),
				},
			},
		}

		intCtx := &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				SecretAdminKey:       {Name: SecretAdminKey, Value: []byte("stale-admin")},
				SecretLaunchAgentKey: {Name: SecretLaunchAgentKey, Value: []byte("old-launch")},
			},
		}
		secrets := intCtx.Secrets()

		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:       log,
			SecretName:   SecretLaunchAgentKey,
			Value:        "new-launch",
			HTTP:         httpCtx,
			Secrets:      secrets,
			Capabilities: &contexts.CapabilityContext{},
		})
		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.cursor.com/v0/agents?limit=1", httpCtx.Requests[0].URL.String())
	})
}

func Test__Cursor__SetupProvider__OnStepSubmit(t *testing.T) {
	s := &SetupProvider{}
	log := logger.DiscardLogger()

	t.Run("unknown step", func(t *testing.T) {
		_, err := s.OnStepSubmit(core.SetupStepContext{
			Step:   core.StepInfo{Name: "nope"},
			Logger: log,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step")
	})

	t.Run("capability selection requires at least one", func(t *testing.T) {
		_, err := s.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepCapabilitySelection, Capabilities: []string{}},
			Logger:       log,
			Capabilities: &contexts.CapabilityContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one capability")
	})

	t.Run("capability selection skips key steps when secrets already satisfy selection", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"agents":[]}`)),
				},
			},
		}
		intCtx := &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				SecretLaunchAgentKey: {Name: SecretLaunchAgentKey, Value: []byte("tok")},
			},
		}
		cap := &contexts.CapabilityContext{}
		next, err := s.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{
				Name:         SetupStepCapabilitySelection,
				Capabilities: []string{"cursor.launchAgent"},
			},
			Logger:       log,
			Capabilities: cap,
			Secrets:      intCtx.Secrets(),
			HTTP:         httpCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepDone, next.Name)
		assert.Equal(t, core.SetupStepTypeDone, next.Type)
		assert.Contains(t, cap.EnabledCapabilities, "cursor.launchAgent")
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.cursor.com/v0/agents?limit=1", httpCtx.Requests[0].URL.String())
	})

	t.Run("capability selection with both key types asks for launch key first", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{CurrentSecrets: map[string]core.IntegrationSecret{}}
		cap := &contexts.CapabilityContext{}
		next, err := s.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{
				Name:         SetupStepCapabilitySelection,
				Capabilities: []string{"cursor.launchAgent", "cursor.getDailyUsageData"},
			},
			Logger:       log,
			Capabilities: cap,
			Secrets:      intCtx.Secrets(),
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepEnterLaunchKey, next.Name)
		assert.Equal(t, core.SetupStepTypeInputs, next.Type)
		require.Len(t, next.Inputs, 1)
		assert.Equal(t, SecretLaunchAgentKey, next.Inputs[0].Name)
	})

	t.Run("enterLaunchKey validation", func(t *testing.T) {
		cap := &contexts.CapabilityContext{}
		cap.Request("cursor.launchAgent")

		_, err := s.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepEnterLaunchKey, Inputs: "not-a-map"},
			Logger:       log,
			Capabilities: cap,
			Secrets:      (&contexts.IntegrationContext{CurrentSecrets: map[string]core.IntegrationSecret{}}).Secrets(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")
	})

	t.Run("enterLaunchKey success enables capabilities when only launch is needed", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"agents":[]}`)),
				},
			},
		}
		cap := &contexts.CapabilityContext{}
		cap.Request("cursor.launchAgent")

		intCtx := &contexts.IntegrationContext{CurrentSecrets: map[string]core.IntegrationSecret{}}
		next, err := s.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{
				Name:   SetupStepEnterLaunchKey,
				Inputs: map[string]any{SecretLaunchAgentKey: "tok"},
			},
			Logger:       log,
			Capabilities: cap,
			Secrets:      intCtx.Secrets(),
			HTTP:         httpCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, core.SetupStepTypeDone, next.Type)
		assert.Contains(t, cap.EnabledCapabilities, "cursor.launchAgent")
		v, getErr := intCtx.Secrets().Get(SecretLaunchAgentKey)
		require.NoError(t, getErr)
		assert.Equal(t, "tok", v)
	})

	t.Run("enterLaunchKey continues to admin step when admin is also needed", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"agents":[]}`)),
				},
			},
		}
		cap := &contexts.CapabilityContext{}
		cap.Request("cursor.launchAgent", "cursor.getDailyUsageData")
		intCtx := &contexts.IntegrationContext{CurrentSecrets: map[string]core.IntegrationSecret{}}

		next, err := s.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{
				Name:   SetupStepEnterLaunchKey,
				Inputs: map[string]any{SecretLaunchAgentKey: "tok"},
			},
			Logger:       log,
			Capabilities: cap,
			Secrets:      intCtx.Secrets(),
			HTTP:         httpCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepEnterAdminKey, next.Name)
		require.Len(t, next.Inputs, 1)
		assert.Equal(t, SecretAdminKey, next.Inputs[0].Name)
	})

	t.Run("enterLaunchKey verifies only admin when admin secret already exists", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"agents":[]}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
			},
		}
		cap := &contexts.CapabilityContext{}
		cap.Request("cursor.launchAgent", "cursor.getDailyUsageData")
		intCtx := &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				SecretAdminKey: {Name: SecretAdminKey, Value: []byte("existing-admin")},
			},
		}

		next, err := s.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{
				Name:   SetupStepEnterLaunchKey,
				Inputs: map[string]any{SecretLaunchAgentKey: "tok"},
			},
			Logger:       log,
			Capabilities: cap,
			Secrets:      intCtx.Secrets(),
			HTTP:         httpCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, core.SetupStepTypeDone, next.Type)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, "https://api.cursor.com/v0/agents?limit=1", httpCtx.Requests[0].URL.String())
		assert.Equal(t, "https://api.cursor.com/teams/daily-usage-data", httpCtx.Requests[1].URL.String())
	})

	t.Run("enterAdminKey success enables capabilities", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
			},
		}
		cap := &contexts.CapabilityContext{}
		cap.Request("cursor.getDailyUsageData")
		intCtx := &contexts.IntegrationContext{CurrentSecrets: map[string]core.IntegrationSecret{}}

		next, err := s.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{
				Name:   SetupStepEnterAdminKey,
				Inputs: map[string]any{SecretAdminKey: "adm"},
			},
			Logger:       log,
			Capabilities: cap,
			Secrets:      intCtx.Secrets(),
			HTTP:         httpCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, core.SetupStepTypeDone, next.Type)
		assert.Contains(t, cap.EnabledCapabilities, "cursor.getDailyUsageData")
		v, getErr := intCtx.Secrets().Get(SecretAdminKey)
		require.NoError(t, getErr)
		assert.Equal(t, "adm", v)
	})

	t.Run("enterAdminKey verifies stored launch key when agent capabilities are requested", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"agents":[]}`)),
				},
			},
		}
		cap := &contexts.CapabilityContext{}
		cap.Request("cursor.launchAgent", "cursor.getDailyUsageData")
		intCtx := &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				SecretLaunchAgentKey: {Name: SecretLaunchAgentKey, Value: []byte("existing-launch")},
			},
		}

		next, err := s.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{
				Name:   SetupStepEnterAdminKey,
				Inputs: map[string]any{SecretAdminKey: "adm"},
			},
			Logger:       log,
			Capabilities: cap,
			Secrets:      intCtx.Secrets(),
			HTTP:         httpCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, core.SetupStepTypeDone, next.Type)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, "https://api.cursor.com/teams/daily-usage-data", httpCtx.Requests[0].URL.String())
		assert.Equal(t, "https://api.cursor.com/v0/agents?limit=1", httpCtx.Requests[1].URL.String())
	})
}

func Test__Cursor__SetupProvider__OnStepRevert(t *testing.T) {
	s := &SetupProvider{}
	log := logger.DiscardLogger()

	t.Run("unknown step", func(t *testing.T) {
		err := s.OnStepRevert(core.SetupStepContext{
			Step:   core.StepInfo{Name: "x"},
			Logger: log,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step")
	})

	t.Run("key step revert preserves stored secrets", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				SecretLaunchAgentKey: {Name: SecretLaunchAgentKey, Value: []byte("sek")},
				SecretAdminKey:       {Name: SecretAdminKey, Value: []byte("adm")},
			},
		}
		require.NoError(t, s.OnStepRevert(core.SetupStepContext{
			Step:    core.StepInfo{Name: SetupStepEnterLaunchKey},
			Logger:  log,
			Secrets: intCtx.Secrets(),
		}))
		launch, err := intCtx.Secrets().Get(SecretLaunchAgentKey)
		require.NoError(t, err)
		assert.Equal(t, "sek", launch)
		admin, err := intCtx.Secrets().Get(SecretAdminKey)
		require.NoError(t, err)
		assert.Equal(t, "adm", admin)
	})
}
