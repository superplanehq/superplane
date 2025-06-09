package executors

import (
	"testing"

	"github.com/stretchr/testify/require"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
)

func Test__SpecValidator(t *testing.T) {
	validator := SpecValidator{}

	t.Run("missing executor spec -> error", func(t *testing.T) {
		_, err := validator.Validate(&pb.ExecutorSpec{
			Type: pb.ExecutorSpec_TYPE_HTTP,
		})
		require.ErrorContains(t, err, "missing HTTP executor spec")
	})

	t.Run("HTTP spec with empty URL -> error", func(t *testing.T) {
		in := &pb.ExecutorSpec{
			Type: pb.ExecutorSpec_TYPE_HTTP,
			Http: &pb.ExecutorSpec_HTTP{},
		}
		_, err := validator.Validate(in)
		require.ErrorContains(t, err, "missing URL")
	})

	t.Run("HTTP spec with invalid status code -> error", func(t *testing.T) {
		in := &pb.ExecutorSpec{
			Type: pb.ExecutorSpec_TYPE_HTTP,
			Http: &pb.ExecutorSpec_HTTP{
				Url: "https://httpbin.org/get",
				ResponsePolicy: &pb.ExecutorSpec_HTTPResponsePolicy{
					StatusCodes: []uint32{1000},
				},
			},
		}
		_, err := validator.Validate(in)
		require.ErrorContains(t, err, "invalid status code: 1000")
	})

	t.Run("valid HTTP spec -> no error", func(t *testing.T) {
		in := &pb.ExecutorSpec{
			Http: &pb.ExecutorSpec_HTTP{
				Url: "https://httpbin.org/get",
				Payload: map[string]string{
					"key": "value",
				},
				Headers: map[string]string{
					"x-key": "x-value",
				},
				ResponsePolicy: &pb.ExecutorSpec_HTTPResponsePolicy{
					StatusCodes: []uint32{200, 201},
				},
			},
		}

		_, err := validator.validateHTTPExecutorSpec(in)
		require.NoError(t, err)
	})
}
