package registry

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	DefaultArtifactsRootDirectory = "/tmp/superplane-artifacts"
	ArtifactsRootDirectoryEnvVar  = "ARTIFACTS_ROOT_DIRECTORY"

	ArtifactResourceTypeCanvas    = "canvas"
	ArtifactResourceTypeNode      = "node"
	ArtifactResourceTypeExecution = "execution"
)

func ArtifactsRootDirectory() string {
	rootDirectory := os.Getenv(ArtifactsRootDirectoryEnvVar)
	if rootDirectory == "" {
		return DefaultArtifactsRootDirectory
	}

	return rootDirectory
}

func NewLocalArtifactStorage(resourceType string, resourceID string) *LocalArtifactStorage {
	return &LocalArtifactStorage{
		RootDirectory: ArtifactsRootDirectory(),
		ResourceType:  resourceType,
		ResourceID:    resourceID,
	}
}

func NewLocalArtifactStorageContext(workflowID string, nodeID string, executionID string) core.ArtifactStorageContext {
	return core.ArtifactStorageContext{
		Canvas:    NewLocalArtifactStorage(ArtifactResourceTypeCanvas, workflowID),
		Node:      NewLocalArtifactStorage(ArtifactResourceTypeNode, NodeArtifactResourceID(workflowID, nodeID)),
		Execution: NewLocalArtifactStorage(ArtifactResourceTypeExecution, executionID),
	}
}

func NodeArtifactResourceID(workflowID string, nodeID string) string {
	return fmt.Sprintf("%s:%s", workflowID, nodeID)
}
