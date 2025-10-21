package semaphore

import (
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/components"
)

type Semaphore struct{}

type Spec struct {
	Ref          string            `json:"ref"`
	PipelineFile string            `json:"pipeline_file"`
	Parameters   map[string]string `json:"parameters"`
}

// var tools = []components.ToolDefinition{
// 	{
// 		Name:        "list-projects",
// 		Label:       "List Projects",
// 		Description: "List projects in Semaphore",
// 		Parameters:  []components.ConfigurationField{},
// 	},
// 	{
// 		Name:        "list-workflows",
// 		Label:       "List Workflows",
// 		Description: "List workflows in Semaphore",
// 		Parameters: []components.ConfigurationField{
// 			{
// 				Name:     "project",
// 				Label:    "Project",
// 				Type:     components.FieldTypeString,
// 				Required: true,
// 			},
// 		},
// 	},
// 	{
// 		Name:        "run-workflow",
// 		Label:       "Run workflow",
// 		Description: "Run a Semaphore workflow",
// 		Parameters: []components.ConfigurationField{
// 			{
// 				Name:     "project",
// 				Label:    "Project",
// 				Type:     components.FieldTypeString,
// 				Required: true,
// 			},
// 			{
// 				Name:        "pipeline_file",
// 				Type:        components.FieldTypeString,
// 				Description: "Name of the pipeline file to use",
// 				Required:    true,
// 			},
// 			{
// 				Name:        "ref",
// 				Type:        components.FieldTypeString,
// 				Description: "Branch or tag where the workflow to be used is located",
// 				Required:    true,
// 			},
// 			{
// 				Name:     "parameters",
// 				Label:    "Parameters",
// 				Type:     components.FieldTypeList,
// 				Required: false,
// 				ListItem: &components.ListItemDefinition{
// 					Type: components.FieldTypeObject,
// 					Schema: []components.ConfigurationField{
// 						{
// 							Name:     "name",
// 							Type:     components.FieldTypeString,
// 							Label:    "Name",
// 							Required: true,
// 						},
// 						{
// 							Name:     "value",
// 							Type:     components.FieldTypeString,
// 							Label:    "Value",
// 							Required: true,
// 						},
// 					},
// 				},
// 			},
// 		},
// 	},
// }

type Metadata struct {
	Workflow Workflow
}

type Workflow struct {
	ID     string
	State  string
	Result string
}

func (s *Semaphore) Name() string {
	return "semaphore"
}

func (s *Semaphore) Label() string {
	return "Semaphore"
}

func (s *Semaphore) Description() string {
	return "Operate on Semaphore resources"
}

func (s *Semaphore) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{
		components.DefaultOutputChannel,
	}
}

type Mode struct {
	Name        string
	Label       string
	Description string
	Parameters  []components.ConfigurationField
}

func (s *Semaphore) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{}
}

func (s *Semaphore) Execute(ctx components.ExecutionContext) error {
	// spec := Spec{}
	// err := mapstructure.Decode(ctx.Configuration, &spec)
	// if err != nil {
	// 	return err
	// }

	// // TODO: run workflow and update metadata

	// //
	// // Ensure the processing engine will check the status of the workflow created
	// //
	return ctx.RequestContext.ScheduleActionCall("checkStatus", nil, time.Minute)
}

func (s *Semaphore) Actions() []components.Action {
	return []components.Action{
		{
			Name:        "checkStatus",
			Description: "Check the status of the Semaphore workflow",
			Parameters:  []components.ConfigurationField{},
		},
		{
			Name:        "stop",
			Description: "Stop the Semaphore workflow",
			Parameters:  []components.ConfigurationField{},
		},
		{
			Name:        "finish",
			Description: "Finish the execution with outputs",
			Parameters: []components.ConfigurationField{
				{
					Name: "outputs",
					Type: components.FieldTypeObject,
				},
			},
		},
	}
}

func (s *Semaphore) HandleAction(ctx components.ActionContext) error {
	switch ctx.Name {
	case "checkStatus":
		return s.checkStatus(ctx)
	case "stop":
		return s.stop(ctx)
	case "finish":
		return s.finish(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (s *Semaphore) checkStatus(ctx components.ActionContext) error {
	// TODO: check workflow/pipeline status
	return nil
}

func (s *Semaphore) stop(ctx components.ActionContext) error {
	// TODO: stop workflow
	return nil
}

func (s *Semaphore) finish(ctx components.ActionContext) error {
	outputs, ok := ctx.Parameters["outputs"]
	if !ok {
		return fmt.Errorf("outputs is required for finishing")
	}

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		components.DefaultOutputChannel.Name: {outputs},
	})
}
