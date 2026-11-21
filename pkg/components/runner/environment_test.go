package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestPromptToolReceivesSelectedSecretWithoutPromptInterpolation(t *testing.T) {
	const token = "z7!"
	secrets := &contexts.SecretsContext{SecretKeys: map[string]map[string][]byte{
		"figma-api-key": {"FIGMA_API_KEY": []byte(token), "FIGMA_ACCESS": []byte(token)},
	}}
	resolved, err := ResolveEnvironment(secrets, []EnvironmentFromEntry{{Source: EnvironmentFromSourceSecret, Secret: configuration.SecretRef{Secret: "figma-api-key"}}}, nil)
	require.NoError(t, err)
	values := environmentValues(resolved.Variables)
	assert.JSONEq(t, `["FIGMA_ACCESS","FIGMA_API_KEY"]`, values[EnvSecretNames])
	prompt := "Read designs with FIGMA_API_KEY."
	commands, files := BuildAgentTask(AgentTaskInput{
		Steps:         []AgentStep{{Name: "Design", Type: AgentStepPrompt, Prompt: &prompt, Secrets: []string{"figma-api-key"}}},
		RunScriptName: "agent.js",
		// The fixture launches the same child process boundary used by agent tools.
		RunScript: `const {spawnSync} = require("child_process");
const tool = spawnSync(process.execPath, ["-e", 'process.exit(process.env.FIGMA_API_KEY === "z7!" ? 0 : 1)'], {env: process.env});
process.exit(tool.status);`,
		PromptCommand: func(name, model string) string { return `node "$SUPERPLANE_TASK_DIR/agent.js"` },
	})
	task := t.TempDir()
	for _, file := range files {
		assert.NotContains(t, file.Content, values[EnvSecretNames])
		if file.Path != "agent.js" {
			assert.NotContains(t, file.Content, token)
		}
		target := filepath.Join(task, file.Path)
		require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
		require.NoError(t, os.WriteFile(target, []byte(file.Content), 0o644))
	}
	command := commands[len(commands)-1]
	assert.NotContains(t, command.Command, token)
	cmd := exec.Command("bash", "-c", command.Command)
	cmd.Dir = task
	cmd.Env = append(os.Environ(), "SUPERPLANE_TASK_DIR="+task, "SUPERPLANE_RESULT_FILE="+filepath.Join(task, "result.json"))
	for _, variable := range resolved.Variables {
		cmd.Env = append(cmd.Env, variable.Name+"="+variable.Value)
	}
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

func TestAgentStepsValidateSecretsForBothTypes(t *testing.T) {
	for _, kind := range []string{AgentStepPrompt, AgentStepBash} {
		for _, test := range []struct {
			secrets []string
			failure string
		}{
			{[]string{"figma-api-key"}, ""},
			{[]string{" "}, "secrets[0] is required"},
			{[]string{"figma-api-key", " figma-api-key "}, "duplicate secret"},
		} {
			err := ValidateAgentSteps([]AgentStep{{Name: "Design", Type: kind, Prompt: strPtr("Read designs"), Command: strPtr("true"), Secrets: test.secrets}})
			if test.failure == "" {
				require.NoError(t, err)
				continue
			}
			require.ErrorContains(t, err, test.failure)
		}
	}
}

func TestSelectedSecretResolutionFailsWithoutDisclosingValues(t *testing.T) {
	const token = "z7!"
	secrets := &contexts.SecretsContext{SecretKeys: map[string]map[string][]byte{
		"invalid":  {"BAD-NAME": []byte(token)},
		"reserved": {EnvSecretNames: []byte(token)},
		"first":    {"FIGMA_ACCESS": []byte(token)},
		"second":   {"FIGMA_ACCESS": []byte(token)},
	}}
	for _, test := range []struct {
		names   []string
		failure string
	}{
		{[]string{"missing"}, "secret keys not found"},
		{[]string{"invalid"}, "invalid environment variable name"},
		{[]string{"reserved"}, "reserved environment variable name"},
		{[]string{"first", "second"}, "duplicate environment variable name"},
	} {
		var sources []EnvironmentFromEntry
		for _, name := range test.names {
			sources = append(sources, EnvironmentFromEntry{Source: EnvironmentFromSourceSecret, Secret: configuration.SecretRef{Secret: name}})
		}
		resolved, err := ResolveEnvironment(secrets, sources, nil)
		require.ErrorContains(t, err, test.failure)
		assert.NotContains(t, err.Error(), token)
		assert.Empty(t, resolved.Variables)
	}
	_, err := ResolveEnvironment(nil, nil, []EnvironmentVariable{{Name: EnvSecretNames, ValueSource: EnvironmentValueSourceLiteral, Value: strPtr("[]")}})
	require.ErrorContains(t, err, "reserved environment variable name")
}

func TestResolveExecutionEnvironmentUsesSavedReviewFallback(t *testing.T) {
	for _, cached := range []bool{false, true} {
		for _, fallback := range []string{"fallback-user", ""} {
			t.Run(fmt.Sprintf("cached=%t/fallback=%s", cached, fallback), func(t *testing.T) {
				const name = "SUPERPLANE_REVIEW_REQUEST_FALLBACK"
				environment := []EnvironmentVariable{{Name: name, ValueSource: EnvironmentValueSourceLiteral, Value: strPtr("old-value")}}
				prepared := ResolvedEnvironment{Variables: []TaskEnvironmentVariable{{Name: name, Value: "old-value"}}, Usage: "Keep usage"}
				ctx := core.ExecutionContext{Configuration: map[string]any{"fallbackReviewer": fallback}}
				if cached {
					ctx.ResolvedEnvironment = &prepared
				}
				resolved, err := ResolveExecutionEnvironment(ctx, nil, environment)
				require.NoError(t, err)
				require.Equal(t, fallback, environmentValues(resolved.Variables)[name])
				count := 0
				for _, variable := range resolved.Variables {
					if variable.Name == name {
						count++
					}
				}
				require.Equal(t, 1, count, "runner tasks reject duplicate environment names")
				require.Equal(t, "old-value", prepared.Variables[0].Value)
				if cached {
					require.Equal(t, prepared.Usage, resolved.Usage)
				}
			})
		}
	}
}

func Test__ValidateEnvironmentFrom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		entries     []EnvironmentFromEntry
		errContains string
	}{
		{
			name: "valid integration",
			entries: []EnvironmentFromEntry{
				{
					Source:      EnvironmentFromSourceIntegration,
					Integration: configuration.IntegrationRef{Name: "my-github-integration"},
				},
			},
		},
		{
			name: "valid secret",
			entries: []EnvironmentFromEntry{
				{
					Source: EnvironmentFromSourceSecret,
					Secret: configuration.SecretRef{Secret: "some-other-secret"},
				},
			},
		},
		{
			name: "missing source",
			entries: []EnvironmentFromEntry{
				{Integration: configuration.IntegrationRef{Name: "my-github-integration"}},
			},
			errContains: "source is required",
		},
		{
			name: "missing integration ref",
			entries: []EnvironmentFromEntry{
				{Source: EnvironmentFromSourceIntegration},
			},
			errContains: "integration is required",
		},
		{
			name: "missing secret name",
			entries: []EnvironmentFromEntry{
				{Source: EnvironmentFromSourceSecret},
			},
			errContains: "secret is required",
		},
		{
			name: "duplicate integration",
			entries: []EnvironmentFromEntry{
				{
					Source:      EnvironmentFromSourceIntegration,
					Integration: configuration.IntegrationRef{Name: "my-github-integration"},
				},
				{
					Source:      EnvironmentFromSourceIntegration,
					Integration: configuration.IntegrationRef{Name: "my-github-integration"},
				},
			},
			errContains: "duplicate environmentFrom integration",
		},
		{
			name: "duplicate secret",
			entries: []EnvironmentFromEntry{
				{
					Source: EnvironmentFromSourceSecret,
					Secret: configuration.SecretRef{Secret: "some-other-secret"},
				},
				{
					Source: EnvironmentFromSourceSecret,
					Secret: configuration.SecretRef{Secret: "some-other-secret"},
				},
			},
			errContains: "duplicate environmentFrom secret",
		},
		{
			name: "invalid source",
			entries: []EnvironmentFromEntry{
				{
					Source:      "organization",
					Integration: configuration.IntegrationRef{Name: "my-github-integration"},
				},
			},
			errContains: "invalid environmentFrom[0].source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateEnvironmentFrom(tt.entries)
			if tt.errContains == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.errContains)
		})
	}
}

func Test__DecodeEnvironmentFromObjectRefs(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"environmentFrom": []any{
			map[string]any{
				"source": EnvironmentFromSourceIntegration,
				"integration": map[string]any{
					"name": "my-github-integration",
				},
			},
			map[string]any{
				"source": EnvironmentFromSourceSecret,
				"secret": map[string]any{
					"secret": "some-other-secret",
				},
			},
		},
	}

	var spec Spec
	dec, err := NewSpecDecoder(&spec)
	require.NoError(t, err)
	require.NoError(t, dec.Decode(raw))

	require.Len(t, spec.EnvironmentFrom, 2)
	assert.Equal(t, configuration.IntegrationRef{Name: "my-github-integration"}, spec.EnvironmentFrom[0].Integration)
	assert.Equal(t, configuration.SecretRef{Secret: "some-other-secret"}, spec.EnvironmentFrom[1].Secret)
}

func Test__ResolveEnvironment(t *testing.T) {
	t.Parallel()

	secrets := &contexts.SecretsContext{
		Values: map[string][]byte{
			"api/token": []byte("literal-secret"),
		},
		IntegrationKeys: map[string]map[string][]byte{
			"my-github-integration": {
				"GITHUB_TOKEN": []byte("gh-token"),
			},
		},
		SecretKeys: map[string]map[string][]byte{
			"some-other-secret": {
				"API_KEY": []byte("secret-key-value"),
			},
		},
	}

	resolved, err := ResolveEnvironment(
		secrets,
		[]EnvironmentFromEntry{
			{
				Source:      EnvironmentFromSourceIntegration,
				Integration: configuration.IntegrationRef{Name: "my-github-integration"},
			},
			{
				Source: EnvironmentFromSourceSecret,
				Secret: configuration.SecretRef{Secret: "some-other-secret"},
			},
		},
		[]EnvironmentVariable{
			{Name: "REPO", ValueSource: EnvironmentValueSourceLiteral, Value: strPtr("org/repo")},
			{Name: "GITHUB_TOKEN", ValueSource: EnvironmentValueSourceLiteral, Value: strPtr("override")},
		},
	)
	require.NoError(t, err)

	got := environmentValues(resolved.Variables)

	assert.Equal(t, "override", got["GITHUB_TOKEN"])
	assert.Equal(t, "org/repo", got["REPO"])
	assert.Equal(t, "secret-key-value", got["API_KEY"])
	assert.Empty(t, resolved.Usage)
	assert.Empty(t, resolved.Setups)
}

func Test__ResolveEnvironmentDisablesPagers(t *testing.T) {
	t.Parallel()

	t.Run("pagers are disabled by default", func(t *testing.T) {
		t.Parallel()

		resolved, err := ResolveEnvironment(nil, nil, nil)
		require.NoError(t, err)

		got := environmentValues(resolved.Variables)
		assert.Equal(t, "cat", got["GIT_PAGER"])
		assert.Equal(t, "cat", got["PAGER"])
	})

	t.Run("configured pager wins", func(t *testing.T) {
		t.Parallel()

		resolved, err := ResolveEnvironment(nil, nil, []EnvironmentVariable{
			{Name: "GIT_PAGER", ValueSource: EnvironmentValueSourceLiteral, Value: strPtr("less")},
		})
		require.NoError(t, err)

		var names []string
		for _, variable := range resolved.Variables {
			names = append(names, variable.Name)
		}

		assert.Equal(t, []string{"PAGER", "GIT_PAGER"}, names)
		assert.Equal(t, "less", environmentValues(resolved.Variables)["GIT_PAGER"])
	})
}

func Test__ResolveEnvironmentCollectsUsageAndSetup(t *testing.T) {
	t.Parallel()

	secrets := &contexts.SecretsContext{
		IntegrationKeys: map[string]map[string][]byte{
			"github-acme": {
				"GITHUB_TOKEN": []byte("gh-token"),
			},
			"semaphore-acme": {
				"SEMAPHORE_API_TOKEN": []byte("sem-token"),
			},
		},
		IntegrationUsage: map[string]string{
			"github-acme":    "Use gh with GITHUB_TOKEN.",
			"semaphore-acme": "Use sem-ai with SEMAPHORE_API_TOKEN.",
		},
		IntegrationSetup: map[string]string{
			"semaphore-acme": "echo install-sem-ai",
		},
		IntegrationSetupName: map[string]string{
			"semaphore-acme": "Set up Semaphore",
		},
	}

	resolved, err := ResolveEnvironment(
		secrets,
		[]EnvironmentFromEntry{
			{
				Source:      EnvironmentFromSourceIntegration,
				Integration: configuration.IntegrationRef{Name: "github-acme"},
			},
			{
				Source:      EnvironmentFromSourceIntegration,
				Integration: configuration.IntegrationRef{Name: "semaphore-acme"},
			},
		},
		nil,
	)
	require.NoError(t, err)

	assert.Equal(t, "Use gh with GITHUB_TOKEN.\n\nUse sem-ai with SEMAPHORE_API_TOKEN.", resolved.Usage)
	require.Len(t, resolved.Setups, 1)
	assert.Equal(t, "Set up Semaphore", resolved.Setups[0].Name)
	assert.Equal(t, "echo install-sem-ai", resolved.Setups[0].Script)
	assert.Equal(t, "gh-token", environmentValues(resolved.Variables)["GITHUB_TOKEN"])
	assert.Equal(t, "sem-token", environmentValues(resolved.Variables)["SEMAPHORE_API_TOKEN"])
}

func environmentValues(environment []TaskEnvironmentVariable) map[string]string {
	values := map[string]string{}
	for _, variable := range environment {
		values[variable.Name] = variable.Value
	}
	return values
}

func Test__ValidateEnvironmentFromConfigurationField(t *testing.T) {
	t.Parallel()

	fields := []configuration.Field{environmentFromConfigurationField()}

	t.Run("integration source without secret", func(t *testing.T) {
		t.Parallel()
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"environmentFrom": []any{
				map[string]any{
					"source": EnvironmentFromSourceIntegration,
					"integration": map[string]any{
						"name": "semaphore",
					},
				},
			},
		})
		require.NoError(t, err)
	})

	t.Run("secret source without integration", func(t *testing.T) {
		t.Parallel()
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"environmentFrom": []any{
				map[string]any{
					"source": EnvironmentFromSourceSecret,
					"secret": map[string]any{
						"secret": "deploy-credentials",
					},
				},
			},
		})
		require.NoError(t, err)
	})
}

func TestAttachExecutionTimeoutEnvUsesDefaultWhenUnset(t *testing.T) {
	t.Parallel()

	got := AttachExecutionTimeoutEnv(nil, 0)
	require.Len(t, got, 1)
	assert.Equal(t, EnvExecutionTimeoutSeconds, got[0].Name)
	assert.Equal(t, fmt.Sprintf("%d", DefaultExecutionTimeoutSeconds), got[0].Value)

	got = AttachExecutionTimeoutEnv([]TaskEnvironmentVariable{{Name: "KEEP", Value: "1"}}, 90)
	require.Len(t, got, 2)
	assert.Equal(t, "90", got[1].Value)
}

func strPtr(v string) *string {
	return &v
}
