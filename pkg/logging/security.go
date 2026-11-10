package logging

import (
	"context"
	"net"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
)

// SecurityIdentity contains only identities established at authentication boundaries.
// SupportAdminAccountID is separate from the effective customer account.
type SecurityIdentity struct {
	RequestID, ClientIP, OrganizationID, UserID, AccountID, SupportAdminAccountID, ServiceID string
}

type securityIdentityKey struct{}

func WithSecurityIdentity(ctx context.Context, identity *SecurityIdentity) context.Context {
	return context.WithValue(ctx, securityIdentityKey{}, identity)
}

func SecurityIdentityFrom(ctx context.Context) *SecurityIdentity {
	identity, _ := ctx.Value(securityIdentityKey{}).(*SecurityIdentity)
	return identity
}

// SecurityEvent writes metadata-only security records to the process logger.
// Callers must pass explicit safe fields, never request bodies or credentials.
func SecurityEvent(ctx context.Context, action, outcome, targetType, targetID string, fields log.Fields) {
	identity := SecurityIdentity{}
	if known := SecurityIdentityFrom(ctx); known != nil {
		identity = *known
	} else if md, ok := metadata.FromIncomingContext(ctx); ok {
		for key, dest := range map[string]*string{
			"x-account-id": &identity.AccountID, "x-user-id": &identity.UserID,
			"x-organization-id": &identity.OrganizationID,
		} {
			if *dest == "" && len(md.Get(key)) > 0 {
				*dest = md.Get(key)[0]
			}
		}
	}
	entryFields := log.Fields{}
	for key, value := range fields {
		entryFields[key] = value
	}
	for key, value := range map[string]string{
		"actor_account_id": identity.AccountID, "actor_user_id": identity.UserID,
		"actor_service_id": identity.ServiceID, "organization_id": identity.OrganizationID,
		"support_admin_account_id": identity.SupportAdminAccountID,
		"request_id":               identity.RequestID, "client_ip": identity.ClientIP,
		"target_type": targetType, "target_id": targetID,
	} {
		if value != "" {
			entryFields[key] = value
		}
	}
	entryFields["type"] = "security_event"
	entryFields["action"] = action
	entryFields["outcome"] = outcome
	log.WithFields(entryFields).Info("security event")
}

// ClientIP follows the existing proxy header order, with a direct-request fallback.
func ClientIP(r *http.Request) string {
	for _, header := range []string{"CF-Connecting-IP", "True-Client-IP", "X-Real-IP"} {
		if ip := strings.TrimSpace(r.Header.Get(header)); ip != "" {
			return ip
		}
	}
	parts := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(parts) - 1; i >= 0; i-- {
		if ip := strings.TrimSpace(parts[i]); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
