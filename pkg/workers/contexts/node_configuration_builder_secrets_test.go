package contexts

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSecretResolver records calls and returns canned values, so the builder
// can be tested in isolation without spinning up a database, encryptor, or
// real secrets provider.
type fakeSecretResolver struct {
	values map[string]map[string]string
	err    error
	calls  []string
}

func (f *fakeSecretResolver) Resolve(name string) (map[string]string, error) {
	f.calls = append(f.calls, name)
	if f.err != nil {
		return nil, f.err
	}
	values, ok := f.values[name]
	if !ok {
		return nil, fmt.Errorf("secret %q not found", name)
	}
	return values, nil
}

func newSecretsBuilder(resolver SecretResolver) *NodeConfigurationBuilder {
	return NewNodeConfigurationBuilder(nil, uuid.New()).WithSecretResolver(resolver)
}

func TestNodeConfigurationBuilder_DeferredPhase_KeepsSecretsPlaceholderIntact(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New())

	out, err := builder.Build(map[string]any{
		"url":   `{{ secrets("api").token }}`,
		"plain": `prefix-{{ secrets("api").token }}-suffix`,
	})
	require.NoError(t, err)
	assert.Equal(t, `{{ secrets("api").token }}`, out["url"])
	assert.Equal(t, `prefix-{{ secrets("api").token }}-suffix`, out["plain"])
}

func TestNodeConfigurationBuilder_DeferredPhase_KeepsTransformedSecretExpression(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New())

	out, err := builder.Build(map[string]any{
		"sshKey": `{{ secrets("server").sshKey + "aaa" }}`,
		"bearer": `{{ "Bearer " + secrets("api").token }}`,
	})
	require.NoError(t, err)
	assert.Equal(t, `{{ secrets("server").sshKey + "aaa" }}`, out["sshKey"])
	assert.Equal(t, `{{ "Bearer " + secrets("api").token }}`, out["bearer"])
}

func TestNodeConfigurationBuilder_RuntimePhase_ResolvesSecretCall(t *testing.T) {
	resolver := &fakeSecretResolver{
		values: map[string]map[string]string{"api": {"token": "abc123"}},
	}

	out, err := newSecretsBuilder(resolver).Build(map[string]any{
		"url": `https://example.com/{{ secrets("api").token }}/resource`,
	})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/abc123/resource", out["url"])
	assert.Equal(t, []string{"api"}, resolver.calls)
}

func TestNodeConfigurationBuilder_RuntimePhase_ResolvesTransformedSecret(t *testing.T) {
	resolver := &fakeSecretResolver{
		values: map[string]map[string]string{
			"server": {"sshKey": "secret-key"},
			"api":    {"token": "abc"},
		},
	}

	out, err := newSecretsBuilder(resolver).Build(map[string]any{
		"sshKey":     `{{ secrets("server").sshKey + "aaa" }}`,
		"authHeader": `{{ "Bearer " + secrets("api").token }}`,
	})
	require.NoError(t, err)
	assert.Equal(t, "secret-keyaaa", out["sshKey"])
	assert.Equal(t, "Bearer abc", out["authHeader"])
}

func TestNodeConfigurationBuilder_RuntimePhase_MissingSecretReturnsError(t *testing.T) {
	resolver := &fakeSecretResolver{values: map[string]map[string]string{}}

	_, err := newSecretsBuilder(resolver).Build(map[string]any{
		"url": `{{ secrets("nonexistent").token }}`,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
	assert.Contains(t, err.Error(), "not found")
}

func TestNodeConfigurationBuilder_RuntimePhase_MissingKeyFails(t *testing.T) {
	//
	// A reference to a key the secret does not define must fail the run
	// instead of silently resolving to an empty string, which would otherwise
	// send a blank token or credential downstream.
	//
	resolver := &fakeSecretResolver{
		values: map[string]map[string]string{"api": {"token": "abc"}},
	}

	_, err := newSecretsBuilder(resolver).Build(map[string]any{
		"value": `prefix-{{ secrets("api").missing }}-suffix`,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `secret "api" has no key "missing"`)
	assert.NotContains(t, err.Error(), "abc")
}

func TestNodeConfigurationBuilder_RuntimePhase_MissingKeyFailsForBareExpression(t *testing.T) {
	//
	// Bare-expression fields (If, Filter, Merge stop-if, Loop until) route
	// through ResolveExpression and must reject missing keys too.
	//
	resolver := &fakeSecretResolver{
		values: map[string]map[string]string{"api": {"token": "abc"}},
	}

	_, err := newSecretsBuilder(resolver).ResolveExpression(`secrets("api").missing == "x"`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `secret "api" has no key "missing"`)
}

func TestNodeConfigurationBuilder_RuntimePhase_ResolvesSecretOncePerEvaluation(t *testing.T) {
	//
	// The static key-existence check and the VM evaluation must share a single
	// resolution per secret name, so repeating the same secret in one
	// expression does not hit the resolver twice.
	//
	resolver := &fakeSecretResolver{
		values: map[string]map[string]string{"api": {"token": "abc"}},
	}

	out, err := newSecretsBuilder(resolver).ResolveExpression(`secrets("api").token + secrets("api").token`)
	require.NoError(t, err)
	assert.Equal(t, "abcabc", out)
	assert.Equal(t, []string{"api"}, resolver.calls)
}

func TestNodeConfigurationBuilder_RuntimePhase_WholeSecretMapIsRejected(t *testing.T) {
	//
	// Selecting no key (secrets("api") instead of secrets("api").token) would
	// otherwise stringify the whole decrypted map and leak every key/value.
	// It must be rejected, and the error must not contain any secret value.
	//
	resolver := &fakeSecretResolver{
		values: map[string]map[string]string{"api": {"token": "supersecret", "key": "anotherone"}},
	}

	for _, expression := range []string{
		`{{ secrets("api") }}`,
		`prefix-{{ secrets("api") }}-suffix`,
		`{{ secrets("api") }} and {{ secrets("api").token }}`,
	} {
		t.Run(expression, func(t *testing.T) {
			_, err := newSecretsBuilder(resolver).Build(map[string]any{"value": expression})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "secrets() must select a specific key")
			assert.NotContains(t, err.Error(), "supersecret")
			assert.NotContains(t, err.Error(), "anotherone")
		})
	}
}

func TestNodeConfigurationBuilder_RuntimePhase_BareExpressionRejectsWholeSecretMap(t *testing.T) {
	//
	// Bare-expression fields (If, Filter, Merge stop-if, Loop until) evaluate
	// through ResolveExpression rather than ResolveTemplateExpressions. A
	// whole-secret-map expression (secrets("api") without a key) must be
	// rejected here too, otherwise components format the non-boolean result
	// with %v and persist the decrypted secret in their failure message.
	//
	resolver := &fakeSecretResolver{
		values: map[string]map[string]string{"api": {"token": "supersecret", "key": "anotherone"}},
	}
	builder := newSecretsBuilder(resolver)

	_, err := builder.ResolveExpression(`secrets("api")`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secrets() must select a specific key")
	assert.NotContains(t, err.Error(), "supersecret")
	assert.NotContains(t, err.Error(), "anotherone")

	//
	// Selecting a specific key still works through the bare-expression path.
	//
	out, err := builder.ResolveExpression(`secrets("api").token`)
	require.NoError(t, err)
	assert.Equal(t, "supersecret", out)
}

func TestNodeConfigurationBuilder_NoResolver_SecretsCallErrors(t *testing.T) {
	//
	// If a placeholder containing secrets() somehow reaches a builder with no
	// resolver and bypasses the deferred-mode skip (eg. via the
	// ResolveExpression API), the call should fail rather than silently
	// returning an empty value.
	//
	builder := NewNodeConfigurationBuilder(nil, uuid.New())

	_, err := builder.ResolveExpression(`secrets("api").token`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secrets()")
}

func TestNodeConfigurationBuilder_SecretsIdentifierIsReserved(t *testing.T) {
	resolver := &fakeSecretResolver{values: map[string]map[string]string{}}
	builder := newSecretsBuilder(resolver)

	_, err := builder.ResolveExpressionWithExtraVariables(`"x"`, map[string]any{
		"secrets": "shadowed",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"secrets"`)
	assert.Contains(t, err.Error(), "reserved")
}

func TestNodeConfigurationBuilder_RuntimePhase_RebuildsResolveAfresh(t *testing.T) {
	//
	// Simulates the retry path: each call into invokeExecutionComponentHook
	// reuses the stored execution.Configuration (with deferred secret
	// placeholders) and a fresh RuntimeSecretResolver. Each call must call
	// the resolver, so updated secret values flow through every attempt.
	//
	resolver := &fakeSecretResolver{
		values: map[string]map[string]string{"api": {"token": "v1"}},
	}
	stored := map[string]any{"url": `https://example.com/{{ secrets("api").token }}`}

	first, err := newSecretsBuilder(resolver).Build(stored)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/v1", first["url"])

	resolver.values["api"]["token"] = "v2"
	resolver.calls = nil

	second, err := newSecretsBuilder(resolver).Build(stored)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/v2", second["url"])
	assert.Equal(t, []string{"api"}, resolver.calls, "resolver must be invoked on every attempt")
}

func TestExpressionInjectsSecret(t *testing.T) {
	cases := []struct {
		expression string
		expected   bool
	}{
		{`secrets("x").y`, true},
		{`secrets("x").y + "aaa"`, true},
		{`"Bearer " + secrets("api").token`, true},
		{`$["node"].field`, false},
		{`root().path`, false},
		{`previous()`, false},
		{`"just a string with secrets( in it"`, false},
		{`malformed (`, false},
	}

	for _, tc := range cases {
		t.Run(tc.expression, func(t *testing.T) {
			assert.Equal(t, tc.expected, expressionInjectsSecret(tc.expression))
		})
	}
}
