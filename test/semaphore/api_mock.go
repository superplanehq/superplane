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

func (m *SemaphoreAPIMock) Close() {
	m.Server.Close()
}

func (m *SemaphoreAPIMock) AddPipeline(ID, workflowID, result string) {
	m.Workflows[workflowID] = Pipeline{ID: ID, Result: result}
}

func (s *SemaphoreAPIMock) Init() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/workflows") {
			s.DescribeWorkflow(w, r)
			return
		}

		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/pipelines") {
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

func (m *SemaphoreAPIMock) DescribeWorkflow(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path, "/")
	workflowID := path[2]

	log.Infof("Describing workflow: %s", workflowID)

	pipeline, ok := m.Workflows[workflowID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data, _ := json.Marshal(semaphore.Workflow{InitialPplID: pipeline.ID})
	w.Write(data)
}

func (m *SemaphoreAPIMock) DescribePipeline(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path, "/")
	pipelineID := path[2]

	log.Infof("Describing pipeline: %s", pipelineID)

	for wfID, p := range m.Workflows {
		if p.ID == pipelineID {
			data, _ := json.Marshal(semaphore.Pipeline{
				ID:         p.ID,
				WorkflowID: wfID,
				State:      semaphore.PipelineStateDone,
				Result:     p.Result,
			})

			w.Write(data)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func (m *SemaphoreAPIMock) TriggerTask(w http.ResponseWriter, r *http.Request) {
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
	data, err := json.Marshal(trigger)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	m.LastTaskTrigger = &trigger
	w.Write(data)
}
