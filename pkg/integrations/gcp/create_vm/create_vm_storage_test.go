package createvm

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func Test_isAllowedBootDiskType(t *testing.T) {
	assert.True(t, isAllowedBootDiskType("pd-balanced"))
	assert.True(t, isAllowedBootDiskType("pd-ssd"))
	assert.True(t, isAllowedBootDiskType("pd-standard"))
	assert.False(t, isAllowedBootDiskType("local-ssd"))
	assert.False(t, isAllowedBootDiskType(""))
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
