package createvm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NormalizeOAuthScopes(t *testing.T) {
	assert.Nil(t, NormalizeOAuthScopes(nil))
	assert.Nil(t, NormalizeOAuthScopes([]string{}))
	assert.Nil(t, NormalizeOAuthScopes([]string{"", "  ", ""}))
	assert.Equal(t, []string{"https://www.googleapis.com/auth/cloud-platform"}, NormalizeOAuthScopes([]string{"  https://www.googleapis.com/auth/cloud-platform  "}))
	assert.Equal(t, []string{"scope1", "scope2"}, NormalizeOAuthScopes([]string{" scope1 ", "", " scope2 "}))
}
