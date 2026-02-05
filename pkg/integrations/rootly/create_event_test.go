package rootly

import (
	"testing"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/core/test"
)

func TestCreateEvent_Setup(t *testing.T) {
	component := &CreateEvent{}

	tests := []struct {
		name          string
		config        map[string]any
		expectedError string
	}{
		{
			name: "valid configuration with all fields",
			config: map[string]any{
				"incidentId": "inc-123",
				"event":      "Investigation started - checking logs",
				"visibility": "internal",
			},
			expectedError: "",
		},
		{
			name: "valid configuration with required fields only",
			config: map[string]any{
				"incidentId": "inc-456",
				"event":      "Status update from monitoring",
			},
			expectedError: "",
		},
		{
			name: "missing incident ID",
			config: map[string]any{
				"event": "This should fail",
			},
			expectedError: "incident ID is required",
		},
		{
			name: "missing event text",
			config: map[string]any{
				"incidentId": "inc-789",
			},
			expectedError: "event is required",
		},
		{
			name:          "empty configuration",
			config:        map[string]any{},
			expectedError: "incident ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := test.NewSetupContext(tt.config)
			err := component.Setup(ctx)

			if tt.expectedError == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error '%s', got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError {
					t.Errorf("expected error '%s', got: %v", tt.expectedError, err)
				}
			}
		})
	}
}

func TestCreateEvent_Configuration(t *testing.T) {
	component := &CreateEvent{}
	config := component.Configuration()

	if len(config) != 3 {
		t.Errorf("expected 3 configuration fields, got %d", len(config))
	}

	// Check incidentId field
	if config[0].Name != "incidentId" {
		t.Errorf("expected first field name 'incidentId', got '%s'", config[0].Name)
	}
	if !config[0].Required {
		t.Error("expected incidentId to be required")
	}

	// Check event field
	if config[1].Name != "event" {
		t.Errorf("expected second field name 'event', got '%s'", config[1].Name)
	}
	if !config[1].Required {
		t.Error("expected event to be required")
	}

	// Check visibility field
	if config[2].Name != "visibility" {
		t.Errorf("expected third field name 'visibility', got '%s'", config[2].Name)
	}
	if config[2].Required {
		t.Error("expected visibility to be optional")
	}

	// Check visibility options
	if config[2].TypeOptions == nil || config[2].TypeOptions.Select == nil {
		t.Error("expected visibility to have select options")
	} else {
		options := config[2].TypeOptions.Select.Options
		if len(options) != 2 {
			t.Errorf("expected 2 visibility options, got %d", len(options))
		}
		if options[0].Value != "internal" || options[1].Value != "external" {
			t.Errorf("expected visibility options 'internal' and 'external', got '%s' and '%s'", options[0].Value, options[1].Value)
		}
	}
}

func TestCreateEvent_OutputChannels(t *testing.T) {
	component := &CreateEvent{}
	channels := component.OutputChannels(nil)

	if len(channels) != 1 {
		t.Errorf("expected 1 output channel, got %d", len(channels))
	}

	if channels[0].Name != core.DefaultOutputChannel.Name {
		t.Errorf("expected default output channel, got '%s'", channels[0].Name)
	}
}

func TestCreateEvent_Name(t *testing.T) {
	component := &CreateEvent{}
	if component.Name() != "rootly.createEvent" {
		t.Errorf("expected name 'rootly.createEvent', got '%s'", component.Name())
	}
}

func TestCreateEvent_Label(t *testing.T) {
	component := &CreateEvent{}
	if component.Label() != "Create Event" {
		t.Errorf("expected label 'Create Event', got '%s'", component.Label())
	}
}
