package semaphore

import (
	"github.com/superplanehq/superplane/pkg/components"
	internal "github.com/superplanehq/superplane/pkg/integrations/semaphore/components"
)

func Components() []components.Component {
	return []components.Component{
		&internal.RunWorkflow{},
		&internal.ListProjects{},
	}
}
