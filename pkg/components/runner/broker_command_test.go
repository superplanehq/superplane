package runner

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrokerCommandJSONRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("unnamed command marshals as an object", func(t *testing.T) {
		b, err := json.Marshal(BrokerCommand{Command: "echo hi"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"command":"echo hi"}`, string(b))

		var got BrokerCommand
		require.NoError(t, json.Unmarshal(b, &got))
		assert.Equal(t, BrokerCommand{Command: "echo hi"}, got)
	})

	t.Run("named command marshals as an object", func(t *testing.T) {
		b, err := json.Marshal(BrokerCommand{Name: "Clone", Command: "git clone …"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"name":"Clone","command":"git clone …"}`, string(b))

		var got BrokerCommand
		require.NoError(t, json.Unmarshal(b, &got))
		assert.Equal(t, BrokerCommand{Name: "Clone", Command: "git clone …"}, got)
	})

	t.Run("kind and preview round trip", func(t *testing.T) {
		in := BrokerCommand{Name: "Clone", Command: "source script.sh", Kind: LiveLogKindBash, Preview: "git clone"}
		b, err := json.Marshal(in)
		require.NoError(t, err)
		assert.JSONEq(t, `{"name":"Clone","command":"source script.sh","kind":"bash","preview":"git clone"}`, string(b))

		var got BrokerCommand
		require.NoError(t, json.Unmarshal(b, &got))
		assert.Equal(t, in, got)
	})

	t.Run("legacy string array still unmarshals", func(t *testing.T) {
		var commands []BrokerCommand
		require.NoError(t, json.Unmarshal([]byte(`["echo a","echo b"]`), &commands))
		assert.Equal(t, []BrokerCommand{
			{Command: "echo a"},
			{Command: "echo b"},
		}, commands)
	})
}

func TestBrokerCommandsFromLines(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []BrokerCommand{
		{Command: "echo a", Kind: LiveLogKindBash, Preview: "echo a"},
		{Command: "echo b", Kind: LiveLogKindBash, Preview: "echo b"},
	}, BrokerCommandsFromLines([]string{"  echo a  ", "", "echo b"}))
}
