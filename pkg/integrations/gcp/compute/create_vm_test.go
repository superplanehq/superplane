package compute

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	compute "google.golang.org/api/compute/v1"
)

func Test_CreateVMConfiguration(t *testing.T) {
	c := &CreateVM{}
	fields := c.Configuration()
	require.NotEmpty(t, fields)

	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.Name)
	}
	assert.Contains(t, names, "instanceName")
	assert.Contains(t, names, "region")
	assert.Contains(t, names, "zone")
	assert.Contains(t, names, "machineType")
	assert.Contains(t, names, "bootDiskSourceType")
	assert.Contains(t, names, "bootDiskOS")
	assert.Contains(t, names, "bootDiskPublicImage")
	assert.Contains(t, names, "bootDiskCustomImage")
	assert.Contains(t, names, "bootDiskSnapshot")
	assert.Contains(t, names, "bootDiskExistingDisk")
	assert.Contains(t, names, "bootDiskType")
	assert.Contains(t, names, "bootDiskSizeGb")
	assert.Contains(t, names, "bootDiskEncryptionKey")
	assert.Contains(t, names, "bootDiskSnapshotSchedule")
	assert.Contains(t, names, "bootDiskAutoDelete")
	assert.Contains(t, names, "localSSDCount")
	assert.Contains(t, names, "additionalDisks")
	assert.Contains(t, names, "network")
	assert.Contains(t, names, "serviceAccount")
	assert.Contains(t, names, "labels")
}

func Test_resolveDiskTypeURL(t *testing.T) {
	t.Run("full path returned as-is", func(t *testing.T) {
		full := "projects/p/zones/z/diskTypes/pd-ssd"
		assert.Equal(t, full, resolveDiskTypeURL("x", "y", full))
	})
	t.Run("short name with project and zone", func(t *testing.T) {
		assert.Equal(t, "projects/my-proj/zones/us-central1-a/diskTypes/pd-ssd", resolveDiskTypeURL("my-proj", "us-central1-a", "pd-ssd"))
	})
	t.Run("empty project or zone returns diskType as-is", func(t *testing.T) {
		assert.Equal(t, "pd-ssd", resolveDiskTypeURL("", "z", "pd-ssd"))
		assert.Equal(t, "pd-ssd", resolveDiskTypeURL("p", "", "pd-ssd"))
	})
}

func Test_BuildBootDisk(t *testing.T) {
	t.Run("existing disk source", func(t *testing.T) {
		cfg := BootDiskConfig{
			SourceDisk: "projects/p/zones/z/disks/d1",
			AutoDelete: true,
		}
		out := BuildBootDisk("p", "z", cfg)
		require.NotNil(t, out)
		assert.True(t, out.Boot)
		assert.Equal(t, "projects/p/zones/z/disks/d1", out.Source)
		assert.True(t, out.AutoDelete)
	})
	t.Run("image source", func(t *testing.T) {
		cfg := BootDiskConfig{
			SourceImage: "projects/debian-cloud/global/images/debian-12",
			SizeGb:      20,
			DiskType:    "pd-ssd",
			AutoDelete:  true,
		}
		out := BuildBootDisk("my-proj", "us-central1-a", cfg)
		require.NotNil(t, out)
		assert.True(t, out.Boot)
		require.NotNil(t, out.InitializeParams)
		assert.Equal(t, "projects/debian-cloud/global/images/debian-12", out.InitializeParams.SourceImage)
		assert.Equal(t, int64(20), out.InitializeParams.DiskSizeGb)
		assert.Contains(t, out.InitializeParams.DiskType, "pd-ssd")
	})
	t.Run("default disk type and size when empty", func(t *testing.T) {
		cfg := BootDiskConfig{SourceImage: "img", SizeGb: 0, DiskType: ""}
		out := BuildBootDisk("p", "z", cfg)
		require.NotNil(t, out)
		require.NotNil(t, out.InitializeParams)
		assert.Equal(t, int64(DefaultDiskSizeGb), out.InitializeParams.DiskSizeGb)
		assert.Contains(t, out.InitializeParams.DiskType, DefaultDiskType)
	})
}

func Test_BuildAdditionalDisks(t *testing.T) {
	assert.Nil(t, BuildAdditionalDisks("p", "z", nil))
	t.Run("existing disk attach", func(t *testing.T) {
		disks := []AdditionalDisk{{SourceDisk: "projects/p/zones/z/disks/d2", AutoDelete: false}}
		out := BuildAdditionalDisks("p", "z", disks)
		require.Len(t, out, 1)
		assert.False(t, out[0].Boot)
		assert.Equal(t, "projects/p/zones/z/disks/d2", out[0].Source)
	})
	t.Run("new disk", func(t *testing.T) {
		disks := []AdditionalDisk{{Name: "data", SizeGb: 100, DiskType: "pd-ssd", AutoDelete: true}}
		out := BuildAdditionalDisks("my-proj", "us-central1-a", disks)
		require.Len(t, out, 1)
		require.NotNil(t, out[0].InitializeParams)
		assert.Equal(t, "data", out[0].InitializeParams.DiskName)
		assert.Equal(t, int64(100), out[0].InitializeParams.DiskSizeGb)
		assert.Contains(t, out[0].InitializeParams.DiskType, "pd-ssd")
	})
}

func Test_BuildLocalSSDDisks(t *testing.T) {
	assert.Nil(t, BuildLocalSSDDisks("p", "z", 0))
	assert.Nil(t, BuildLocalSSDDisks("p", "z", -1))
	t.Run("count capped at 8", func(t *testing.T) {
		out := BuildLocalSSDDisks("p", "z", 10)
		require.Len(t, out, 8)
	})
	t.Run("builds SCRATCH NVME disks", func(t *testing.T) {
		out := BuildLocalSSDDisks("p", "z", 2)
		require.Len(t, out, 2)
		for i, d := range out {
			assert.Equal(t, "SCRATCH", d.Type)
			assert.Equal(t, "NVME", d.Interface)
			assert.True(t, d.AutoDelete)
			assert.Equal(t, fmt.Sprintf("local-ssd-%d", i), d.DeviceName)
		}
	})
}

func Test_BuildShieldedInstanceConfig(t *testing.T) {
	t.Run("disabled returns nil", func(t *testing.T) {
		cfg := SecurityConfig{ShieldedVM: false}
		assert.Nil(t, BuildShieldedInstanceConfig(cfg))
	})
	t.Run("enabled with zero values for options", func(t *testing.T) {
		cfg := SecurityConfig{ShieldedVM: true}
		out := BuildShieldedInstanceConfig(cfg)
		require.NotNil(t, out)
		assert.False(t, out.EnableSecureBoot)
		assert.False(t, out.EnableVtpm)
		assert.False(t, out.EnableIntegrityMonitoring)
	})
	t.Run("secure boot can be enabled", func(t *testing.T) {
		cfg := SecurityConfig{
			ShieldedVM:                          true,
			ShieldedVMEnableSecureBoot:          true,
			ShieldedVMEnableVtpm:                false,
			ShieldedVMEnableIntegrityMonitoring: false,
		}
		out := BuildShieldedInstanceConfig(cfg)
		require.NotNil(t, out)
		assert.True(t, out.EnableSecureBoot)
		assert.False(t, out.EnableVtpm)
		assert.False(t, out.EnableIntegrityMonitoring)
	})
}

func Test_BuildConfidentialInstanceConfig(t *testing.T) {
	t.Run("disabled returns nil", func(t *testing.T) {
		cfg := SecurityConfig{ConfidentialVM: false}
		assert.Nil(t, BuildConfidentialInstanceConfig(cfg))
	})
	t.Run("enabled defaults to SEV", func(t *testing.T) {
		cfg := SecurityConfig{ConfidentialVM: true}
		out := BuildConfidentialInstanceConfig(cfg)
		require.NotNil(t, out)
		assert.True(t, out.EnableConfidentialCompute)
		assert.Equal(t, ConfidentialInstanceTypeSEV, out.ConfidentialInstanceType)
	})
	t.Run("explicit type", func(t *testing.T) {
		cfg := SecurityConfig{
			ConfidentialVM:     true,
			ConfidentialVMType: ConfidentialInstanceTypeTDX,
		}
		out := BuildConfidentialInstanceConfig(cfg)
		require.NotNil(t, out)
		assert.Equal(t, ConfidentialInstanceTypeTDX, out.ConfidentialInstanceType)
	})
}

func Test_NormalizeOAuthScopes(t *testing.T) {
	assert.Nil(t, NormalizeOAuthScopes(nil))
	assert.Nil(t, NormalizeOAuthScopes([]string{}))
	assert.Nil(t, NormalizeOAuthScopes([]string{"", "  ", ""}))
	assert.Equal(t, []string{"https://www.googleapis.com/auth/cloud-platform"}, NormalizeOAuthScopes([]string{"  https://www.googleapis.com/auth/cloud-platform  "}))
	assert.Equal(t, []string{"scope1", "scope2"}, NormalizeOAuthScopes([]string{" scope1 ", "", " scope2 "}))
}

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

func Test_BuildInstanceTags(t *testing.T) {
	t.Run("only network tags", func(t *testing.T) {
		out := BuildInstanceTags("tag1, tag2", nil)
		assert.Equal(t, []string{"tag1", "tag2"}, out)
	})
	t.Run("only firewall tags", func(t *testing.T) {
		out := BuildInstanceTags("", []string{"allow-ssh", "http-server"})
		assert.Equal(t, []string{"allow-ssh", "http-server"}, out)
	})
	t.Run("merged and deduplicated", func(t *testing.T) {
		out := BuildInstanceTags("tag1, tag2", []string{"tag2", "allow-ssh"})
		assert.Equal(t, []string{"tag1", "tag2", "allow-ssh"}, out)
	})
	t.Run("empty both", func(t *testing.T) {
		out := BuildInstanceTags("", nil)
		assert.Nil(t, out)
	})
}

func Test_BuildInstanceMetadata(t *testing.T) {
	t.Run("empty config returns nil", func(t *testing.T) {
		out := BuildInstanceMetadata(ManagementConfig{})
		assert.Nil(t, out)
	})
	t.Run("startup and shutdown script", func(t *testing.T) {
		cfg := ManagementConfig{
			StartupScript:  " echo hello ",
			ShutdownScript: " echo bye ",
		}
		out := BuildInstanceMetadata(cfg)
		require.NotNil(t, out)
		require.Len(t, out.Items, 2)
		keys := make(map[string]string)
		for _, it := range out.Items {
			keys[it.Key] = *it.Value
		}
		assert.Equal(t, "echo hello", keys["startup-script"])
		assert.Equal(t, "echo bye", keys["shutdown-script"])
	})
	t.Run("custom metadata items", func(t *testing.T) {
		cfg := ManagementConfig{
			MetadataItems: []MetadataKeyValue{
				{Key: "env", Value: "prod"},
				{Key: "team", Value: "platform"},
			},
		}
		out := BuildInstanceMetadata(cfg)
		require.NotNil(t, out)
		require.Len(t, out.Items, 2)
		keys := make(map[string]string)
		for _, it := range out.Items {
			keys[it.Key] = *it.Value
		}
		assert.Equal(t, "prod", keys["env"])
		assert.Equal(t, "platform", keys["team"])
	})
	t.Run("skips empty key and dedupes", func(t *testing.T) {
		cfg := ManagementConfig{
			MetadataItems: []MetadataKeyValue{
				{Key: "", Value: "v"},
				{Key: "k", Value: "first"},
				{Key: "k", Value: "second"},
			},
		}
		out := BuildInstanceMetadata(cfg)
		require.NotNil(t, out)
		require.Len(t, out.Items, 1)
		assert.Equal(t, "first", *out.Items[0].Value)
	})
}

func Test_BuildScheduling(t *testing.T) {
	t.Run("default automatic restart and migrate", func(t *testing.T) {
		out := BuildScheduling(ManagementConfig{})
		require.NotNil(t, out)
		require.NotNil(t, out.AutomaticRestart)
		assert.True(t, *out.AutomaticRestart)
		assert.Equal(t, OnHostMaintenanceMigrate, out.OnHostMaintenance)
	})
	t.Run("explicit automatic restart false", func(t *testing.T) {
		f := false
		out := BuildScheduling(ManagementConfig{AutomaticRestart: &f})
		require.NotNil(t, out)
		require.NotNil(t, out.AutomaticRestart)
		assert.False(t, *out.AutomaticRestart)
	})
	t.Run("on host maintenance terminate", func(t *testing.T) {
		out := BuildScheduling(ManagementConfig{OnHostMaintenance: OnHostMaintenanceTerminate})
		require.NotNil(t, out)
		assert.Equal(t, OnHostMaintenanceTerminate, out.OnHostMaintenance)
	})
}

func Test_trimmedNonEmptyStrings(t *testing.T) {
	assert.Nil(t, trimmedNonEmptyStrings(nil))
	assert.Empty(t, trimmedNonEmptyStrings([]string{"", "  ", ""}))
	assert.Equal(t, []string{"a", "b"}, trimmedNonEmptyStrings([]string{" a ", "", " b "}))
}

func Test_normalizeNodeAffinityOperator(t *testing.T) {
	assert.Equal(t, NodeAffinityOperatorIn, normalizeNodeAffinityOperator(""))
	assert.Equal(t, NodeAffinityOperatorIn, normalizeNodeAffinityOperator("IN"))
	assert.Equal(t, NodeAffinityOperatorNotIn, normalizeNodeAffinityOperator("NOT_IN"))
	assert.Equal(t, NodeAffinityOperatorIn, normalizeNodeAffinityOperator("unknown"))
}

func Test_BuildGuestAccelerators(t *testing.T) {
	t.Run("empty config returns nil", func(t *testing.T) {
		out := BuildGuestAccelerators(AdvancedConfig{})
		assert.Nil(t, out)
	})
	t.Run("skips empty type or zero count", func(t *testing.T) {
		cfg := AdvancedConfig{
			GuestAccelerators: []GuestAcceleratorEntry{
				{AcceleratorType: "", AcceleratorCount: 1},
				{AcceleratorType: "nvidia-tesla-t4", AcceleratorCount: 0},
				{AcceleratorType: "  nvidia-tesla-t4  ", AcceleratorCount: 2},
			},
		}
		out := BuildGuestAccelerators(cfg)
		require.Len(t, out, 1)
		assert.Equal(t, "nvidia-tesla-t4", out[0].AcceleratorType)
		assert.Equal(t, int64(2), out[0].AcceleratorCount)
	})
	t.Run("builds multiple accelerators", func(t *testing.T) {
		cfg := AdvancedConfig{
			GuestAccelerators: []GuestAcceleratorEntry{
				{AcceleratorType: "nvidia-tesla-t4", AcceleratorCount: 1},
				{AcceleratorType: "nvidia-tesla-v100", AcceleratorCount: 2},
			},
		}
		out := BuildGuestAccelerators(cfg)
		require.Len(t, out, 2)
		assert.Equal(t, "nvidia-tesla-t4", out[0].AcceleratorType)
		assert.Equal(t, int64(1), out[0].AcceleratorCount)
		assert.Equal(t, "nvidia-tesla-v100", out[1].AcceleratorType)
		assert.Equal(t, int64(2), out[1].AcceleratorCount)
	})
}

func Test_BuildNodeAffinities(t *testing.T) {
	t.Run("empty config returns nil", func(t *testing.T) {
		out := BuildNodeAffinities(AdvancedConfig{})
		assert.Nil(t, out)
	})
	t.Run("skips empty key or empty values", func(t *testing.T) {
		cfg := AdvancedConfig{
			NodeAffinities: []NodeAffinityEntry{
				{Key: "", Operator: NodeAffinityOperatorIn, Values: []string{"v1"}},
				{Key: "key", Operator: NodeAffinityOperatorIn, Values: nil},
				{Key: "  key  ", Operator: NodeAffinityOperatorNotIn, Values: []string{"  v1  "}},
			},
		}
		out := BuildNodeAffinities(cfg)
		require.Len(t, out, 1)
		assert.Equal(t, "key", out[0].Key)
		assert.Equal(t, NodeAffinityOperatorNotIn, out[0].Operator)
		assert.Equal(t, []string{"v1"}, out[0].Values)
	})
}

func Test_BuildInstanceResourcePolicies(t *testing.T) {
	assert.Nil(t, BuildInstanceResourcePolicies(AdvancedConfig{}))
	assert.Equal(t, []string{"policy-1"}, BuildInstanceResourcePolicies(AdvancedConfig{
		ResourcePolicies: []string{"  policy-1  "},
	}))
	assert.Equal(t, []string{"a", "b"}, BuildInstanceResourcePolicies(AdvancedConfig{
		ResourcePolicies: []string{" a ", "", " b "},
	}))
}

func Test_BuildLabels(t *testing.T) {
	assert.Nil(t, BuildLabels(AdvancedConfig{}))
	t.Run("skips empty key and dedupes by key", func(t *testing.T) {
		cfg := AdvancedConfig{
			Labels: []LabelEntry{
				{Key: "", Value: "v"},
				{Key: "env", Value: " prod "},
				{Key: "env", Value: "ignored"},
			},
		}
		out := BuildLabels(cfg)
		require.Len(t, out, 1)
		assert.Equal(t, "prod", out["env"])
	})
}

func Test_ApplyAdvancedScheduling(t *testing.T) {
	t.Run("nil scheduling is no-op", func(t *testing.T) {
		ApplyAdvancedScheduling(nil, AdvancedConfig{MinNodeCpus: 4})
	})
	t.Run("sets node affinities and min node cpus", func(t *testing.T) {
		s := &compute.Scheduling{}
		cfg := AdvancedConfig{
			MinNodeCpus: 4,
			NodeAffinities: []NodeAffinityEntry{
				{Key: "compute.soletenant.node", Operator: NodeAffinityOperatorIn, Values: []string{"node-1"}},
			},
		}
		ApplyAdvancedScheduling(s, cfg)
		require.NotNil(t, s.MinNodeCpus)
		assert.Equal(t, int64(4), s.MinNodeCpus)
		require.Len(t, s.NodeAffinities, 1)
		assert.Equal(t, "compute.soletenant.node", s.NodeAffinities[0].Key)
		assert.Equal(t, []string{"node-1"}, s.NodeAffinities[0].Values)
	})
}

func Test_resolveImageURL(t *testing.T) {
	t.Run("full path returned as-is", func(t *testing.T) {
		full := "projects/foo/global/images/my-image"
		assert.Equal(t, full, resolveImageURL("proj", full))
	})
	t.Run("short name with project", func(t *testing.T) {
		assert.Equal(t, "projects/my-proj/global/images/debian-12", resolveImageURL("my-proj", "debian-12"))
	})
	t.Run("empty project returns ref as-is", func(t *testing.T) {
		assert.Equal(t, "debian-12", resolveImageURL("", "debian-12"))
	})
}

func Test_resolveSnapshotURL(t *testing.T) {
	t.Run("full path returned as-is", func(t *testing.T) {
		full := "projects/foo/global/snapshots/snap-1"
		assert.Equal(t, full, resolveSnapshotURL("proj", full))
	})
	t.Run("short name with project", func(t *testing.T) {
		assert.Equal(t, "projects/my-proj/global/snapshots/snap-1", resolveSnapshotURL("my-proj", "snap-1"))
	})
	t.Run("empty project returns ref as-is", func(t *testing.T) {
		assert.Equal(t, "snap-1", resolveSnapshotURL("", "snap-1"))
	})
}

func Test_resolveDiskURL(t *testing.T) {
	t.Run("full path returned as-is", func(t *testing.T) {
		full := "projects/foo/zones/z1/disks/d1"
		assert.Equal(t, full, resolveDiskURL("p", "z", full))
	})
	t.Run("short name with project and zone", func(t *testing.T) {
		assert.Equal(t, "projects/my-proj/zones/us-central1-a/disks/my-disk", resolveDiskURL("my-proj", "us-central1-a", "my-disk"))
	})
	t.Run("empty project or zone returns ref as-is", func(t *testing.T) {
		assert.Equal(t, "my-disk", resolveDiskURL("", "z", "my-disk"))
		assert.Equal(t, "my-disk", resolveDiskURL("p", "", "my-disk"))
	})
}

func Test_deriveRegionFromZone(t *testing.T) {
	t.Run("standard zone", func(t *testing.T) {
		assert.Equal(t, "us-central1", deriveRegionFromZone("us-central1-a"))
		assert.Equal(t, "europe-west1", deriveRegionFromZone("europe-west1-b"))
	})
	t.Run("single segment returns empty", func(t *testing.T) {
		assert.Equal(t, "", deriveRegionFromZone("nohyphen"))
	})
	t.Run("zone with multiple hyphens", func(t *testing.T) {
		assert.Equal(t, "us-east1", deriveRegionFromZone("us-east1-c"))
	})
}

func Test_ensureMetadataItem(t *testing.T) {
	t.Run("nil metadata creates new with item", func(t *testing.T) {
		m := ensureMetadataItem(nil, "key1", "v1")
		require.NotNil(t, m)
		require.Len(t, m.Items, 1)
		assert.Equal(t, "key1", m.Items[0].Key)
		assert.Equal(t, "v1", *m.Items[0].Value)
	})
	t.Run("adds to existing metadata", func(t *testing.T) {
		v1 := "v1"
		m := &compute.Metadata{Items: []*compute.MetadataItems{{Key: "a", Value: &v1}}}
		m = ensureMetadataItem(m, "b", "v2")
		require.Len(t, m.Items, 2)
		assert.Equal(t, "b", m.Items[1].Key)
		assert.Equal(t, "v2", *m.Items[1].Value)
	})
	t.Run("updates existing key", func(t *testing.T) {
		v1 := "old"
		m := &compute.Metadata{Items: []*compute.MetadataItems{{Key: "k", Value: &v1}}}
		m = ensureMetadataItem(m, "k", "new")
		require.Len(t, m.Items, 1)
		assert.Equal(t, "new", *m.Items[0].Value)
	})
}

func Test_InstancePayloadFromGetResponse(t *testing.T) {
	t.Run("valid response", func(t *testing.T) {
		body := []byte(`{
			"id": "1234567890123456789",
			"name": "my-vm",
			"selfLink": "https://www.googleapis.com/compute/v1/projects/p/zones/us-central1-a/instances/my-vm",
			"status": "RUNNING",
			"zone": "https://www.googleapis.com/compute/v1/projects/p/zones/us-central1-a",
			"machineType": "https://www.googleapis.com/compute/v1/projects/p/zones/us-central1-a/machineTypes/e2-medium",
			"networkInterfaces": [
				{"networkIP": "10.0.0.2", "accessConfigs": [{"natIP": "34.1.2.3"}]}
			]
		}`)
		payload, err := InstancePayloadFromGetResponse(body, "us-central1-a")
		require.NoError(t, err)
		assert.Equal(t, "1234567890123456789", payload["instanceId"])
		assert.Equal(t, "my-vm", payload["name"])
		assert.Equal(t, "RUNNING", payload["status"])
		assert.Equal(t, "us-central1-a", payload["zone"])
		assert.Equal(t, "e2-medium", payload["machineType"])
		assert.Equal(t, "10.0.0.2", payload["internalIP"])
		assert.Equal(t, "34.1.2.3", payload["externalIP"])
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := InstancePayloadFromGetResponse([]byte(`{invalid`), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse")
	})

	t.Run("empty network interfaces still returns payload", func(t *testing.T) {
		body := []byte(`{"id":"1","name":"v","selfLink":"","status":"","zone":"","machineType":""}`)
		payload, err := InstancePayloadFromGetResponse(body, "z1")
		require.NoError(t, err)
		assert.Equal(t, "1", payload["instanceId"])
		assert.Equal(t, "z1", payload["zone"])
	})
}

func Test_BuildInstanceFromConfig(t *testing.T) {
	minimalConfig := func() CreateVMConfig {
		return CreateVMConfig{
			InstanceName: "test-vm",
			Zone:         "us-central1-a",
			Region:       "us-central1",
			MachineType:  "e2-medium",
			OSAndStorageConfig: OSAndStorageConfig{
				BootDiskSourceType:  BootDiskSourcePublicImage,
				BootDiskPublicImage: "projects/debian-cloud/global/images/family/debian-12",
			},
			NetworkingConfig: NetworkingConfig{Network: "default"},
		}
	}

	t.Run("minimal valid config builds instance", func(t *testing.T) {
		config := minimalConfig()
		inst, err := BuildInstanceFromConfig("my-proj", "us-central1-a", "us-central1", config)
		require.NoError(t, err)
		require.NotNil(t, inst)
		assert.Equal(t, "test-vm", inst.Name)
		assert.Contains(t, inst.MachineType, "e2-medium")
		require.Len(t, inst.Disks, 1)
		assert.True(t, inst.Disks[0].Boot)
		require.Len(t, inst.NetworkInterfaces, 1)
		assert.NotEmpty(t, inst.NetworkInterfaces[0].Network)
	})

	t.Run("empty instance name returns error", func(t *testing.T) {
		config := minimalConfig()
		config.InstanceName = ""
		_, err := BuildInstanceFromConfig("p", "z", "r", config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "instance name")
	})

	t.Run("empty networking uses default network", func(t *testing.T) {
		config := minimalConfig()
		config.NetworkingConfig = NetworkingConfig{}
		inst, err := BuildInstanceFromConfig("p", "us-central1-a", "us-central1", config)
		require.NoError(t, err)
		require.Len(t, inst.NetworkInterfaces, 1)
		assert.Equal(t, "projects/p/global/networks/default", inst.NetworkInterfaces[0].Network)
	})
}

func Test_validateCreateVMConfig(t *testing.T) {
	t.Run("valid config returns ok", func(t *testing.T) {
		config := CreateVMConfig{
			InstanceName: "my-vm-01",
			Zone:         "us-central1-a",
			MachineType:  "e2-medium",
		}
		msg, ok := validateCreateVMConfig(config)
		require.True(t, ok)
		assert.Empty(t, msg)
	})

	t.Run("empty instance name returns error", func(t *testing.T) {
		config := CreateVMConfig{Zone: "us-central1-a", MachineType: "e2-medium"}
		msg, ok := validateCreateVMConfig(config)
		require.False(t, ok)
		assert.Equal(t, "instance name is required", msg)
	})

	t.Run("instance name must match GCP pattern", func(t *testing.T) {
		for _, name := range []string{"My-VM", "123vm", "vm_01", "-vm", "vm-"} {
			config := CreateVMConfig{InstanceName: name, Zone: "us-central1-a", MachineType: "e2-medium"}
			msg, ok := validateCreateVMConfig(config)
			require.False(t, ok, "expected invalid for %q", name)
			assert.Contains(t, msg, "instance name must be")
		}
	})

	t.Run("valid instance names", func(t *testing.T) {
		for _, name := range []string{"my-vm", "my-vm-01", "a1", "z"} {
			config := CreateVMConfig{InstanceName: name, Zone: "us-central1-a", MachineType: "e2-medium"}
			_, ok := validateCreateVMConfig(config)
			require.True(t, ok, "expected valid for %q", name)
		}
	})

	t.Run("empty zone returns error", func(t *testing.T) {
		config := CreateVMConfig{InstanceName: "my-vm", Zone: "  ", MachineType: "e2-medium"}
		msg, ok := validateCreateVMConfig(config)
		require.False(t, ok)
		assert.Equal(t, "zone is required", msg)
	})

	t.Run("empty machine type returns error", func(t *testing.T) {
		config := CreateVMConfig{InstanceName: "my-vm", Zone: "us-central1-a", MachineType: ""}
		msg, ok := validateCreateVMConfig(config)
		require.False(t, ok)
		assert.Equal(t, "machine type is required", msg)
	})
}
