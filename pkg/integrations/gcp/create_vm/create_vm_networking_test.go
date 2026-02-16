package createvm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParseNetworkTags(t *testing.T) {
	assert.Nil(t, ParseNetworkTags(""))
	assert.Empty(t, ParseNetworkTags("  ,  , "))
	assert.Equal(t, []string{"tag1", "tag2"}, ParseNetworkTags("tag1,tag2"))
	assert.Equal(t, []string{"tag1", "tag2"}, ParseNetworkTags("  tag1 , tag2  "))
}

func Test_resolveNetworkURL(t *testing.T) {
	t.Run("full URL returned as-is", func(t *testing.T) {
		full := "projects/p/global/networks/my-net"
		assert.Equal(t, full, resolveNetworkURL("other", full))
	})
	t.Run("short name with project", func(t *testing.T) {
		assert.Equal(t, "projects/my-proj/global/networks/default", resolveNetworkURL("my-proj", "default"))
	})
	t.Run("empty project or network returns network as-is", func(t *testing.T) {
		assert.Equal(t, "default", resolveNetworkURL("", "default"))
		assert.Equal(t, "", resolveNetworkURL("p", ""))
	})
}

func Test_resolveSubnetworkURL(t *testing.T) {
	t.Run("empty subnetwork returns empty", func(t *testing.T) {
		assert.Equal(t, "", resolveSubnetworkURL("p", "r", ""))
		assert.Equal(t, "", resolveSubnetworkURL("p", "r", "   "))
	})
	t.Run("full URL returned as-is", func(t *testing.T) {
		full := "projects/p/regions/r/subnetworks/s"
		assert.Equal(t, full, resolveSubnetworkURL("x", "y", full))
	})
	t.Run("short name with project and region", func(t *testing.T) {
		assert.Equal(t, "projects/my-proj/regions/us-central1/subnetworks/my-subnet", resolveSubnetworkURL("my-proj", "us-central1", "my-subnet"))
	})
	t.Run("empty project or region returns subnetwork as-is", func(t *testing.T) {
		assert.Equal(t, "sub", resolveSubnetworkURL("", "r", "sub"))
		assert.Equal(t, "sub", resolveSubnetworkURL("p", "", "sub"))
	})
}

func Test_BuildNetworkInterfaces(t *testing.T) {
	t.Run("empty config uses default network", func(t *testing.T) {
		out := BuildNetworkInterfaces("my-proj", "us-central1", NetworkingConfig{})
		require.Len(t, out, 1)
		assert.Equal(t, "projects/my-proj/global/networks/default", out[0].Network)
		assert.Equal(t, "", out[0].Subnetwork)
	})
	t.Run("network and subnetwork set", func(t *testing.T) {
		cfg := NetworkingConfig{
			Network:    "my-net",
			Subnetwork: "my-subnet",
		}
		out := BuildNetworkInterfaces("p", "us-central1", cfg)
		require.Len(t, out, 1)
		assert.Equal(t, "projects/p/global/networks/my-net", out[0].Network)
		assert.Equal(t, "projects/p/regions/us-central1/subnetworks/my-subnet", out[0].Subnetwork)
	})
	t.Run("ephemeral external IP by default", func(t *testing.T) {
		out := BuildNetworkInterfaces("p", "r", NetworkingConfig{Network: "default"})
		require.Len(t, out, 1)
		require.Len(t, out[0].AccessConfigs, 1)
		assert.Equal(t, "ONE_TO_ONE_NAT", out[0].AccessConfigs[0].Type)
		assert.Empty(t, out[0].AccessConfigs[0].NatIP)
	})
	t.Run("static external IP", func(t *testing.T) {
		cfg := NetworkingConfig{
			Network:           "default",
			ExternalIPType:    ExternalIPStatic,
			ExternalIPAddress: " 34.1.2.3 ",
		}
		out := BuildNetworkInterfaces("p", "r", cfg)
		require.Len(t, out, 1)
		require.Len(t, out[0].AccessConfigs, 1)
		assert.Equal(t, "34.1.2.3", out[0].AccessConfigs[0].NatIP)
	})
	t.Run("static internal IP", func(t *testing.T) {
		cfg := NetworkingConfig{
			Network:           "default",
			InternalIPType:    InternalIPStatic,
			InternalIPAddress: " 10.0.0.5 ",
		}
		out := BuildNetworkInterfaces("p", "r", cfg)
		require.Len(t, out, 1)
		assert.Equal(t, "10.0.0.5", out[0].NetworkIP)
	})
	t.Run("external IP none has no access configs", func(t *testing.T) {
		cfg := NetworkingConfig{Network: "default", ExternalIPType: ExternalIPNone}
		out := BuildNetworkInterfaces("p", "r", cfg)
		require.Len(t, out, 1)
		assert.Nil(t, out[0].AccessConfigs)
	})
	t.Run("NIC type and stack type", func(t *testing.T) {
		cfg := NetworkingConfig{
			Network:   "default",
			NicType:   NICTypeGVNIC,
			StackType: StackTypeDualStack,
		}
		out := BuildNetworkInterfaces("p", "r", cfg)
		require.Len(t, out, 1)
		assert.Equal(t, NICTypeGVNIC, out[0].NicType)
		assert.Equal(t, StackTypeDualStack, out[0].StackType)
	})
}
