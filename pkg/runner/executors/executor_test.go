package executors

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/extensions/hub/protocol"
)

func Test__BundleURL(t *testing.T) {
	t.Parallel()

	executor, err := New(Config{
		HubURL:     "http://example.com?existing=value",
		CacheDir:   "/tmp/cache",
		DenoBinary: "deno",
		Runner:     ExecRunner{},
	})

	require.NoError(t, err)
	bundleURL, err := executor.bundleURL(&protocol.InvokeExtension{
		BundleToken: "bundle-token",
	})

	require.NoError(t, err)
	parsed, err := url.Parse(bundleURL)
	require.NoError(t, err)

	require.Equal(t, "http", parsed.Scheme)
	require.Equal(t, "example.com", parsed.Host)
	require.Equal(t, "/api/v1/extensions/bundle.js", parsed.Path)
	require.Equal(t, "value", parsed.Query().Get("existing"))
	require.Equal(t, "bundle-token", parsed.Query().Get(protocol.QueryToken))
}
