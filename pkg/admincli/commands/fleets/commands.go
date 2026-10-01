package fleets

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type listCommand struct{}

func newListCommand(options core.BindOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "list", Short: "List fleets", Args: cobra.NoArgs}
	core.Bind(cmd, &listCommand{}, options)
	return cmd
}

func (c *listCommand) Execute(ctx core.CommandContext) error {
	response, _, err := ctx.API.RunnersAPI.RunnersListFleets(ctx.Context).Execute()
	if err != nil {
		return core.FormatCommandError(err)
	}
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		if _, err := fmt.Fprintln(writer, "ID\tENABLED\tRUNNER_VERSION\tOS\tARCH\tCPU\tMEMORY\tDISK"); err != nil {
			return err
		}
		for _, fleet := range response.GetFleets() {
			spec := fleet.GetSpec()
			if _, err := fmt.Fprintf(
				writer,
				"%s\t%t\t%s\t%s\t%s\t%s\t%s\t%s\n",
				fleet.GetId(),
				fleet.GetEnabled(),
				valueOrDash(fleet.GetRunnerVersion()),
				valueOrDash(spec.GetOperatingSystem()),
				valueOrDash(spec.GetArchitecture()),
				formatCPU(spec.GetCpuMillicores()),
				formatMemory(spec.GetMemoryMb()),
				formatDisk(spec.GetDiskGb()),
			); err != nil {
				return err
			}
		}
		return writer.Flush()
	})
}

type describeCommand struct {
	FleetID string
}

func newDescribeCommand(options core.BindOptions) *cobra.Command {
	handler := &describeCommand{}
	cmd := &cobra.Command{
		Use:   "describe <fleet-id>",
		Short: "Describe a fleet and its current capacity",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			handler.FleetID = args[0]
		},
	}
	core.Bind(cmd, handler, options)
	return cmd
}

func (c *describeCommand) Execute(ctx core.CommandContext) error {
	fleetResponse, _, err := ctx.API.RunnersAPI.RunnersDescribeFleet(ctx.Context, c.FleetID).Execute()
	if err != nil {
		return core.FormatCommandError(err)
	}

	capacity, _, err := ctx.API.RunnersAPI.RunnersGetFleetCapacity(ctx.Context, c.FleetID).Execute()
	if err != nil {
		return core.FormatCommandError(err)
	}

	result, err := newFleetDescription(fleetResponse.Fleet, capacity)
	if err != nil {
		return err
	}
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(result)
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderFleetDescription(stdout, result)
	})
}

type fleetDescription struct {
	FleetID       string                                          `json:"fleetId" yaml:"fleetId"`
	RunnerVersion string                                          `json:"runnerVersion" yaml:"runnerVersion"`
	Enabled       bool                                            `json:"enabled" yaml:"enabled"`
	Spec          openapi_client.RunnersFleetSpec                 `json:"spec" yaml:"spec"`
	CreatedAt     *time.Time                                      `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt     *time.Time                                      `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
	Capacity      *openapi_client.RunnersGetFleetCapacityResponse `json:"capacity,omitempty" yaml:"capacity,omitempty"`
}

func newFleetDescription(
	fleet *openapi_client.RunnersFleet,
	capacity *openapi_client.RunnersGetFleetCapacityResponse,
) (fleetDescription, error) {
	if fleet == nil {
		return fleetDescription{}, fmt.Errorf("server returned an empty fleet")
	}
	return fleetDescription{
		FleetID:       fleet.GetId(),
		RunnerVersion: fleet.GetRunnerVersion(),
		Enabled:       fleet.GetEnabled(),
		Spec:          fleet.GetSpec(),
		CreatedAt:     fleet.CreatedAt,
		UpdatedAt:     fleet.UpdatedAt,
		Capacity:      capacity,
	}, nil
}

type fileCommand struct {
	File string
}

func addFileFlag(cmd *cobra.Command, handler *fileCommand) {
	cmd.Flags().StringVarP(&handler.File, "file", "f", "", "fleet definition file, or - for stdin")
	_ = cmd.MarkFlagRequired("file")
}

type createCommand struct {
	fileCommand
}

func newCreateCommand(options core.BindOptions) *cobra.Command {
	handler := &createCommand{}
	cmd := &cobra.Command{Use: "create", Short: "Create a fleet", Args: cobra.NoArgs}
	addFileFlag(cmd, &handler.fileCommand)
	core.Bind(cmd, handler, options)
	return cmd
}

func (c *createCommand) Execute(ctx core.CommandContext) error {
	var request openapi_client.RunnersCreateFleetRequest
	if err := decodeFleetFile(ctx, c.File, &request); err != nil {
		return err
	}
	if strings.TrimSpace(request.GetFleetId()) == "" {
		return fmt.Errorf("fleetId is required")
	}
	if request.Spec == nil {
		return fmt.Errorf("spec is required")
	}
	if strings.TrimSpace(request.GetRunnerVersion()) == "" {
		return fmt.Errorf("runnerVersion is required")
	}

	response, _, err := ctx.API.RunnersAPI.RunnersCreateFleet(ctx.Context).Body(request).Execute()
	if err != nil {
		return core.FormatCommandError(err)
	}
	return renderFleetResponse(ctx, response.Fleet)
}

type updateCommand struct {
	fileCommand
}

type updateFile struct {
	FleetID       string                                          `json:"fleetId"`
	RunnerVersion *string                                         `json:"runnerVersion,omitempty"`
	Enabled       *bool                                           `json:"enabled,omitempty"`
	Spec          *openapi_client.RunnersFleetSpec                `json:"spec,omitempty"`
	CreatedAt     *time.Time                                      `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time                                      `json:"updatedAt,omitempty"`
	Capacity      *openapi_client.RunnersGetFleetCapacityResponse `json:"capacity,omitempty"`
}

func newUpdateCommand(options core.BindOptions) *cobra.Command {
	handler := &updateCommand{}
	cmd := &cobra.Command{Use: "update", Short: "Update a fleet", Args: cobra.NoArgs}
	addFileFlag(cmd, &handler.fileCommand)
	core.Bind(cmd, handler, options)
	return cmd
}

func (c *updateCommand) Execute(ctx core.CommandContext) error {
	var file updateFile
	if err := decodeFleetFile(ctx, c.File, &file); err != nil {
		return err
	}
	if strings.TrimSpace(file.FleetID) == "" {
		return fmt.Errorf("fleetId is required")
	}
	if file.RunnerVersion == nil && file.Enabled == nil && file.Spec == nil {
		return fmt.Errorf("at least one of runnerVersion, enabled, or spec is required")
	}

	request := openapi_client.RunnersUpdateFleetBody{
		RunnerVersion: file.RunnerVersion,
		Enabled:       file.Enabled,
		Spec:          file.Spec,
	}
	response, _, err := ctx.API.RunnersAPI.RunnersUpdateFleet(ctx.Context, file.FleetID).Body(request).Execute()
	if err != nil {
		return core.FormatCommandError(err)
	}
	return renderFleetResponse(ctx, response.Fleet)
}

func decodeFleetFile(ctx core.CommandContext, path string, target any) error {
	var (
		data []byte
		err  error
	)
	if path == "-" {
		data, err = io.ReadAll(ctx.Cmd.InOrStdin())
	} else {
		// #nosec G304 -- the user explicitly selects the fleet definition file.
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return fmt.Errorf("read fleet definition: %w", err)
	}
	if err := core.NewDecoder(data).DecodeYAML(target); err != nil {
		return fmt.Errorf("decode fleet definition: %w", err)
	}
	return nil
}

func renderFleetResponse(ctx core.CommandContext, fleet *openapi_client.RunnersFleet) error {
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(fleet)
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderFleet(stdout, fleet)
	})
}

func renderFleetDescription(stdout io.Writer, description fleetDescription) error {
	fleet := &openapi_client.RunnersFleet{
		Id:            &description.FleetID,
		Enabled:       &description.Enabled,
		Spec:          &description.Spec,
		RunnerVersion: &description.RunnerVersion,
		CreatedAt:     description.CreatedAt,
		UpdatedAt:     description.UpdatedAt,
	}
	if err := renderFleet(stdout, fleet); err != nil {
		return err
	}
	capacity := description.Capacity
	_, err := fmt.Fprintf(
		stdout,
		"\nCapacity:\n  Runnable tasks: %s\n  Pending runners: %s\n  Idle runners: %s\n  Busy runners: %s\n  Terminated runners: %s\n  Generation: %s\n",
		valueOrDash(capacity.GetRunnableTasks()),
		valueOrDash(capacity.GetPendingRunners()),
		valueOrDash(capacity.GetIdleRunners()),
		valueOrDash(capacity.GetBusyRunners()),
		valueOrDash(capacity.GetTerminatedRunners()),
		valueOrDash(capacity.GetGeneration()),
	)
	return err
}

func renderFleet(stdout io.Writer, fleet *openapi_client.RunnersFleet) error {
	if fleet == nil {
		return fmt.Errorf("server returned an empty fleet")
	}
	spec := fleet.GetSpec()
	_, err := fmt.Fprintf(
		stdout,
		"ID: %s\nEnabled: %t\nRunner version: %s\nOperating system: %s\nArchitecture: %s\nCPU: %s\nMemory: %s\nDisk: %s\nCapabilities: %s\nCreated: %s\nUpdated: %s\n",
		fleet.GetId(),
		fleet.GetEnabled(),
		valueOrDash(fleet.GetRunnerVersion()),
		valueOrDash(spec.GetOperatingSystem()),
		valueOrDash(spec.GetArchitecture()),
		formatCPU(spec.GetCpuMillicores()),
		formatMemory(spec.GetMemoryMb()),
		formatDisk(spec.GetDiskGb()),
		valueOrDash(strings.Join(spec.GetCapabilities(), ", ")),
		formatTime(fleet.CreatedAt),
		formatTime(fleet.UpdatedAt),
	)
	return err
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func formatCPU(value int32) string {
	if value == 0 {
		return "-"
	}
	return fmt.Sprintf("%.3g vCPU", float64(value)/1000)
}

func formatMemory(value int32) string {
	if value == 0 {
		return "-"
	}
	return fmt.Sprintf("%d MB", value)
}

func formatDisk(value int32) string {
	if value == 0 {
		return "-"
	}
	return fmt.Sprintf("%d GB", value)
}

func formatTime(value *time.Time) string {
	if value == nil {
		return "-"
	}
	return value.UTC().Format(time.RFC3339)
}
