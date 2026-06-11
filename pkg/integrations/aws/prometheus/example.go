package prometheus

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_workspace.json
var exampleOutputCreateWorkspaceBytes []byte

var exampleOutputCreateWorkspaceOnce sync.Once
var exampleOutputCreateWorkspace map[string]any

//go:embed example_output_get_workspace.json
var exampleOutputGetWorkspaceBytes []byte

var exampleOutputGetWorkspaceOnce sync.Once
var exampleOutputGetWorkspace map[string]any

//go:embed example_output_update_workspace.json
var exampleOutputUpdateWorkspaceBytes []byte

var exampleOutputUpdateWorkspaceOnce sync.Once
var exampleOutputUpdateWorkspace map[string]any

//go:embed example_output_delete_workspace.json
var exampleOutputDeleteWorkspaceBytes []byte

var exampleOutputDeleteWorkspaceOnce sync.Once
var exampleOutputDeleteWorkspace map[string]any

//go:embed example_output_create_rule_group_namespace.json
var exampleOutputCreateRuleGroupNamespaceBytes []byte

var exampleOutputCreateRuleGroupNamespaceOnce sync.Once
var exampleOutputCreateRuleGroupNamespace map[string]any

//go:embed example_output_get_rule_group_namespace.json
var exampleOutputGetRuleGroupNamespaceBytes []byte

var exampleOutputGetRuleGroupNamespaceOnce sync.Once
var exampleOutputGetRuleGroupNamespace map[string]any

//go:embed example_output_update_rule_group_namespace.json
var exampleOutputUpdateRuleGroupNamespaceBytes []byte

var exampleOutputUpdateRuleGroupNamespaceOnce sync.Once
var exampleOutputUpdateRuleGroupNamespace map[string]any

//go:embed example_output_delete_rule_group_namespace.json
var exampleOutputDeleteRuleGroupNamespaceBytes []byte

var exampleOutputDeleteRuleGroupNamespaceOnce sync.Once
var exampleOutputDeleteRuleGroupNamespace map[string]any

func (c *CreateWorkspace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateWorkspaceOnce,
		exampleOutputCreateWorkspaceBytes,
		&exampleOutputCreateWorkspace,
	)
}

func (c *GetWorkspace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetWorkspaceOnce,
		exampleOutputGetWorkspaceBytes,
		&exampleOutputGetWorkspace,
	)
}

func (c *UpdateWorkspace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpdateWorkspaceOnce,
		exampleOutputUpdateWorkspaceBytes,
		&exampleOutputUpdateWorkspace,
	)
}

func (c *DeleteWorkspace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeleteWorkspaceOnce,
		exampleOutputDeleteWorkspaceBytes,
		&exampleOutputDeleteWorkspace,
	)
}

func (c *CreateRuleGroupNamespace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateRuleGroupNamespaceOnce,
		exampleOutputCreateRuleGroupNamespaceBytes,
		&exampleOutputCreateRuleGroupNamespace,
	)
}

func (c *GetRuleGroupNamespace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetRuleGroupNamespaceOnce,
		exampleOutputGetRuleGroupNamespaceBytes,
		&exampleOutputGetRuleGroupNamespace,
	)
}

func (c *UpdateRuleGroupNamespace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpdateRuleGroupNamespaceOnce,
		exampleOutputUpdateRuleGroupNamespaceBytes,
		&exampleOutputUpdateRuleGroupNamespace,
	)
}

func (c *DeleteRuleGroupNamespace) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeleteRuleGroupNamespaceOnce,
		exampleOutputDeleteRuleGroupNamespaceBytes,
		&exampleOutputDeleteRuleGroupNamespace,
	)
}
