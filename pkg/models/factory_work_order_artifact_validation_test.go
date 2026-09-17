package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileArtifactValidation(t *testing.T) {
	assert.False(t, IsValidWorkOrderArtifactType(FactoryWorkOrderArtifactTypeFile), "only the runner upload service can create file artifacts")
	valid := map[string]any{
		"fileId":      "2e35918e-c88e-43f2-aa02-3acdf637183a",
		"filename":    "checkout.webm",
		"contentType": "video/webm",
		"sizeBytes":   int64(128),
		"title":       "Checkout",
		"url":         "https://app.example/api/v1/public/artifacts/capability/checkout.webm",
	}
	assert.NoError(t, validateArtifactData(FactoryWorkOrderArtifactTypeFile, valid))

	missingURL := map[string]any{
		"fileId":      valid["fileId"],
		"filename":    valid["filename"],
		"contentType": valid["contentType"],
		"sizeBytes":   valid["sizeBytes"],
		"title":       valid["title"],
	}
	assert.ErrorIs(t, validateArtifactData(FactoryWorkOrderArtifactTypeFile, missingURL), ErrFactoryWorkOrderArtifactInvalid)
	assert.ErrorIs(t, validateArtifactData(FactoryWorkOrderArtifactTypeFile, map[string]any{
		"fileId":      valid["fileId"],
		"filename":    valid["filename"],
		"contentType": valid["contentType"],
		"sizeBytes":   valid["sizeBytes"],
		"title":       valid["title"],
		"url":         "javascript:alert(1)",
	}), ErrFactoryWorkOrderArtifactInvalid)
}
