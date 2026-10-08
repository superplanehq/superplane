package public

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc"
	"gorm.io/gorm"
)

func Test__AccountLookupError(t *testing.T) {
	lookupErr := errors.New("db down")

	tests := []struct {
		name           string
		err            error
		missingAccount bool
		cancelRequest  bool
		wantStatus     int
		wantRecorded   bool
	}{
		{
			name:           "record not found is unauthorized",
			err:            gorm.ErrRecordNotFound,
			missingAccount: true,
			wantStatus:     http.StatusUnauthorized,
		},
		{
			name:          "deadline after disconnect is client closed",
			err:           context.DeadlineExceeded,
			cancelRequest: true,
			wantStatus:    statusClientClosedRequest,
		},
		{
			name:         "generic lookup error is recorded",
			err:          lookupErr,
			wantStatus:   http.StatusInternalServerError,
			wantRecorded: true,
		},
		{
			name:         "deadline on a live request is recorded",
			err:          context.DeadlineExceeded,
			wantStatus:   http.StatusGatewayTimeout,
			wantRecorded: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			if test.cancelRequest {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			request := httptest.NewRequest(http.MethodGet, "/account", nil).WithContext(ctx)
			request = grpc.WithServerErrorReport(request)
			response := httptest.NewRecorder()

			writeAccountLookupError(response, request, accountLookupFailure{
				err:            test.err,
				detail:         "Error reloading account",
				missingAccount: test.missingAccount,
			})

			assert.Equal(t, test.wantStatus, response.Code)
			noted, recorded := grpc.NotedServerErrorFrom(request.Context())
			if !test.wantRecorded {
				assert.False(t, recorded)
				return
			}

			require.True(t, recorded)
			assert.ErrorIs(t, noted.Cause, test.err)
			assert.Empty(t, response.Body.String())
		})
	}
}
