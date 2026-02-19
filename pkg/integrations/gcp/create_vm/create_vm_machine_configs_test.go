package createvm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_lastSegment(t *testing.T) {
	assert.Equal(t, "", lastSegment(""))
	assert.Equal(t, "e2-medium", lastSegment("zones/us-central1-a/machineTypes/e2-medium"))
	assert.Equal(t, "name", lastSegment("a/b/c/name"))
	assert.Equal(t, "only", lastSegment("only"))
}

func Test_DeriveFamily(t *testing.T) {
	assert.Equal(t, "E2", DeriveFamily("e2-medium"))
	assert.Equal(t, "N2", DeriveFamily("n2-standard-4"))
	assert.Equal(t, "C2", DeriveFamily("c2-standard-8"))
	assert.Equal(t, "", DeriveFamily(""))
	assert.Equal(t, "", DeriveFamily("   "))
	assert.Equal(t, "N2", DeriveFamily("  n2-standard-4  "))
}

func Test_zoneToRegion(t *testing.T) {
	assert.Equal(t, "us-central1", zoneToRegion("us-central1-a"))
	assert.Equal(t, "europe-west1", zoneToRegion("europe-west1-b"))
	assert.Equal(t, "", zoneToRegion(""))
	assert.Equal(t, "", zoneToRegion("   "))
	assert.Equal(t, "us-east1", zoneToRegion("us-east1-c"))
	t.Run("no hyphen returns zone as-is", func(t *testing.T) {
		assert.Equal(t, "region", zoneToRegion("region"))
	})
}

func Test_FormatMachineTypeSummary(t *testing.T) {
	assert.Equal(t, "", FormatMachineTypeSummary(nil))
	t.Run("formats vCPU and memory", func(t *testing.T) {
		mt := &MachineType{GuestCPUs: 4, MemoryMB: 16384}
		assert.Equal(t, "4 vCPU, 16 GB memory", FormatMachineTypeSummary(mt))
	})
	t.Run("small memory rounds to 1 GB", func(t *testing.T) {
		mt := &MachineType{GuestCPUs: 1, MemoryMB: 512}
		assert.Equal(t, "1 vCPU, 1 GB memory", FormatMachineTypeSummary(mt))
	})
	t.Run("thousands with commas", func(t *testing.T) {
		mt := &MachineType{GuestCPUs: 288, MemoryMB: 1105920} // 1080 GB
		assert.Equal(t, "288 vCPU, 1,080 GB memory", FormatMachineTypeSummary(mt))
	})
}
