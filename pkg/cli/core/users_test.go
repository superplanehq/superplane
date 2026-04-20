package core

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestSplitUserIdentifier(t *testing.T) {
	cases := []struct {
		name        string
		positional  string
		emailFlag   string
		wantID      string
		wantEmail   string
		wantErr     bool
		errContains string
	}{
		{name: "both empty", positional: "", emailFlag: "", wantID: "", wantEmail: ""},
		{name: "positional id", positional: "user-1", emailFlag: "", wantID: "user-1", wantEmail: ""},
		{name: "positional email", positional: "alice@example.com", emailFlag: "", wantID: "", wantEmail: "alice@example.com"},
		{name: "email flag only", positional: "", emailFlag: "bob@example.com", wantID: "", wantEmail: "bob@example.com"},
		{name: "trim positional id", positional: "  user-2  ", emailFlag: "", wantID: "user-2", wantEmail: ""},
		{name: "trim positional email", positional: "  carol@example.com  ", emailFlag: "", wantID: "", wantEmail: "carol@example.com"},
		{name: "trim email flag", positional: "", emailFlag: "  dan@example.com  ", wantID: "", wantEmail: "dan@example.com"},
		{name: "both provided rejected", positional: "user-1", emailFlag: "alice@example.com", wantErr: true, errContains: "not both"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, email, err := SplitUserIdentifier(tc.positional, tc.emailFlag)
			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantID, id)
			require.Equal(t, tc.wantEmail, email)
		})
	}
}

func TestOrganizationDomainType(t *testing.T) {
	require.Equal(t,
		openapi_client.AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_ORGANIZATION,
		OrganizationDomainType(),
	)
}
