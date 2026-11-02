// Package fleetapi selects an explicit API scope and normalizes its presentation.
package fleetapi

import (
	"github.com/superplanehq/superplane/pkg/cli/core"
	sdk "github.com/superplanehq/superplane/pkg/openapi_client"
)

type Client struct {
	ctx            core.CommandContext
	installation   bool
	organizationID string
}

func New(ctx core.CommandContext) (*Client, error) {
	installation, _ := ctx.Cmd.Flags().GetBool("installation")
	c := &Client{ctx: ctx, installation: installation}
	if installation {
		return c, nil
	}
	if config, ok := ctx.Config.(interface{ GetOrganizationID() string }); ok {
		c.organizationID = config.GetOrganizationID()
	}
	if c.organizationID != "" {
		return c, nil
	}
	id, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return nil, err
	}
	c.organizationID = id
	return c, nil
}

func (c *Client) ListFleets() (*sdk.RunnersListFleetsResponse, error) {
	if c.installation {
		out, _, err := c.ctx.API.RunnersAPI.RunnersListFleets(c.ctx.Context).Execute()
		return out, err
	}
	out, _, err := c.ctx.API.OrganizationsAPI.OrganizationsListManagedRunnerFleets(c.ctx.Context, c.organizationID).Execute()
	if err != nil {
		return nil, err
	}
	result := &sdk.RunnersListFleetsResponse{Fleets: []sdk.AdminRunnersFleet{}}
	for _, fleet := range out.GetFleets() {
		result.Fleets = append(result.Fleets, *fleetPresentation(&fleet))
	}
	return result, nil
}

func (c *Client) DescribeFleet(id string) (*sdk.RunnersDescribeFleetResponse, error) {
	if c.installation {
		out, _, err := c.ctx.API.RunnersAPI.RunnersDescribeFleet(c.ctx.Context, id).Execute()
		return out, err
	}
	out, _, err := c.ctx.API.OrganizationsAPI.OrganizationsDescribeRunnerFleet(c.ctx.Context, c.organizationID, id).Execute()
	if err != nil {
		return nil, err
	}
	return &sdk.RunnersDescribeFleetResponse{Fleet: fleetPresentation(out.Fleet)}, nil
}

func (c *Client) Capacity(id string) (*sdk.RunnersGetFleetCapacityResponse, error) {
	if c.installation {
		out, _, err := c.ctx.API.RunnersAPI.RunnersGetFleetCapacity(c.ctx.Context, id).Execute()
		return out, err
	}
	out, _, err := c.ctx.API.OrganizationsAPI.OrganizationsGetRunnerFleetCapacity(c.ctx.Context, c.organizationID, id).Execute()
	return (*sdk.RunnersGetFleetCapacityResponse)(out), err
}

func (c *Client) CreateFleet(body sdk.RunnersCreateFleetRequest) (*sdk.RunnersCreateFleetResponse, error) {
	if c.installation {
		out, _, err := c.ctx.API.RunnersAPI.RunnersCreateFleet(c.ctx.Context).Body(body).Execute()
		return out, err
	}
	request := sdk.OrganizationsCreateRunnerFleetBody{FleetId: body.FleetId, RunnerVersion: body.RunnerVersion,
		Enabled: body.Enabled, Spec: organizationSpec(body.Spec)}
	out, _, err := c.ctx.API.OrganizationsAPI.OrganizationsCreateRunnerFleet(c.ctx.Context, c.organizationID).Body(request).Execute()
	if err != nil {
		return nil, err
	}
	return &sdk.RunnersCreateFleetResponse{Fleet: fleetPresentation(out.Fleet)}, nil
}

func (c *Client) UpdateFleet(id string, body sdk.RunnersUpdateFleetBody) (*sdk.RunnersUpdateFleetResponse, error) {
	if c.installation {
		out, _, err := c.ctx.API.RunnersAPI.RunnersUpdateFleet(c.ctx.Context, id).Body(body).Execute()
		return out, err
	}
	request := sdk.OrganizationsUpdateRunnerFleetBody{RunnerVersion: body.RunnerVersion, Enabled: body.Enabled, Spec: organizationSpec(body.Spec)}
	out, _, err := c.ctx.API.OrganizationsAPI.OrganizationsUpdateRunnerFleet(c.ctx.Context, c.organizationID, id).Body(request).Execute()
	if err != nil {
		return nil, err
	}
	return &sdk.RunnersUpdateFleetResponse{Fleet: fleetPresentation(out.Fleet)}, nil
}

func (c *Client) CreateRunner(fleet string, body sdk.RunnersCreateRunnerBody) (*sdk.RunnersCreateRunnerResponse, error) {
	if c.installation {
		out, _, err := c.ctx.API.RunnersAPI.RunnersCreateRunner(c.ctx.Context, fleet).Body(body).Execute()
		return out, err
	}
	request := sdk.OrganizationsCreateFleetRunnerBody{FleetId: &fleet, TaskId: body.TaskId, IdempotencyKey: body.IdempotencyKey,
		Ephemeral: body.Ephemeral, FleetManagerId: body.FleetManagerId, Provider: body.Provider}
	out, _, err := c.ctx.API.OrganizationsAPI.OrganizationsCreateFleetRunner(c.ctx.Context, c.organizationID).Body(request).Execute()
	if err != nil {
		return nil, err
	}
	return &sdk.RunnersCreateRunnerResponse{Runner: (*sdk.AdminRunnersRunner)(out.Runner), RegistrationToken: out.RegistrationToken,
		RegistrationExpiresAt: out.RegistrationExpiresAt, RunnerApiUrl: out.RunnerApiUrl, DisplayName: out.DisplayName}, nil
}

func (c *Client) ListRunners(fleet string, states []string, limit int32, after string) (*sdk.RunnersListRunnersResponse, error) {
	if c.installation {
		request := c.ctx.API.RunnersAPI.RunnersListRunners(c.ctx.Context, fleet).Limit(limit)
		if len(states) > 0 {
			request = request.States(states)
		}
		if after != "" {
			request = request.AfterId(after)
		}
		out, _, err := request.Execute()
		return out, err
	}
	request := c.ctx.API.OrganizationsAPI.OrganizationsListFleetRunners(c.ctx.Context, c.organizationID).FleetId(fleet).Limit(limit)
	if len(states) > 0 {
		request = request.States(states)
	}
	if after != "" {
		request = request.AfterId(after)
	}
	out, _, err := request.Execute()
	if err != nil {
		return nil, err
	}
	result := &sdk.RunnersListRunnersResponse{Runners: []sdk.AdminRunnersRunner{}, TotalCount: out.TotalCount, HasNextPage: out.HasNextPage}
	for _, runner := range out.GetRunners() {
		result.Runners = append(result.Runners, sdk.AdminRunnersRunner(runner))
	}
	return result, nil
}

func (c *Client) DescribeRunner(fleet, id string) (*sdk.RunnersDescribeRunnerResponse, error) {
	if c.installation {
		out, _, err := c.ctx.API.RunnersAPI.RunnersDescribeRunner(c.ctx.Context, fleet, id).Execute()
		return out, err
	}
	out, _, err := c.ctx.API.OrganizationsAPI.OrganizationsDescribeFleetRunner(c.ctx.Context, c.organizationID, id).FleetId(fleet).Execute()
	if err != nil {
		return nil, err
	}
	return &sdk.RunnersDescribeRunnerResponse{Runner: (*sdk.AdminRunnersRunner)(out.Runner)}, nil
}

func (c *Client) DeleteRunner(fleet, id string, states []string) (*sdk.RunnersDeleteRunnerResponse, error) {
	if c.installation {
		request := c.ctx.API.RunnersAPI.RunnersDeleteRunner(c.ctx.Context, fleet, id)
		if len(states) > 0 {
			request = request.States(states)
		}
		out, _, err := request.Execute()
		return out, err
	}
	request := c.ctx.API.OrganizationsAPI.OrganizationsTerminateFleetRunner(c.ctx.Context, c.organizationID, id).FleetId(fleet)
	if len(states) > 0 {
		request = request.States(states)
	}
	out, _, err := request.Execute()
	if err != nil {
		return nil, err
	}
	return &sdk.RunnersDeleteRunnerResponse{Runner: (*sdk.AdminRunnersRunner)(out.Runner)}, nil
}

func (c *Client) ListTasks(fleet string, states []string, limit int32, after string) (*sdk.RunnersListFleetTasksResponse, error) {
	if c.installation {
		request := c.ctx.API.RunnersAPI.RunnersListFleetTasks(c.ctx.Context, fleet).Limit(limit)
		if len(states) > 0 {
			request = request.States(states)
		}
		if after != "" {
			request = request.AfterId(after)
		}
		out, _, err := request.Execute()
		return out, err
	}
	request := c.ctx.API.OrganizationsAPI.OrganizationsListRunnerFleetTasks(c.ctx.Context, c.organizationID, fleet).Limit(limit)
	if len(states) > 0 {
		request = request.States(states)
	}
	if after != "" {
		request = request.AfterId(after)
	}
	out, _, err := request.Execute()
	if err != nil {
		return nil, err
	}
	result := &sdk.RunnersListFleetTasksResponse{Tasks: []sdk.RunnersTask{}, TotalCount: out.TotalCount, HasNextPage: out.HasNextPage}
	for _, task := range out.GetTasks() {
		result.Tasks = append(result.Tasks, sdk.RunnersTask(task))
	}
	return result, nil
}

func organizationSpec(spec *sdk.RunnersFleetSpec) *sdk.OrganizationsOrganizationRunnerFleetSpec {
	return (*sdk.OrganizationsOrganizationRunnerFleetSpec)(spec)
}

func fleetPresentation(fleet *sdk.SuperplaneOrganizationsFleet) *sdk.AdminRunnersFleet {
	if fleet == nil {
		return nil
	}
	return &sdk.AdminRunnersFleet{Id: fleet.Id, Enabled: fleet.Enabled, RunnerVersion: fleet.RunnerVersion,
		Spec: (*sdk.RunnersFleetSpec)(fleet.Spec), CreatedAt: fleet.CreatedAt, UpdatedAt: fleet.UpdatedAt}
}
