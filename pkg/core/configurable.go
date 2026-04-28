package core

import "github.com/superplanehq/superplane/pkg/configuration"

/*
 * Configurable is a component (action, trigger, widget) that can be configured.
 */
type Configurable interface {
	Configuration() []configuration.Field
}
