package azure

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

func TestStartVMComponent_Metadata(t *testing.T) {
	component := &StartVMComponent{}

	assert.Equal(t, "azure.startVirtualMachine", component.Name())
	assert.Equal(t, "Start Virtual Machine", component.Label())
	assert.Equal(t, "azure", component.Icon())
	assert.Equal(t, "blue", component.Color())
	assert.NotEmpty(t, component.Description())
	assert.NotEmpty(t, component.Documentation())
}

func TestStartVMComponent_Configuration(t *testing.T) {
	component := &StartVMComponent{}
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

func TestStartVMComponent_ExampleOutput(t *testing.T) {
	component := &StartVMComponent{}
	example := component.ExampleOutput()

	require.NotNil(t, example)
	assert.Contains(t, example, "id")
	assert.Contains(t, example, "name")
	assert.Contains(t, example, "resourceGroup")
}

func TestStartVMComponent_OutputChannels(t *testing.T) {
	component := &StartVMComponent{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func TestStartVMComponent_Actions(t *testing.T) {
	component := &StartVMComponent{}
	actions := component.Actions()
	assert.Empty(t, actions)
}

func TestStartVMComponent_HandleAction(t *testing.T) {
	component := &StartVMComponent{}
	ctx := core.ActionContext{
		Name:   "test",
		Logger: logrus.NewEntry(logrus.New()),
	}
	err := component.HandleAction(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no actions defined")
}

func TestStartVMComponent_Setup_Valid(t *testing.T) {
	component := &StartVMComponent{}
	ctx := newSetupContext(map[string]any{
		"resourceGroup": "my-rg",
		"name":          "my-vm",
	})
	err := component.Setup(ctx)
	assert.NoError(t, err)
}

func TestStartVMComponent_Setup_MissingResourceGroup(t *testing.T) {
	component := &StartVMComponent{}
	ctx := newSetupContext(map[string]any{
		"name": "my-vm",
	})
	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resource group is required")
}

func TestStartVMComponent_Setup_MissingName(t *testing.T) {
	component := &StartVMComponent{}
	ctx := newSetupContext(map[string]any{
		"resourceGroup": "my-rg",
	})
	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VM name is required")
}
