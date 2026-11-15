package authentication

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMergeSSOSessionReplacesOneConnection(t *testing.T) {
	first := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	second := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	merged := MergeSSOSession(FormatSSOSessions([]SSOSession{{
		ConnectionID: first,
		AuthUnix:     10,
	}}), second, 20)
	merged = MergeSSOSession(merged, first, 30)

	sessions := ParseSSOSessions(merged)
	require.Len(t, sessions, 2)
	require.Equal(t, first, sessions[0].ConnectionID)
	require.Equal(t, int64(30), sessions[0].AuthUnix)
	require.Equal(t, second, sessions[1].ConnectionID)
	require.Equal(t, int64(20), sessions[1].AuthUnix)
}
