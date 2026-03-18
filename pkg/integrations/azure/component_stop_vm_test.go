package azure

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

func TestStopVMComponent_Metadata(t *testing.T) {
	component := &StopVMComponent{}

	assert.Equal(t, "azure.stopVirtualMachine", component.Name())
	assert.Equal(t, "Stop Virtual Machine", component.Label())
	assert.Equal(t, "azure", component.Icon())
	assert.Equal(t, "blue", component.Color())
	assert.NotEmpty(t, component.Description())
	assert.NotEmpty(t, component.Documentation())
}

func TestStopVMComponent_Configuration(t *testing.T) {
	component := &StopVMComponent{}
	fields := component.Configuration()

	require.Len(t, fields, 2)

	resourceGroupField := fields[0]
	assert.Equal(t, "resourceGroup", resourceGroupField.Name)
	assert.Equal(t, "Resource Group", resourceGroupField.Label)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, resourceGroupField.Type)
	assert.True(t, resourceGroupField.Required)

	nameField := fields[1]
	assert.Equal(t, "name", nameField.Name)
	assert.Equal(t, "VM Name", nameField.Label)
	assert.Equal(t, configuration.FieldTypeString, nameField.Type)
	assert.True(t, nameField.Required)
}

func TestStopVMComponent_ExampleOutput(t *testing.T) {
	component := &StopVMComponent{}
	example := component.ExampleOutput()

	require.NotNil(t, example)
	assert.Contains(t, example, "id")
	assert.Contains(t, example, "name")
	assert.Contains(t, example, "resourceGroup")
}

func TestStopVMComponent_OutputChannels(t *testing.T) {
	component := &StopVMComponent{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func TestStopVMComponent_Actions(t *testing.T) {
	component := &StopVMComponent{}
	actions := component.Actions()
	assert.Empty(t, actions)
}

func TestStopVMComponent_HandleAction(t *testing.T) {
	component := &StopVMComponent{}
	ctx := core.ActionContext{
		Name:   "test",
		Logger: logrus.NewEntry(logrus.New()),
	}
	err := component.HandleAction(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no actions defined")
}

func TestStopVMComponent_Setup_Valid(t *testing.T) {
	component := &StopVMComponent{}
	ctx := newSetupContext(map[string]any{
		"resourceGroup": "my-rg",
		"name":          "my-vm",
	})
	err := component.Setup(ctx)
	assert.NoError(t, err)
}

func TestStopVMComponent_Setup_MissingResourceGroup(t *testing.T) {
	component := &StopVMComponent{}
	ctx := newSetupContext(map[string]any{
		"name": "my-vm",
	})
	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resource group is required")
}

func TestStopVMComponent_Setup_MissingName(t *testing.T) {
	component := &StopVMComponent{}
	ctx := newSetupContext(map[string]any{
		"resourceGroup": "my-rg",
	})
	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VM name is required")
}
