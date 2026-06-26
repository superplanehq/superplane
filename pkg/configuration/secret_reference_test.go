package configuration

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__IsSecretReference(t *testing.T) {
	cases := []struct {
		body string
		want bool
	}{
		{"secrets.foo.bar", true},
		{"  secrets.foo.bar  ", true},
		{"secrets.my-secret.my-key", true},
		{"secrets.foo", false},
		{"secrets.a.b.c", false},
		{"secrets.foo.", false},
		{"secrets..bar", false},
		{"$.payload", false},
		{"root().url", false},
		{"secretspayload", false},
		{"", false},
	}

	for _, tc := range cases {
		t.Run(tc.body, func(t *testing.T) {
			assert.Equal(t, tc.want, IsSecretReference(tc.body))
		})
	}
}

func Test__SecretReferenceRegex(t *testing.T) {
	cases := []struct {
		input string
		want  [][]string // [match, name, key]
	}{
		{
			input: "https://api.example.com/{{ secrets.api.token }}",
			want:  [][]string{{"{{ secrets.api.token }}", "api", "token"}},
		},
		{
			input: "{{secrets.my-secret.my-key}}",
			want:  [][]string{{"{{secrets.my-secret.my-key}}", "my-secret", "my-key"}},
		},
		{
			input: "Bearer {{ secrets.svc.token }} and {{ secrets.other.key }}",
			want: [][]string{
				{"{{ secrets.svc.token }}", "svc", "token"},
				{"{{ secrets.other.key }}", "other", "key"},
			},
		},
		{
			input: "{{ secrets.foo.bar.baz }}",
			want:  nil,
		},
		{
			input: "{{ $.payload.token }}",
			want:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			matches := SecretReferenceRegex.FindAllStringSubmatch(tc.input, -1)
			assert.Equal(t, tc.want, matches)
		})
	}
}

func Test__ResolveSecretReferences__NoMatch(t *testing.T) {
	out, err := ResolveSecretReferences("plain text without refs", func(name, key string) ([]byte, error) {
		t.Fatalf("lookup should not be called for %s/%s", name, key)
		return nil, nil
	})

	require.NoError(t, err)
	assert.Equal(t, "plain text without refs", out)
}

func Test__ResolveSecretReferences__SingleAndMultiple(t *testing.T) {
	lookup := func(name, key string) ([]byte, error) {
		return []byte(name + ":" + key), nil
	}

	out, err := ResolveSecretReferences("url={{ secrets.api.token }}", lookup)
	require.NoError(t, err)
	assert.Equal(t, "url=api:token", out)

	out, err = ResolveSecretReferences("a={{ secrets.s.a }}&b={{ secrets.s.b }}", lookup)
	require.NoError(t, err)
	assert.Equal(t, "a=s:a&b=s:b", out)
}

func Test__ResolveSecretReferences__TrimsWhitespace(t *testing.T) {
	var captured [][2]string
	lookup := func(name, key string) ([]byte, error) {
		captured = append(captured, [2]string{name, key})
		return []byte("v"), nil
	}

	out, err := ResolveSecretReferences("{{    secrets.api.token    }}", lookup)
	require.NoError(t, err)
	assert.Equal(t, "v", out)
	assert.Equal(t, [][2]string{{"api", "token"}}, captured)
}

func Test__ResolveSecretReferences__LookupError(t *testing.T) {
	lookup := func(name, key string) ([]byte, error) {
		return nil, fmt.Errorf("not found")
	}

	out, err := ResolveSecretReferences("{{ secrets.missing.key }}", lookup)
	assert.Equal(t, "", out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret \"missing\"")
	assert.Contains(t, err.Error(), "key \"key\"")
	assert.Contains(t, err.Error(), "not found")
}

func Test__ResolveSecretReferences__DashesInNameAndKey(t *testing.T) {
	lookup := func(name, key string) ([]byte, error) {
		assert.Equal(t, "my-secret", name)
		assert.Equal(t, "my-key", key)
		return []byte("ok"), nil
	}

	out, err := ResolveSecretReferences("{{ secrets.my-secret.my-key }}", lookup)
	require.NoError(t, err)
	assert.Equal(t, "ok", out)
}
