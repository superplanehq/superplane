package core

import (
	"github.com/superplanehq/superplane/pkg/configuration"
)

/*
 * Widgets are used to represent every node not event or execution related.
 * Use it to display and group data in the UI canvas.
 */
type Widget interface {

	/*
	 * The unique identifier for the widget.
	 * This is how nodes reference it, and is used for registration.
	 */
	Name() string

	/*
	 * The label for the widget.
	 * This is how nodes are displayed in the UI.
	 */
	Label() string

	/*
	 * A good description of what the widget does.
	 * Helpful for documentation and user interfaces.
	 */
	Description() string

	/*
	 * The icon for the widget.
	 */
	Icon() string

	/*
	 * The color for the widget.
	 */
	Color() string

	/*
	 * The configuration fields exposed by the widget.
	 */
	Configuration() []configuration.Field
}
