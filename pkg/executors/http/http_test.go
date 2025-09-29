package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test_HTTP__Execute(t *testing.T) {
	executor := NewHTTPExecutor(http.DefaultClient)

	executionID := uuid.New()
	stageID := uuid.New()
	execution := models.StageExecution{
		ID:      executionID,
		StageID: stageID,
	}

	t.Run("200 response is successful", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()

		spec, err := json.Marshal(HTTPSpec{URL: server.URL,
			ResponsePolicy: &HTTPResponsePolicy{
				StatusCodes: []uint32{200},
			},
		})

		require.NoError(t, err)
		response, err := executor.Execute(spec, executors.ExecutionParameters{
			StageID:     stageID.String(),
			ExecutionID: executionID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.True(t, response.Successful())
	})

	t.Run("400 response is not successful", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))

		defer server.Close()

		spec, err := json.Marshal(&HTTPSpec{
			URL: server.URL,
			ResponsePolicy: &HTTPResponsePolicy{
				StatusCodes: []uint32{200},
			},
		})

		require.NoError(t, err)
		_, err = executor.Execute(spec, executors.ExecutionParameters{
			StageID:     stageID.String(),
			ExecutionID: executionID.String(),
		})

		require.ErrorContains(t, err, "invalid HTTP response: status code 400 not in allowed codes")
	})

	t.Run("body contains spec payload", func(t *testing.T) {
		var body map[string]string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &body)
			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()

		spec, err := json.Marshal(&HTTPSpec{
			URL:     server.URL,
			Payload: map[string]string{"foo": "bar"},
			ResponsePolicy: &HTTPResponsePolicy{
				StatusCodes: []uint32{200},
			},
		})

		require.NoError(t, err)
		response, err := executor.Execute(spec, executors.ExecutionParameters{
			StageID:     stageID.String(),
			ExecutionID: executionID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.True(t, response.Successful())
		assert.Equal(t, "bar", body["foo"])
		assert.Equal(t, execution.StageID.String(), body["stageId"])
		assert.Equal(t, execution.ID.String(), body["executionId"])
	})

	t.Run("headers contains spec payload", func(t *testing.T) {
		headers := map[string]string{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for k, v := range r.Header {
				headers[strings.ToLower(k)] = strings.ToLower(v[0])
			}
			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()

		spec, err := json.Marshal(&HTTPSpec{
			URL:     server.URL,
			Headers: map[string]string{"x-foo": "bar"},
			ResponsePolicy: &HTTPResponsePolicy{
				StatusCodes: []uint32{200},
			},
		})

		require.NoError(t, err)
		response, err := executor.Execute(spec, executors.ExecutionParameters{
			StageID:     stageID.String(),
			ExecutionID: executionID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.True(t, response.Successful())
		assert.Equal(t, "bar", headers["x-foo"])
	})

	t.Run("outputs are returned in the response body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"outputs": {"foo": "bar"}}`))
		}))

		defer server.Close()

		spec, err := json.Marshal(&HTTPSpec{
			URL: server.URL,
			ResponsePolicy: &HTTPResponsePolicy{
				StatusCodes: []uint32{200},
			},
		})

		require.NoError(t, err)
		response, err := executor.Execute(spec, executors.ExecutionParameters{
			StageID:     stageID.String(),
			ExecutionID: executionID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.True(t, response.Successful())
		assert.Equal(t, map[string]any{"foo": "bar"}, response.Outputs())
	})

	t.Run("nil response policy handles gracefully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()

		spec, err := json.Marshal(HTTPSpec{
			URL:            server.URL,
			ResponsePolicy: nil,
		})

		require.NoError(t, err)
		_, err = executor.Execute(spec, executors.ExecutionParameters{
			StageID:     stageID.String(),
			ExecutionID: executionID.String(),
		})

		require.ErrorContains(t, err, "invalid HTTP response: status code 200 not in allowed codes")
	})

	t.Run("missing response policy in JSON handles gracefully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()

		jsonSpec := `{"url": "` + server.URL + `", "headers": {}, "payload": {}}`

		_, err := executor.Execute([]byte(jsonSpec), executors.ExecutionParameters{
			StageID:     stageID.String(),
			ExecutionID: executionID.String(),
		})

		require.ErrorContains(t, err, "invalid HTTP response: status code 200 not in allowed codes")
	})

	t.Run("panic regression test - nil response policy access", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()

		jsonSpec := `{"url": "` + server.URL + `", "headers": {}, "payload": {}, "responsePolicy": null}`

		// This should not panic and should handle gracefully
		require.NotPanics(t, func() {
			_, err := executor.Execute([]byte(jsonSpec), executors.ExecutionParameters{
				StageID:     stageID.String(),
				ExecutionID: executionID.String(),
			})

			require.ErrorContains(t, err, "invalid HTTP response: status code 200 not in allowed codes")
		})
	})
}

func Test_HTTP__Validate(t *testing.T) {
	executor := NewHTTPExecutor(http.DefaultClient)

	t.Run("HTTP spec with empty URL -> error", func(t *testing.T) {
		spec := HTTPSpec{URL: ""}
		data, err := json.Marshal(&spec)
		require.NoError(t, err)

		err = executor.Validate(context.Background(), data)
		require.ErrorContains(t, err, "missing URL")
	})

	t.Run("HTTP spec with invalid status code -> error", func(t *testing.T) {
		spec := HTTPSpec{
			URL: "https://httpbin.org/get",
			ResponsePolicy: &HTTPResponsePolicy{
				StatusCodes: []uint32{1000},
			},
		}

		data, err := json.Marshal(&spec)
		require.NoError(t, err)

		err = executor.Validate(context.Background(), data)
		require.ErrorContains(t, err, "invalid status code: 1000")
	})

	t.Run("valid HTTP spec -> no error", func(t *testing.T) {
		spec := HTTPSpec{
			URL: "https://httpbin.org/get",
			Payload: map[string]string{
				"key": "value",
			},
			Headers: map[string]string{
				"x-key": "x-value",
			},
			ResponsePolicy: &HTTPResponsePolicy{
				StatusCodes: []uint32{200, 201},
			},
		}

		data, err := json.Marshal(&spec)
		require.NoError(t, err)
		require.NoError(t, executor.Validate(context.Background(), data))
	})

	t.Run("HTTP spec with nil response policy -> no error", func(t *testing.T) {
		spec := HTTPSpec{
			URL:            "https://httpbin.org/get",
			ResponsePolicy: nil, // Explicitly nil
		}

		data, err := json.Marshal(&spec)
		require.NoError(t, err)

		err = executor.Validate(context.Background(), data)
		require.NoError(t, err) // Should not error with nil ResponsePolicy
	})

	t.Run("HTTP spec missing response policy field -> no error", func(t *testing.T) {
		// JSON without responsePolicy field
		jsonSpec := `{"url": "https://httpbin.org/get", "headers": {}, "payload": {}}`

		err := executor.Validate(context.Background(), []byte(jsonSpec))
		require.NoError(t, err) // Should not error with missing ResponsePolicy field
	})
}
