package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogUploadPolicyMaxUploadBytes(t *testing.T) {
	policy := LogUploadPolicy{TargetChunkBytes: 32 * 1024}
	assert.Equal(t, int64(64*1024), policy.MaxUploadBytes())

	assert.Equal(
		t,
		int64(128*1024),
		DefaultLogUploadPolicy().MaxUploadBytes(),
	)
}
