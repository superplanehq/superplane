package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRestartVMComponent_Setup_Valid(t *testing.T) {
	component := &RestartVMComponent{}
	ctx := newSetupContext(map[string]any{
		"resourceGroup": "my-rg",
		"name":          "my-vm",
	})
	err := component.Setup(ctx)
	assert.NoError(t, err)
}

func TestRestartVMComponent_Setup_MissingResourceGroup(t *testing.T) {
	component := &RestartVMComponent{}
	ctx := newSetupContext(map[string]any{
		"name": "my-vm",
	})
	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resource group is required")
}

func TestRestartVMComponent_Setup_MissingName(t *testing.T) {
	component := &RestartVMComponent{}
	ctx := newSetupContext(map[string]any{
		"resourceGroup": "my-rg",
	})
	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VM name is required")
}
