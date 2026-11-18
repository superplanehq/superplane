package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	requestIDHeader    = "X-Request-Id"
	maxRequestIDLength = 128
)

func withRequestLogFields(r *http.Request, requestID string) *http.Request {
	return r.WithContext(logging.WithRequestLogFields(r.Context(), log.Fields{"request_id": requestID}))
}

// SetRequestLogUser records the organization, user, and account for the access log.
func SetRequestLogUser(ctx context.Context, user *models.User) {
	if user == nil {
		return
	}

	fields := logging.RequestLogFields(ctx)
	if fields == nil {
		return
	}

	fields["organization_id"] = uuidString(user.OrganizationID)
	fields["user_id"] = uuidString(user.ID)
	if user.AccountID != nil {
		fields["account_id"] = uuidString(*user.AccountID)
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

	fields := logging.RequestLogFields(ctx)
	if fields == nil {
		return
	}

	fields["account_id"] = uuidString(account.ID)
}

// SetRequestLogSupportAdmin records the admin account that started the support session.
func SetRequestLogSupportAdmin(ctx context.Context, adminAccountID string) {
	fields := logging.RequestLogFields(ctx)
	if fields == nil {
		return
	}

	fields["support_admin_account_id"] = strings.TrimSpace(adminAccountID)
}

// RequestIDFromContext returns the id assigned to this HTTP request.
func RequestIDFromContext(ctx context.Context) string {
	fields := logging.RequestLogFields(ctx)
	if fields == nil {
		return ""
	}
	requestID, _ := fields["request_id"].(string)
	return requestID
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
