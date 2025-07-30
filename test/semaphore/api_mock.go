package semaphore

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
)

type SemaphoreAPIMock struct {
	Server    *httptest.Server
	Workflows map[string]Pipeline
	Projects  []string

	LastRunTask     *semaphore.RunTaskRequest
	LastRunWorkflow *semaphore.CreateWorkflowRequest
}

type Pipeline struct {
	ID     string
	Result string
}

func NewSemaphoreAPIMock() *SemaphoreAPIMock {
	return &SemaphoreAPIMock{
		Projects:  []string{"demo-project", "demo-project-2"},
		Workflows: map[string]Pipeline{},
	}
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

		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1alpha/projects") {
			s.DescribeProject(w, r)
			return
		}

		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v1alpha/plumber-workflows") {
			s.RunWorkflow(w, r)
			return
		}

		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/run_now") {
			s.RunTask(w, r)
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

func (s *SemaphoreAPIMock) DescribeProject(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path, "/")
	projectName := path[4]

	if !slices.Contains(s.Projects, projectName) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data, _ := json.Marshal(semaphore.Project{
		Metadata: &semaphore.ProjectMetadata{
			ProjectName: projectName,
			ProjectID:   uuid.New().String(),
		},
	})

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
					PipelineID: p.ID,
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

func (s *SemaphoreAPIMock) RunTask(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	var runTaskRequest semaphore.RunTaskRequest
	err = json.Unmarshal(body, &runTaskRequest)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	response := semaphore.RunTaskResponse{WorkflowID: uuid.New().String()}
	data, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	s.LastRunTask = &runTaskRequest
	w.Write(data)
}

func (s *SemaphoreAPIMock) RunWorkflow(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	var params semaphore.CreateWorkflowRequest
	err = json.Unmarshal(body, &params)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	data, err := json.Marshal(semaphore.CreateWorkflowResponse{
		WorkflowID: uuid.New().String(),
	})

	if err != nil {
		w.WriteHeader(500)
		return
	}

	s.LastRunWorkflow = &params
	w.Write(data)
}
