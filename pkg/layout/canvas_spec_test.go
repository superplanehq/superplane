package layout

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func Test__HasFlags(t *testing.T) {
	t.Run("nil command", func(t *testing.T) {
		require.False(t, HasFlags(core.CommandContext{}))
	})

	t.Run("auto-layout flag changed", func(t *testing.T) {
		cmd := cobra.Command{}
		cmd.Flags().String("auto-layout", "", "")
		cmd.Flags().String("auto-layout-scope", "", "")
		cmd.Flags().StringArray("auto-layout-node", nil, "")
		require.NoError(t, cmd.ParseFlags([]string{"--auto-layout=horizontal"}))
		require.True(t, HasFlags(core.CommandContext{Cmd: &cmd}))
	})

	t.Run("auto-layout-scope flag changed", func(t *testing.T) {
		cmd := cobra.Command{}
		cmd.Flags().String("auto-layout", "", "")
		cmd.Flags().String("auto-layout-scope", "", "")
		cmd.Flags().StringArray("auto-layout-node", nil, "")
		require.NoError(t, cmd.ParseFlags([]string{"--auto-layout-scope=full-canvas"}))
		require.True(t, HasFlags(core.CommandContext{Cmd: &cmd}))
	})

	t.Run("auto-layout-node flag changed", func(t *testing.T) {
		cmd := cobra.Command{}
		cmd.Flags().String("auto-layout", "", "")
		cmd.Flags().String("auto-layout-scope", "", "")
		cmd.Flags().StringArray("auto-layout-node", nil, "")
		require.NoError(t, cmd.ParseFlags([]string{"--auto-layout-node=a"}))
		require.True(t, HasFlags(core.CommandContext{Cmd: &cmd}))
	})

	t.Run("no auto-layout flags parsed", func(t *testing.T) {
		cmd := cobra.Command{}
		cmd.Flags().String("auto-layout", "", "")
		require.NoError(t, cmd.ParseFlags(nil))
		require.False(t, HasFlags(core.CommandContext{Cmd: &cmd}))
	})
}

func Test__ParseAutoLayoutRejectsInvalidAlgorithm(t *testing.T) {
	_, err := ParseAutoLayout("vertical", "", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported auto layout")
}

func Test__ParseAutoLayoutRejectsInvalidScope(t *testing.T) {
	_, err := ParseAutoLayout("horizontal", "whole-world", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported auto layout scope")
}

func Test__ParseAutoLayoutScopeFullAlias(t *testing.T) {
	autoLayout, err := ParseAutoLayout("horizontal", "full", nil)
	require.NoError(t, err)
	require.NotNil(t, autoLayout)
	require.Equal(t, "SCOPE_FULL_CANVAS", autoLayout.Scope)
}

func Test__ParseAutoLayoutDisableAcceptsOffAndNone(t *testing.T) {
	for _, v := range []string{"off", "none", "NONE"} {
		t.Run(v, func(t *testing.T) {
			al, err := ParseAutoLayout(v, "", nil)
			require.NoError(t, err)
			require.Nil(t, al)
		})
	}
}

func Test__ParseAutoLayoutDefaultsAlgorithmToHorizontal(t *testing.T) {
	autoLayout, err := ParseAutoLayout("", "connected-component", []string{"node-1", " node-2 ", "node-1"})
	if err != nil {
		t.Fatalf("ParseAutoLayout returned error: %v", err)
	}
	if autoLayout == nil {
		t.Fatalf("expected autoLayout to be set")
	}
	if autoLayout.Algorithm != "ALGORITHM_HORIZONTAL" {
		t.Fatalf("expected horizontal auto-layout, got %s", autoLayout.Algorithm)
	}
	if autoLayout.Scope != "SCOPE_CONNECTED_COMPONENT" {
		t.Fatalf("expected connected-component scope, got %s", autoLayout.Scope)
	}
	if !reflect.DeepEqual(autoLayout.NodeIDs, []string{"node-1", "node-2"}) {
		t.Fatalf("expected node ids [node-1 node-2], got %v", autoLayout.NodeIDs)
	}
}

func Test__ParseAutoLayoutDisable(t *testing.T) {
	autoLayout, err := ParseAutoLayout("disable", "", nil)
	if err != nil {
		t.Fatalf("ParseAutoLayout returned error: %v", err)
	}
	if autoLayout != nil {
		t.Fatalf("expected nil autoLayout when disabled, got %#v", autoLayout)
	}
}

func Test__ParseAutoLayoutDisableRejectsScopeOrNodes(t *testing.T) {
	if _, err := ParseAutoLayout("disable", "connected-component", nil); err == nil {
		t.Fatalf("expected error when scope is set together with disable")
	}
	if _, err := ParseAutoLayout("disable", "", []string{"node-1"}); err == nil {
		t.Fatalf("expected error when node ids are set together with disable")
	}
}
