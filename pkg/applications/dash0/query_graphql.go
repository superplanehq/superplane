package dash0

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type QueryGraphQL struct{}

type QueryGraphQLSpec struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

func (q *QueryGraphQL) Name() string {
	return "dash0.queryGraphQL"
}

func (q *QueryGraphQL) Label() string {
	return "Query GraphQL"
}

func (q *QueryGraphQL) Description() string {
	return "Execute a GraphQL query against Dash0 and return the response data"
}

func (q *QueryGraphQL) Icon() string {
	return "database"
}

func (q *QueryGraphQL) Color() string {
	return "blue"
}

func (q *QueryGraphQL) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (q *QueryGraphQL) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "query",
			Label:       "GraphQL Query",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The GraphQL query to execute",
			Placeholder: "query { __typename }",
		},
		{
			Name:        "variables",
			Label:       "Variables",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "GraphQL variables as a JSON object",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{},
				},
			},
		},
	}
}

func (q *QueryGraphQL) Setup(ctx core.SetupContext) error {
	spec := QueryGraphQLSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Query == "" {
		return errors.New("query is required")
	}

	// Basic validation - check that query is not just whitespace
	if len(spec.Query) == 0 || len(strings.TrimSpace(spec.Query)) == 0 {
		return errors.New("query cannot be empty")
	}

	return nil
}

func (q *QueryGraphQL) Execute(ctx core.ExecutionContext) error {
	spec := QueryGraphQLSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	data, err := client.ExecuteGraphQL(spec.Query, spec.Variables)
	if err != nil {
		return fmt.Errorf("failed to execute GraphQL query: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.graphql.response",
		[]any{data},
	)
}

func (q *QueryGraphQL) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (q *QueryGraphQL) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (q *QueryGraphQL) Actions() []core.Action {
	return []core.Action{}
}

func (q *QueryGraphQL) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (q *QueryGraphQL) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
