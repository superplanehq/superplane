package installation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__SubstituteInstallParams(t *testing.T) {
	t.Run("replaces placeholders", func(t *testing.T) {
		yaml := []byte(`repository: "{{ install_params.repository }}"`)
		params := map[string]string{"repository": "my-org/my-repo"}
		result := SubstituteInstallParams(yaml, params)
		assert.Equal(t, `repository: "my-org/my-repo"`, string(result))
	})

	t.Run("replaces multiple occurrences", func(t *testing.T) {
		yaml := []byte("repo: {{ install_params.repo }}\nrepo2: {{ install_params.repo }}")
		params := map[string]string{"repo": "acme/app"}
		result := SubstituteInstallParams(yaml, params)
		assert.Equal(t, "repo: acme/app\nrepo2: acme/app", string(result))
	})

	t.Run("handles whitespace variations", func(t *testing.T) {
		yaml := []byte("a: {{install_params.x}}\nb: {{ install_params.x }}\nc: {{  install_params.x  }}")
		params := map[string]string{"x": "val"}
		result := SubstituteInstallParams(yaml, params)
		assert.Equal(t, "a: val\nb: val\nc: val", string(result))
	})

	t.Run("leaves unresolved placeholders", func(t *testing.T) {
		yaml := []byte("a: {{ install_params.unknown }}")
		params := map[string]string{"other": "val"}
		result := SubstituteInstallParams(yaml, params)
		assert.Equal(t, "a: {{ install_params.unknown }}", string(result))
	})

	t.Run("no placeholders is a no-op", func(t *testing.T) {
		yaml := []byte("repository: storejs")
		params := map[string]string{"repository": "my-org/my-repo"}
		result := SubstituteInstallParams(yaml, params)
		assert.Equal(t, "repository: storejs", string(result))
	})
}

func Test__ValidateInstallParams(t *testing.T) {
	schema := []InstallParam{
		{Name: "repo", Required: true},
		{Name: "script", Required: true},
		{Name: "optional", Required: false},
		{Name: "with_default", Required: true, Default: "fallback"},
	}

	t.Run("passes with all required params", func(t *testing.T) {
		err := ValidateInstallParams(schema, map[string]string{"repo": "a", "script": "b"})
		require.NoError(t, err)
	})

	t.Run("fails when required param missing", func(t *testing.T) {
		err := ValidateInstallParams(schema, map[string]string{"repo": "a"})
		require.ErrorContains(t, err, "install parameter \"script\" is required")
	})

	t.Run("passes when required param has default", func(t *testing.T) {
		err := ValidateInstallParams(schema, map[string]string{"repo": "a", "script": "b"})
		require.NoError(t, err)
	})

	t.Run("fails when required param is whitespace", func(t *testing.T) {
		err := ValidateInstallParams(schema, map[string]string{"repo": "a", "script": "  "})
		require.ErrorContains(t, err, "install parameter \"script\" is required")
	})
}

func Test__ResolveInstallParams(t *testing.T) {
	schema := []InstallParam{
		{Name: "repo", Default: "default-repo"},
		{Name: "script"},
	}

	t.Run("uses provided values", func(t *testing.T) {
		resolved := ResolveInstallParams(schema, map[string]string{"repo": "my-repo", "script": "my-script"})
		assert.Equal(t, "my-repo", resolved["repo"])
		assert.Equal(t, "my-script", resolved["script"])
	})

	t.Run("falls back to defaults", func(t *testing.T) {
		resolved := ResolveInstallParams(schema, map[string]string{"script": "s"})
		assert.Equal(t, "default-repo", resolved["repo"])
		assert.Equal(t, "s", resolved["script"])
	})

	t.Run("skips params without value or default", func(t *testing.T) {
		resolved := ResolveInstallParams(schema, map[string]string{})
		assert.Equal(t, "default-repo", resolved["repo"])
		_, hasScript := resolved["script"]
		assert.False(t, hasScript)
	})
}
