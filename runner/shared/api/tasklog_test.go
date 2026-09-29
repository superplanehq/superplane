package api

import (
	"encoding/json"
	"testing"
)

func TestTaskLogSinkCloudWatchJSON(t *testing.T) {
	s := TaskLogSinkCloudWatchFromParts("/my/group", "tasks/abc", "eu-west-1")
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var got TaskLogSink
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Type != TaskLogTypeCloudWatch || got.CloudWatch == nil {
		t.Fatalf("decode: %+v", got)
	}
	if got.CloudWatch.LogGroupName != "/my/group" || got.CloudWatch.LogStreamName != "tasks/abc" || got.CloudWatch.Region != "eu-west-1" {
		t.Fatalf("fields: %+v", got.CloudWatch)
	}
}
