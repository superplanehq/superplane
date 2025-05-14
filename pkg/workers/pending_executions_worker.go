package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/resolver"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	schedulerproto "github.com/superplanehq/superplane/pkg/protos/periodic_scheduler"
	wfproto "github.com/superplanehq/superplane/pkg/protos/plumber_w_f.workflow"
	repoproxyproto "github.com/superplanehq/superplane/pkg/protos/repo_proxy"
)

type PendingExecutionsWorker struct {
	RepoProxyURL string
	SchedulerURL string
	JwtSigner    *jwt.Signer
}

func (w *PendingExecutionsWorker) Start() {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing pending events: %v", err)
		}

		time.Sleep(time.Second)
	}
}

func (w *PendingExecutionsWorker) Tick() error {
	executions, err := models.ListPendingStageExecutions()
	if err != nil {
		return fmt.Errorf("error listing pending stage executions: %v", err)
	}

	for _, execution := range executions {
		stage, err := models.FindStageByID(execution.StageID)
		if err != nil {
			return fmt.Errorf("error finding stage %s: %v", execution.StageID, err)
		}

		logger := logging.ForStage(stage)
		if err := w.ProcessExecution(logger, stage, execution); err != nil {
			return fmt.Errorf("error processing execution %s: %v", execution.ID, err)
		}
	}

	return nil
}

// TODO
// There is an issue here where, if we are having issues updating the state of the execution in the database,
// we might end up creating more executions than we should.
func (w *PendingExecutionsWorker) ProcessExecution(logger *log.Entry, stage *models.Stage, execution models.StageExecution) error {
	resolver := resolver.NewResolver(execution, stage.RunTemplate.Data())
	template, err := resolver.Resolve()
	if err != nil {
		return fmt.Errorf("error resolving run template: %v", err)
	}

	executionID, err := w.StartExecution(logger, stage, execution, *template)
	if err != nil {
		return fmt.Errorf("error starting execution: %v", err)
	}

	err = execution.Start(executionID)
	if err != nil {
		return fmt.Errorf("error moving execution to started state: %v", err)
	}

	err = messages.NewExecutionStartedMessage(stage.CanvasID.String(), &execution).Publish()
	if err != nil {
		return fmt.Errorf("error publishing execution started message: %v", err)
	}

	logger.Infof("Started execution %s", executionID)

	return nil
}

// TODO: implement some retry and give up mechanism
func (w *PendingExecutionsWorker) StartExecution(logger *log.Entry, stage *models.Stage, execution models.StageExecution, template models.RunTemplate) (string, error) {
	switch template.Type {
	case models.RunTemplateTypeSemaphore:
		//
		// If a task ID is specified, we trigger a task instead of a plain workflow.
		//
		if template.Semaphore.TaskID != "" {
			return w.TriggerSemaphoreTask(logger, stage, execution, template.Semaphore)
		}

		return w.StartPlainWorkflow(logger, stage, template.Semaphore)
	default:
		return "", fmt.Errorf("unknown run template type")
	}
}

func (w *PendingExecutionsWorker) TriggerSemaphoreTask(logger *log.Entry, stage *models.Stage, execution models.StageExecution, template *models.SemaphoreRunTemplate) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(w.SchedulerURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("error connecting to task API: %v", err)
	}

	defer conn.Close()

	// TODO: call RBAC API to check if s.CreatedBy can create workflow

	parameters, err := w.buildParameters(execution, template.Parameters)
	if err != nil {
		return "", fmt.Errorf("error building parameters: %v", err)
	}

	client := schedulerproto.NewPeriodicServiceClient(conn)
	res, err := client.RunNow(ctx, &schedulerproto.RunNowRequest{
		Id:              template.TaskID,
		Requester:       stage.CreatedBy.String(),
		Branch:          template.Branch,
		PipelineFile:    template.PipelineFile,
		ParameterValues: parameters,
	})

	if err != nil {
		return "", fmt.Errorf("error calling task API: %v", err)
	}

	if res.Status.Code != code.Code_OK {
		return "", fmt.Errorf("task API returned %v: %s", res.Status.Code, res.Status.Message)
	}

	logger.Infof("Semaphore task triggered - workflow=%s", res.Trigger.ScheduledWorkflowId)
	return res.Trigger.ScheduledWorkflowId, nil
}

func (w *PendingExecutionsWorker) StartPlainWorkflow(logger *log.Entry, stage *models.Stage, template *models.SemaphoreRunTemplate) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(w.RepoProxyURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("error connecting to repo proxy API: %v", err)
	}

	defer conn.Close()

	// TODO: call RBAC API to check if s.CreatedBy can create workflow

	client := repoproxyproto.NewRepoProxyServiceClient(conn)
	res, err := client.Create(ctx, &repoproxyproto.CreateRequest{
		ProjectId:      template.ProjectID,
		RequestToken:   uuid.New().String(),
		RequesterId:    stage.CreatedBy.String(),
		DefinitionFile: template.PipelineFile,
		TriggeredBy:    wfproto.TriggeredBy_API,
		Git: &repoproxyproto.CreateRequest_Git{
			Reference: "refs/heads/" + template.Branch,
		},
	})

	if err != nil {
		return "", fmt.Errorf("error calling repo proxy API: %v", err)
	}

	logger.Infof("Semaphore workflow created: workflow=%s", res.WorkflowId)
	return res.WorkflowId, nil
}

func (w *PendingExecutionsWorker) buildParameters(execution models.StageExecution, parameters map[string]string) ([]*schedulerproto.ParameterValue, error) {
	//
	// Aside from the parameters specified in the run template,
	// we also need to include the token for the execution to push extra tags.
	//
	parameterValues := []*schedulerproto.ParameterValue{
		{Name: "SEMAPHORE_STAGE_ID", Value: execution.StageID.String()},
		{Name: "SEMAPHORE_STAGE_EXECUTION_ID", Value: execution.ID.String()},
	}

	token, err := w.JwtSigner.Generate(execution.ID.String(), 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("error generating tags token: %v", err)
	}

	//
	// TODO: this is a sensitive value, so we can't display it in the UI.
	// We'd need to update plumber to support sensitive parameters if we are gonna do it this way.
	// Otherwise, we'd need to expose these values from zebra.
	//
	parameterValues = append(parameterValues, &schedulerproto.ParameterValue{
		Name:  "SEMAPHORE_STAGE_EXECUTION_TOKEN",
		Value: token,
	})

	for key, value := range parameters {
		parameterValues = append(parameterValues, &schedulerproto.ParameterValue{
			Name:  key,
			Value: value,
		})
	}

	return parameterValues, nil
}
