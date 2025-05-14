package semaphore

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/apis/semaphore"
)

type SemaphoreAPIMock struct {
	Server    *httptest.Server
	Workflows map[string]Pipeline

	LastTaskTrigger *semaphore.TaskTrigger
}

type Pipeline struct {
	ID     string
	Result string
}

func NewSemaphoreAPIMock() *SemaphoreAPIMock {
	return &SemaphoreAPIMock{Workflows: map[string]Pipeline{}}
}

func (s *SemaphoreAPIMock) Close() {
	s.Server.Close()
}

func (s *SemaphoreAPIMock) AddPipeline(ID, workflowID, result string) {
	s.Workflows[workflowID] = Pipeline{ID: ID, Result: result}
}

func (s *SemaphoreAPIMock) Init() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2/workflows") {
			s.DescribeWorkflow(w, r)
			return
		}

		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1alpha/pipelines") {
			s.DescribePipeline(w, r)
			return
		}

		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/triggers") {
			s.TriggerTask(w, r)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	s.Server = server
}

func (s *SemaphoreAPIMock) DescribeWorkflow(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path, "/")
	workflowID := path[4]

	log.Infof("Workflows: %v", s.Workflows)
	log.Infof("Describing workflow: %s", workflowID)

	pipeline, ok := s.Workflows[workflowID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data, _ := json.Marshal(semaphore.Workflow{InitialPplID: pipeline.ID})
	w.Write(data)
}

func (s *SemaphoreAPIMock) DescribePipeline(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path, "/")
	pipelineID := path[4]

	log.Infof("Describing pipeline: %s", pipelineID)

	for wfID, p := range s.Workflows {
		if p.ID == pipelineID {
			data, _ := json.Marshal(semaphore.PipelineResponse{
				Pipeline: &semaphore.Pipeline{
					ID:         p.ID,
					WorkflowID: wfID,
					State:      semaphore.PipelineStateDone,
					Result:     p.Result,
				},
			})

			w.Write(data)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func (s *SemaphoreAPIMock) TriggerTask(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	var trigger semaphore.TaskTrigger
	err = json.Unmarshal(body, &trigger)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	trigger.Metadata.WorkflowID = uuid.New().String()
	trigger.Metadata.Status = "PASSED"
	data, err := json.Marshal(trigger)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	s.LastTaskTrigger = &trigger
	w.Write(data)
}
