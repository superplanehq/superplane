package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOktaOAuthStateRoundTrip(t *testing.T) {
	s := NewSigner("unit-test-secret")
	state, err := s.SignOktaOAuthState("550e8400-e29b-41d4-a716-446655440000", "/dashboard", time.Minute)
	require.NoError(t, err)

	org, redirect, err := s.ParseOktaOAuthState(state)
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", org)
	assert.Equal(t, "/dashboard", redirect)
}

func TestOktaOAuthStateWrongSecret(t *testing.T) {
	s1 := NewSigner("a")
	s2 := NewSigner("b")
	state, err := s1.SignOktaOAuthState("550e8400-e29b-41d4-a716-446655440000", "/", time.Minute)
	require.NoError(t, err)
	_, _, err = s2.ParseOktaOAuthState(state)
	require.Error(t, err)
}
