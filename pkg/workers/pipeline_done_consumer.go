package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/events"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pplproto "github.com/superplanehq/superplane/pkg/protos/plumber.pipeline"
	"github.com/superplanehq/superplane/pkg/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type PipelineDoneConsumer struct {
	Consumer       *tackle.Consumer
	RabbitMQURL    string
	PipelineAPIURL string
}

func NewPipelineDoneConsumer(rabbitMQURL, pipelineAPIURL string) *PipelineDoneConsumer {
	return &PipelineDoneConsumer{
		RabbitMQURL:    rabbitMQURL,
		PipelineAPIURL: pipelineAPIURL,
		Consumer:       tackle.NewConsumer(),
	}
}

func (c *PipelineDoneConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		Service:        "superplane",
		ConnectionName: "superplane",
		RemoteExchange: "pipeline_state_exchange",
		RoutingKey:     "done",
	}

	err := retry.WithConstantWait("RabbitMQ connection", 5, time.Second, func() error {
		return c.Consumer.Start(&options, c.Consume)
	})

	if err != nil {
		return fmt.Errorf("error starting consumer: %v", err)
	}

	return nil
}

func (c *PipelineDoneConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *PipelineDoneConsumer) Consume(delivery tackle.Delivery) error {
	pipelineEvent := &pplproto.PipelineEvent{}
	err := proto.Unmarshal(delivery.Body(), pipelineEvent)
	if err != nil {
		return err
	}

	ID := pipelineEvent.PipelineId
	log.Infof("Received message for %s", ID)

	//
	// TODO
	//
	// Currently, we need to describe the pipeline to find the workflow ID and the result of the pipeline.
	// Going to the pipeline API on every event for this is expensive, and
	// if we put the workflow ID and pipeline result in the message,
	// we wouldn't need to do this.
	//
	pipeline, err := c.describePipeline(ID)
	if err != nil {
		log.Errorf("Error describing pipeline %s: %v", ID, err)
		return err
	}

	//
	// Not all pipelines are related to stage executions, so we
	// check if there are is a stage execution associated with this workflow first.
	//
	execution, err := models.FindExecutionByReference(pipeline.WfId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Infof("No execution for %s - ignoring", ID)
			return nil
		}
	}

	//
	// The message doesn't contain the result of the pipeline,
	// so we need to go to use the pipeline API for that,
	// and map the pipeline result to a stage execution result.
	//
	logger := logging.ForExecution(execution)

	//
	// Update the stage execution accordingly.
	//
	result := c.resolveExecutionResult(logger, pipeline)

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		tags, err := c.processExecutionTags(tx, logger, execution, result)
		if err != nil {
			logger.Errorf("Error processing execution tags: %v", err)
			return err
		}

		if err := execution.FinishInTransaction(tx, result); err != nil {
			logger.Errorf("Error updating execution state: %v", err)
			return err
		}

		err = models.UpdateStageEventsInTransaction(
			tx, []string{execution.StageEventID.String()}, models.StageEventStateProcessed, "",
		)

		if err != nil {
			logger.Errorf("Error updating stage event state: %v", err)
			return err
		}

		//
		// Lastly, since the stage for this execution might be connected to other stages,
		// we create a new event for the completion of this stage.
		//
		if err := c.createStageCompletionEvent(tx, execution, tags); err != nil {
			logger.Errorf("Error creating stage completion event: %v", err)
			return err
		}

		logger.Infof("Execution state updated: %s", result)
		return nil
	})

	if err == nil {
		stage, err := models.FindStageByID(execution.StageID)
		if err != nil {
			logger.Errorf("Error finding stage for execution: %v", err)
			return err
		}

		err = messages.NewExecutionFinishedMessage(stage.CanvasID.String(), execution).Publish()
		if err != nil {
			logger.Errorf("Error publishing execution finished message: %v", err)
		}
	}

	return err
}

func (c *PipelineDoneConsumer) describePipeline(id string) (*pplproto.Pipeline, error) {
	conn, err := grpc.NewClient(c.PipelineAPIURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("error connecting to repo proxy API: %v", err)
	}

	defer conn.Close()

	client := pplproto.NewPipelineServiceClient(conn)
	res, err := client.Describe(context.TODO(), &pplproto.DescribeRequest{
		PplId:    id,
		Detailed: false,
	})

	if err != nil {
		return nil, fmt.Errorf("error describing pipeline: %v", err)
	}

	return res.Pipeline, nil
}

func (c *PipelineDoneConsumer) resolveExecutionResult(logger *log.Entry, pipeline *pplproto.Pipeline) string {
	logger.Infof("Pipeline %s state: %v", pipeline.PplId, pipeline.Result)

	switch pipeline.Result {
	case pplproto.Pipeline_PASSED:
		return models.StageExecutionResultPassed
	default:
		return models.StageExecutionResultFailed
	}
}

func (c *PipelineDoneConsumer) createStageCompletionEvent(tx *gorm.DB, execution *models.StageExecution, tags map[string]string) error {
	stage, err := models.FindStageByIDInTransaction(tx, execution.StageID)
	if err != nil {
		return err
	}

	e, err := events.NewStageExecutionCompletion(execution, tags)
	if err != nil {
		return fmt.Errorf("error creating stage completion event: %v", err)
	}

	raw, err := json.Marshal(&e)
	if err != nil {
		return fmt.Errorf("error marshaling event: %v", err)
	}

	_, err = models.CreateEventInTransaction(tx, execution.StageID, stage.Name, models.SourceTypeStage, raw, []byte(`{}`))
	if err != nil {
		return fmt.Errorf("error creating event: %v", err)
	}

	return nil
}

func (c *PipelineDoneConsumer) processExecutionTags(tx *gorm.DB, logger *log.Entry, execution *models.StageExecution, result string) (map[string]string, error) {
	allTags := map[string]string{}

	//
	// Include extra tags from execution, if any.
	//
	if execution.Tags != nil {
		err := json.Unmarshal(execution.Tags, &allTags)
		if err != nil {
			return nil, fmt.Errorf("error adding tags from execution: %v", err)
		}
	}

	//
	// Include tags from event
	//
	tagsFromEvent, err := models.FindStageEventTagsInTransaction(tx, execution.StageEventID)
	if err != nil {
		return nil, fmt.Errorf("error finding tags from stage event: %v", err)
	}

	for _, t := range tagsFromEvent {
		allTags[t.Name] = t.Value
	}

	if len(allTags) == 0 {
		logger.Warningf("No tags")
		return allTags, nil
	}

	newState := resolveTagState(result)
	err = models.UpdateStageEventTagStateInBulk(tx, execution.StageEventID, newState, allTags)
	if err != nil {
		logger.Errorf("Error updating tags to %s state: %v", newState, err)
		return nil, err
	}

	logger.Infof("Updated tags %v to %s state", allTags, newState)
	return allTags, nil
}

func resolveTagState(result string) string {
	switch result {
	case models.StageExecutionResultPassed:
		return models.TagStateHealthy
	default:
		return models.TagStateUnhealthy
	}
}
