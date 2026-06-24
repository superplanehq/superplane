package grpcactionsstatus

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScan(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "handler.go")
	require.NoError(t, os.WriteFile(path, []byte(`package sample

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Handler() error {
	return status.Error(codes.InvalidArgument, "bad input")
}
`), 0o644))

	violations, err := Scan(root)
	require.NoError(t, err)
	require.Len(t, violations, 2)
	assert.Equal(t, "imports google.golang.org/grpc/status", violations[0].Detail)
	assert.Equal(t, "uses status.Error", violations[1].Detail)
}

func TestScanSkipsTests(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "handler_test.go")
	require.NoError(t, os.WriteFile(path, []byte(`package sample

import "google.golang.org/grpc/status"

func TestHandler(t *testing.T) {
	_ = status.Error(0, "ignored")
}
`), 0o644))

	violations, err := Scan(root)
	require.NoError(t, err)
	assert.Empty(t, violations)
}
