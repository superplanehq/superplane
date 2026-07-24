package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/test/support/contexts"
)

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

	got := map[string]string{}
	for _, variable := range resolved {
		got[variable.Name] = variable.Value
	}

	assert.Equal(t, "override", got["GITHUB_TOKEN"])
	assert.Equal(t, "org/repo", got["REPO"])
	assert.Equal(t, "secret-key-value", got["API_KEY"])
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

func strPtr(v string) *string {
	return &v
}
