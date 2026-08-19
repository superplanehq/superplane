package factories

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cli "github.com/superplanehq/superplane/test/support/cli"
)

func TestResolveFactoryID(t *testing.T) {
	factoryID := "11111111-1111-1111-1111-111111111111"

	t.Run("uses explicit factory flag", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(
				`{"factories":[{"id":%q,"name":"shipping"}]}`,
				factoryID,
			)))
		}))
		t.Cleanup(server.Close)

		ctx, _ := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{
			ActiveFactory: "ignored",
		})
		got, err := ResolveFactoryID(ctx, "shipping")
		require.NoError(t, err)
		assert.Equal(t, factoryID, got)
	})

	t.Run("falls back to active factory", func(t *testing.T) {
		ctx, _ := cli.NewCommandContextWithConfig(t, nil, "text", &cli.FakeConfig{
			ActiveFactory: factoryID,
		})
		got, err := ResolveFactoryID(ctx, "")
		require.NoError(t, err)
		assert.Equal(t, factoryID, got)
	})

	t.Run("errors when factory missing", func(t *testing.T) {
		ctx, _ := cli.NewCommandContextWithConfig(t, nil, "text", &cli.FakeConfig{})
		_, err := ResolveFactoryID(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "superplane factory active")
	})
}
