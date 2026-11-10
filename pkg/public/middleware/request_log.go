package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	requestIDHeader    = "X-Request-Id"
	maxRequestIDLength = 128
)

func withRequestLogFields(r *http.Request, requestID string) *http.Request {
	ctx := logging.WithSecurityIdentity(r.Context(), &logging.SecurityIdentity{RequestID: requestID, ClientIP: logging.ClientIP(r)})
	return r.WithContext(ctx)
}

func requestLogFieldsFrom(ctx context.Context) *logging.SecurityIdentity {
	return logging.SecurityIdentityFrom(ctx)
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

	fields.OrganizationID = uuidString(user.OrganizationID)
	fields.UserID = uuidString(user.ID)
	if user.AccountID != nil {
		fields.AccountID = uuidString(*user.AccountID)
	}
}

// requestLogAccountIdentity keeps the support session target account as account_id
// and the admin account as support_admin_account_id.
func requestLogAccountIdentity(account, effective *models.Account, adminAccountID string) (*models.Account, string) {
	if effective == nil {
		return account, ""
	}
	return effective, strings.TrimSpace(adminAccountID)
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

	fields.AccountID = uuidString(account.ID)
}

// SetRequestLogSupportAdmin records the admin account that started the support session.
func SetRequestLogSupportAdmin(ctx context.Context, adminAccountID string) {
	fields := requestLogFieldsFrom(ctx)
	if fields == nil {
		return
	}

	fields.SupportAdminAccountID = strings.TrimSpace(adminAccountID)
}

// RequestIDFromContext returns the id assigned to this HTTP request.
func RequestIDFromContext(ctx context.Context) string {
	fields := requestLogFieldsFrom(ctx)
	if fields == nil {
		return ""
	}
	return fields.RequestID
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
