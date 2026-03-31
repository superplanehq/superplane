package scim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrimaryEmail(t *testing.T) {
	email, err := primaryEmail(map[string]interface{}{
		"emails": []interface{}{
			map[string]interface{}{"value": "other@x.com", "primary": false},
			map[string]interface{}{"value": "MAIN@x.com", "primary": true},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "main@x.com", email)
}

func TestPrimaryEmailFallbackFirst(t *testing.T) {
	email, err := primaryEmail(map[string]interface{}{
		"emails": []interface{}{
			map[string]interface{}{"value": "first@x.com"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "first@x.com", email)
}
