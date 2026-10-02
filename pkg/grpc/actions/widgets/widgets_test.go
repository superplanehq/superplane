package widgets

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

type fakeWidget struct {
	name         string
	label        string
	description  string
	icon         string
	color        string
	configFields []configuration.Field
}

func (f fakeWidget) Name() string                         { return f.name }
func (f fakeWidget) Label() string                        { return f.label }
func (f fakeWidget) Description() string                  { return f.description }
func (f fakeWidget) Icon() string                         { return f.icon }
func (f fakeWidget) Color() string                        { return f.color }
func (f fakeWidget) Configuration() []configuration.Field { return f.configFields }

func TestDescribeWidget(t *testing.T) {
	reg := &registry.Registry{
		Widgets: map[string]core.Widget{
			"test-widget": fakeWidget{
				name:         "test-widget",
				label:        "Test Label",
				description:  "Test Description",
				icon:         "test-icon",
				color:        "#123456",
				configFields: []configuration.Field{{Name: "limit"}},
			},
		},
	}

	response, err := DescribeWidget(context.Background(), reg, "test-widget")
	require.NoError(t, err)

	widget := response.Widget
	require.Equal(t, "test-widget", widget.Name)
	require.Equal(t, "Test Label", widget.Label)
	require.Equal(t, "Test Description", widget.Description)
	require.Equal(t, "test-icon", widget.Icon)
	require.Equal(t, "#123456", widget.Color)
	require.Len(t, widget.Configuration, 1)
	require.Equal(t, "limit", widget.Configuration[0].Name)
}

func TestDescribeWidgetUnknownName(t *testing.T) {
	reg := &registry.Registry{
		Widgets: map[string]core.Widget{},
	}

	_, err := DescribeWidget(context.Background(), reg, "missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not registered")
}

func TestListWidgets(t *testing.T) {
	reg := &registry.Registry{
		Widgets: map[string]core.Widget{
			"beta":  fakeWidget{name: "beta"},
			"alpha": fakeWidget{name: "alpha"},
		},
	}

	response, err := ListWidgets(context.Background(), reg)
	require.NoError(t, err)
	require.Len(t, response.Widgets, 2)
	require.Equal(t, "alpha", response.Widgets[0].Name)
	require.Equal(t, "beta", response.Widgets[1].Name)
}
