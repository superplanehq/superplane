package public

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRepositorySpecFileHTTPStatus(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "NotFound -> 404 with grpc message",
			err:        status.Error(codes.NotFound, "canvas live version not found"),
			wantStatus: http.StatusNotFound,
			wantBody:   "canvas live version not found",
		},
		{
			name:       "PermissionDenied -> 403",
			err:        status.Error(codes.PermissionDenied, "version is not visible in current flow"),
			wantStatus: http.StatusForbidden,
			wantBody:   "version is not visible in current flow",
		},
		{
			name:       "Unauthenticated -> 401",
			err:        status.Error(codes.Unauthenticated, "user not authenticated"),
			wantStatus: http.StatusUnauthorized,
			wantBody:   "user not authenticated",
		},
		{
			name:       "InvalidArgument -> 400",
			err:        status.Error(codes.InvalidArgument, "invalid version id"),
			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid version id",
		},
		{
			name:       "FailedPrecondition -> 400",
			err:        status.Error(codes.FailedPrecondition, "bad state"),
			wantStatus: http.StatusBadRequest,
			wantBody:   "bad state",
		},
		{
			name:       "Internal -> 500 with generic message",
			err:        status.Error(codes.Internal, "boom"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "Failed to get file",
		},
		{
			name:       "non-gRPC error -> 500",
			err:        errors.New("plain error"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "Failed to get file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeRepositorySpecFileError(rec, tc.err, "canvas.yaml", "test-canvas-id")
			assert.Equal(t, tc.wantStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tc.wantBody)
		})
	}
}
