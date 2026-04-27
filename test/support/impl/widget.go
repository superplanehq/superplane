package impl

import (
	"github.com/superplanehq/superplane/pkg/configuration"
)

type DummyWidgetOptions struct {
	Name string
}

type DummyWidget struct {
	name string
}

func NewDummyWidget(options DummyWidgetOptions) *DummyWidget {
	name := options.Name
	if name == "" {
		name = "dummy"
	}

	return &DummyWidget{
		name: name,
	}
}

func (w *DummyWidget) Name() string {
	return w.name
}

func (w *DummyWidget) Label() string {
	return "dummy"
}

func (w *DummyWidget) Description() string {
	return "Just a dummy widget used in unit tests"
}

func (w *DummyWidget) Icon() string {
	return "dummy"
}

func (w *DummyWidget) Color() string {
	return "dummy"
}

func (w *DummyWidget) Configuration() []configuration.Field {
	return nil
}
