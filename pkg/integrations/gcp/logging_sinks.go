package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

const loggingBaseURL = "https://logging.googleapis.com/v2"

const vmCreateLogFilter = `protoPayload.serviceName="compute.googleapis.com" AND (protoPayload.methodName="v1.compute.instances.insert" OR protoPayload.methodName="beta.compute.instances.insert" OR protoPayload.methodName="compute.instances.insert")`

func CreateVMCreatedSink(ctx context.Context, client *Client, projectID, sinkID, topicFullName string) (writerIdentity string, err error) {
	destination := fmt.Sprintf("pubsub.googleapis.com/%s", topicFullName)
	url := fmt.Sprintf("%s/projects/%s/sinks?uniqueWriterIdentity=true", loggingBaseURL, projectID)
	body := map[string]string{
		"name":        sinkID,
		"destination": destination,
		"filter":      vmCreateLogFilter,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal sink body: %w", err)
	}
	resp, err := client.ExecRequest(ctx, "POST", url, bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("create log sink: %w", err)
	}
	var sink struct {
		WriterIdentity string `json:"writerIdentity"`
	}
	if err := json.Unmarshal(resp, &sink); err != nil {
		return "", fmt.Errorf("parse sink response: %w", err)
	}
	return sink.WriterIdentity, nil
}

func DeleteSink(ctx context.Context, client *Client, projectID, sinkID string) error {
	url := fmt.Sprintf("%s/projects/%s/sinks/%s", loggingBaseURL, projectID, sinkID)
	_, err := client.ExecRequest(ctx, "DELETE", url, nil)
	return err
}
