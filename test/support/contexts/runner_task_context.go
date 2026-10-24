package contexts

import "github.com/superplanehq/superplane/pkg/core"

// RunnerTaskContext captures task operations without an external runner service.
type RunnerTaskContext struct {
	CreatedID  string
	FleetID    string
	Payload    []byte
	Task       *core.RunnerTask
	CanceledID string
	CreateErr  error
	FindErr    error
	CancelErr  error
}

func (c *RunnerTaskContext) Create(id, fleetID string, payload []byte) error {
	c.CreatedID = id
	c.FleetID = fleetID
	c.Payload = append([]byte(nil), payload...)
	return c.CreateErr
}
func (c *RunnerTaskContext) Find(string) (*core.RunnerTask, error) { return c.Task, c.FindErr }
func (c *RunnerTaskContext) RequestCancel(id string) error         { c.CanceledID = id; return c.CancelErr }

var _ core.RunnerTaskContext = (*RunnerTaskContext)(nil)
