package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	requestIDHeader    = "X-Request-Id"
	maxRequestIDLength = 128
)

type requestLogContextKey struct{}

// requestLogFields is filled by auth middleware after LoggingMiddleware seeds it.
// The pointer stays on the request context so later handlers can add identity
// before the access log line is written.
type requestLogFields struct {
	requestID             string
	organizationID        string
	userID                string
	accountID             string
	impersonatorAccountID string
}

func withRequestLogFields(r *http.Request, requestID string) *http.Request {
	fields := &requestLogFields{requestID: requestID}
	ctx := context.WithValue(r.Context(), requestLogContextKey{}, fields)
	return r.WithContext(ctx)
}

func requestLogFieldsFrom(ctx context.Context) *requestLogFields {
	fields, _ := ctx.Value(requestLogContextKey{}).(*requestLogFields)
	return fields
}

// SetRequestLogUser records the organization, user, and account for the access log.
func SetRequestLogUser(ctx context.Context, user *models.User) {
	if user == nil {
		return
	}

	fields := requestLogFieldsFrom(ctx)
	if fields == nil {
		return
	}

	fields.organizationID = uuidString(user.OrganizationID)
	fields.userID = uuidString(user.ID)
	if user.AccountID != nil {
		fields.accountID = uuidString(*user.AccountID)
	}
}

// SetRequestLogAccount records the authenticated account for the access log.
func SetRequestLogAccount(ctx context.Context, account *models.Account) {
	if account == nil {
		return
	}

	fields := requestLogFieldsFrom(ctx)
	if fields == nil {
		return
	}

	fields.accountID = uuidString(account.ID)
}

// SetRequestLogImpersonator records the admin account that started impersonation.
func SetRequestLogImpersonator(ctx context.Context, adminAccountID string) {
	fields := requestLogFieldsFrom(ctx)
	if fields == nil {
		return
	}

	fields.impersonatorAccountID = strings.TrimSpace(adminAccountID)
}

// RequestIDFromContext returns the id assigned to this HTTP request.
func RequestIDFromContext(ctx context.Context) string {
	fields := requestLogFieldsFrom(ctx)
	if fields == nil {
		return ""
	}
	return fields.requestID
}

func requestIDFromHeader(r *http.Request) string {
	incoming := strings.TrimSpace(r.Header.Get(requestIDHeader))
	if validRequestID(incoming) {
		return incoming
	}
	return uuid.NewString()
}

func validRequestID(value string) bool {
	if value == "" || len(value) > maxRequestIDLength {
		return false
	}
	return !strings.ContainsAny(value, "\r\n")
}

func uuidString(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}
