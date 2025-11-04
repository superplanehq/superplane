package github

import (
	"github.com/superplanehq/superplane/pkg/components"
	internal "github.com/superplanehq/superplane/pkg/integrations/github/components"
)

func Components() []components.Component {
	return []components.Component{
		&internal.ListRepositories{},
		&internal.RunWorkflow{},
	}
}
