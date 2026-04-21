package compute

import (
	"context"
	"testing"
	"encoding/json"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
)

type mockSubscription struct {
	core.Subscription
	sentBody []byte
}

func (m *mockSubscription) SendMessage(body interface{}) error {
	b, _ := json.Marshal(body)
	m.sentBody = b
	return nil
}

func TestOnInstanceCreated_WebhookHandler(t *testing.T) {
	trigger := &OnInstanceCreated{}
	
	ociEvent := map[string]interface{}{
		"eventType": "com.oraclecloud.computeapi.launchinstance.end",
		"data": map[string]interface{}{
			"resourceName": "test-vm",
			"resourceId": "ocid1.instance.123",
		},
	}
	body, _ := json.Marshal(ociEvent)

	sub := &mockSubscription{}
	ctx := core.TriggerHandlerContext{
		Event: body,
		Subscription: sub,
	}

	err := trigger.Handle(ctx)

	assert.NoError(t, err)
}

func TestOnInstanceStateChange_WebhookHandler(t *testing.T) {
	trigger := &OnInstanceStateChange{}
	
	ociEvent := map[string]interface{}{
		"eventType": "com.oraclecloud.computeapi.instance.statechange",
		"data": map[string]interface{}{
			"state": "RUNNING",
		},
	}
	body, _ := json.Marshal(ociEvent)

	sub := &mockSubscription{}
	ctx := core.TriggerHandlerContext{
		Event: body,
		Subscription: sub,
	}

	err := trigger.Handle(ctx)
	assert.NoError(t, err)
}
