package executors

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test_HTTP(t *testing.T) {
	executionID := uuid.New()
	stageID := uuid.New()
	execution := models.StageExecution{
		ID:      executionID,
		StageID: stageID,
	}

	t.Run("200 response is successful", func(t *testing.T) {
		executor, err := NewHTTPExecutor(execution, nil)
		require.NoError(t, err)
		require.NotNil(t, executor)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()

		response, err := executor.Execute(models.ExecutorSpec{
			HTTP: &models.HTTPExecutorSpec{
				URL: server.URL,
				ResponsePolicy: &models.HTTPResponsePolicy{
					StatusCodes: []uint32{200},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.True(t, response.Successful())
	})

	t.Run("400 response is not successful", func(t *testing.T) {
		executor, err := NewHTTPExecutor(execution, nil)
		require.NoError(t, err)
		require.NotNil(t, executor)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))

		defer server.Close()

		response, err := executor.Execute(models.ExecutorSpec{
			HTTP: &models.HTTPExecutorSpec{
				URL: server.URL,
				ResponsePolicy: &models.HTTPResponsePolicy{
					StatusCodes: []uint32{200},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.False(t, response.Successful())
	})

	t.Run("body contains spec payload", func(t *testing.T) {
		executor, err := NewHTTPExecutor(execution, nil)
		require.NoError(t, err)
		require.NotNil(t, executor)

		var body map[string]string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &body)
			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()

		response, err := executor.Execute(models.ExecutorSpec{
			HTTP: &models.HTTPExecutorSpec{
				URL:     server.URL,
				Payload: map[string]string{"foo": "bar"},
				ResponsePolicy: &models.HTTPResponsePolicy{
					StatusCodes: []uint32{200},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.True(t, response.Successful())
		assert.Equal(t, "bar", body["foo"])
		assert.Equal(t, execution.StageID.String(), body["stageId"])
		assert.Equal(t, execution.ID.String(), body["executionId"])
	})

	t.Run("headers contains spec payload", func(t *testing.T) {
		executor, err := NewHTTPExecutor(execution, nil)
		require.NoError(t, err)
		require.NotNil(t, executor)

		headers := map[string]string{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for k, v := range r.Header {
				headers[strings.ToLower(k)] = strings.ToLower(v[0])
			}
			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()

		response, err := executor.Execute(models.ExecutorSpec{
			HTTP: &models.HTTPExecutorSpec{
				URL:     server.URL,
				Headers: map[string]string{"x-foo": "bar"},
				ResponsePolicy: &models.HTTPResponsePolicy{
					StatusCodes: []uint32{200},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.True(t, response.Successful())
		assert.Equal(t, "bar", headers["x-foo"])
	})

	t.Run("outputs are returned in the response body", func(t *testing.T) {
		executor, err := NewHTTPExecutor(execution, nil)
		require.NoError(t, err)
		require.NotNil(t, executor)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"outputs": {"foo": "bar"}}`))
		}))

		defer server.Close()

		response, err := executor.Execute(models.ExecutorSpec{
			HTTP: &models.HTTPExecutorSpec{
				URL: server.URL,
				ResponsePolicy: &models.HTTPResponsePolicy{
					StatusCodes: []uint32{200},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.True(t, response.Successful())
		assert.Equal(t, map[string]any{"foo": "bar"}, response.Outputs())
	})
}
