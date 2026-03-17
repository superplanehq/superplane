package azure

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func newSetupContext(config map[string]any) core.SetupContext {
	return core.SetupContext{
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: config,
		Metadata:      &contexts.MetadataContext{},
	}
}

func TestDeleteVMComponent_Metadata(t *testing.T) {
	component := &DeleteVMComponent{}

	assert.Equal(t, "azure.deleteVirtualMachine", component.Name())
	assert.Equal(t, "Delete Virtual Machine", component.Label())
	assert.Equal(t, "azure", component.Icon())
	assert.Equal(t, "blue", component.Color())
	assert.NotEmpty(t, component.Description())
	assert.NotEmpty(t, component.Documentation())
	assert.Contains(t, component.Description(), "Deletes")
}

func TestDeleteVMComponent_Configuration(t *testing.T) {
	component := &DeleteVMComponent{}
	fields := component.Configuration()

	require.Len(t, fields, 2, "Should have exactly 2 configuration fields")

	resourceGroupField := fields[0]
	assert.Equal(t, "resourceGroup", resourceGroupField.Name)
	assert.Equal(t, "Resource Group", resourceGroupField.Label)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, resourceGroupField.Type)
	assert.True(t, resourceGroupField.Required)
	assert.NotEmpty(t, resourceGroupField.Description)

	nameField := fields[1]
	assert.Equal(t, "name", nameField.Name)
	assert.Equal(t, "VM Name", nameField.Label)
	assert.Equal(t, configuration.FieldTypeString, nameField.Type)
	assert.True(t, nameField.Required)
	assert.NotEmpty(t, nameField.Description)
}

func TestDeleteVMComponent_ExampleOutput(t *testing.T) {
	component := &DeleteVMComponent{}
	example := component.ExampleOutput()

	require.NotNil(t, example)
	assert.Contains(t, example, "id")
	assert.Contains(t, example, "name")
	assert.Contains(t, example, "resourceGroup")
	assert.Equal(t, "my-vm", example["name"])
	assert.Equal(t, "my-rg", example["resourceGroup"])
}

func TestDeleteVMComponent_OutputChannels(t *testing.T) {
	component := &DeleteVMComponent{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func TestDeleteVMComponent_Actions(t *testing.T) {
	component := &DeleteVMComponent{}
	actions := component.Actions()

	assert.NotNil(t, actions)
	assert.Empty(t, actions)
}

func TestDeleteVMComponent_HandleAction(t *testing.T) {
	component := &DeleteVMComponent{}

	ctx := core.ActionContext{
		Name:   "test",
		Logger: logrus.NewEntry(logrus.New()),
	}

	err := component.HandleAction(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no actions defined")
}

func TestDeleteVMComponent_Setup_Valid(t *testing.T) {
	component := &DeleteVMComponent{}

	ctx := newSetupContext(map[string]any{
		"resourceGroup": "my-rg",
		"name":          "my-vm",
	})

	err := component.Setup(ctx)
	assert.NoError(t, err)
}

func TestDeleteVMComponent_Setup_MissingResourceGroup(t *testing.T) {
	component := &DeleteVMComponent{}

	ctx := newSetupContext(map[string]any{
		"name": "my-vm",
	})

	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resource group is required")
}

func TestDeleteVMComponent_Setup_MissingName(t *testing.T) {
	component := &DeleteVMComponent{}

	ctx := newSetupContext(map[string]any{
		"resourceGroup": "my-rg",
	})

	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VM name is required")
}
