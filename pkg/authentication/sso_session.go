package authentication

import (
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SSOSession is one organization sign-in recorded on the account token.
type SSOSession struct {
	ConnectionID uuid.UUID
	AuthUnix     int64
}

// ParseSSOSessions reads the sso claim. Invalid entries are skipped.
func ParseSSOSessions(raw string) []SSOSession {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	sessions := make([]SSOSession, 0, len(parts))
	for _, part := range parts {
		connectionID, authUnix, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		id, err := uuid.Parse(connectionID)
		if err != nil {
			continue
		}
		unix, err := strconv.ParseInt(authUnix, 10, 64)
		if err != nil {
			continue
		}
		sessions = append(sessions, SSOSession{ConnectionID: id, AuthUnix: unix})
	}
	return sessions
}

// FormatSSOSessions writes the sso claim.
func FormatSSOSessions(sessions []SSOSession) string {
	parts := make([]string, 0, len(sessions))
	for _, session := range sessions {
		parts = append(parts, session.ConnectionID.String()+":"+strconv.FormatInt(session.AuthUnix, 10))
	}
	return strings.Join(parts, ",")
}

// MergeSSOSession replaces the entry for connectionID and keeps the others.
func MergeSSOSession(raw string, connectionID uuid.UUID, authUnix int64) string {
	sessions := ParseSSOSessions(raw)
	replaced := false
	for i, session := range sessions {
		if session.ConnectionID == connectionID {
			sessions[i].AuthUnix = authUnix
			replaced = true
		}
	}
	if !replaced {
		sessions = append(sessions, SSOSession{ConnectionID: connectionID, AuthUnix: authUnix})
	}
	return FormatSSOSessions(sessions)
}

// NewestSSOSession returns the entry with the latest auth time.
func NewestSSOSession(raw string) (SSOSession, bool) {
	sessions := ParseSSOSessions(raw)
	if len(sessions) == 0 {
		return SSOSession{}, false
	}
	newest := sessions[0]
	for _, session := range sessions[1:] {
		if session.AuthUnix >= newest.AuthUnix {
			newest = session
		}
	}
	return newest, true
}

// AccountSession is the account cookie data the SSO checks need.
type AccountSession struct {
	AccountID    string
	IssuedAt     int64
	SessionStart time.Time
	SSO          string
}
