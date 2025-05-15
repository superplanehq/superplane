package semaphore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Semaphore struct {
	URL   string
	Token string
}

func NewSemaphoreAPI(URL, token string) *Semaphore {
	return &Semaphore{
		URL:   URL,
		Token: token,
	}
}

type TaskTrigger struct {
	Kind       string              `json:"kind"`
	APIVersion string              `json:"apiVersion"`
	Metadata   TaskTriggerMetadata `json:"metadata,omitempty"`
	Spec       TaskTriggerSpec     `json:"spec"`
}

type TaskTriggerMetadata struct {
	WorkflowID string `json:"workflow_id"`
	Status     string `json:"status"`
}

type TaskTriggerSpec struct {
	Branch       string                 `json:"branch"`
	PipelineFile string                 `json:"pipeline_file"`
	Parameters   []TaskTriggerParameter `json:"parameters"`
}

type TaskTriggerParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Workflow struct {
	InitialPplID string `json:"initial_ppl_id"`
}

const (
	PipelineStateDone    = "done"
	PipelineResultPassed = "passed"
	PipelineResultFailed = "failed"
)

type PipelineResponse struct {
	Pipeline *Pipeline `json:"pipeline"`
}

type Pipeline struct {
	ID         string `json:"ppl_id"`
	WorkflowID string `json:"wf_id"`
	State      string `json:"state"`
	Result     string `json:"result"`
}

func (s *Semaphore) DescribeWorkflow(workflowID string) (*Workflow, error) {
	URL := fmt.Sprintf("%s/api/v2/workflows/%s", s.URL, workflowID)
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+s.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request got %d code", res.StatusCode)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	var workflow Workflow
	err = json.Unmarshal(responseBody, &workflow)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &workflow, nil
}

// NOTE: pipelines v2 API is not working :)
func (s *Semaphore) DescribePipeline(pipelineID string) (*Pipeline, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/pipelines/%s", s.URL, pipelineID)
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+s.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request got %d code", res.StatusCode)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	var pipelineResponse PipelineResponse
	err = json.Unmarshal(responseBody, &pipelineResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return pipelineResponse.Pipeline, nil
}

func (s *Semaphore) TriggerTask(projectID, taskID string, spec TaskTriggerSpec) (string, error) {
	URL := fmt.Sprintf("%s/api/v2/projects/%s/tasks/%s/triggers", s.URL, projectID, taskID)

	body, err := json.Marshal(&TaskTrigger{
		APIVersion: "v2",
		Kind:       "TaskTrigger",
		Spec:       spec,
	})

	if err != nil {
		return "", fmt.Errorf("error marshaling task trigger: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, URL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+s.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error executing request: %v", err)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	var trigger TaskTrigger
	err = json.Unmarshal(responseBody, &trigger)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	if trigger.Metadata.Status != "PASSED" {
		return "", fmt.Errorf("trigger status was %s", trigger.Metadata.Status)
	}

	return trigger.Metadata.WorkflowID, nil
}
