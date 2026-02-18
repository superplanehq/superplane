package createvm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	compute "google.golang.org/api/compute/v1"
)

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
