package group

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterWidget("group", &Group{})
}

type Group struct{}

func (g *Group) Name() string {
	return "group"
}

func (g *Group) Label() string {
	return "Group"
}

func (g *Group) Description() string {
	return "Visually group related nodes together on the canvas"
}

func (g *Group) Icon() string {
	return "group"
}

func (g *Group) Color() string {
	return "purple"
}

func (g *Group) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "label",
			Label:       "Label",
			Type:        configuration.FieldTypeString,
			Description: "Display label for the group",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Description: "Optional description shown below the label",
		},
		{
			Name:        "color",
			Label:       "Color",
			Type:        configuration.FieldTypeSelect,
			Description: "Color theme for the group",
			Default:     "purple",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Purple", Value: "purple"},
						{Label: "Blue", Value: "blue"},
						{Label: "Green", Value: "green"},
						{Label: "Cyan", Value: "cyan"},
						{Label: "Orange", Value: "orange"},
						{Label: "Rose", Value: "rose"},
						{Label: "Amber", Value: "amber"},
					},
				},
			},
		},
		{
			Name:        "childNodeIds",
			Label:       "Child Nodes",
			Type:        configuration.FieldTypeList,
			Description: "IDs of nodes contained within this group",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}
