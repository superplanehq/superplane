package automations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"gorm.io/gorm"
)

type Placement struct {
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	Stage          string
}

type Artifact struct {
	ID       string
	Revision string
}

// Check validates a new automation for the destination. Apply validates and
// publishes against the current artifact when updating an existing automation.
// Apply and Remove write through the supplied transaction. They can return a
// callback for notifications that the service sends after a successful commit.
type Backend interface {
	ArtifactStages(*gorm.DB, Placement, []string) (map[string]string, error)
	Check(context.Context, *gorm.DB, Placement, *automationspec.Document) error
	Apply(context.Context, *gorm.DB, Placement, uuid.UUID, uuid.UUID, Artifact, *automationspec.Document) (Artifact, func(), error)
	Remove(context.Context, *gorm.DB, Placement, Artifact) (func(), error)
}

type LegacyAutomation struct {
	ID        uuid.UUID
	Placement Placement
	Name      string
	Artifact  Artifact
}

// LegacyImporter is optional. Backends that only handle v2 definitions do
// not need to read Canvas data.
type LegacyImporter interface {
	ListLegacy(context.Context, *gorm.DB, Placement) ([]LegacyAutomation, error)
	FindLegacy(context.Context, *gorm.DB, uuid.UUID, uuid.UUID) (LegacyAutomation, error)
	ImportLegacy(context.Context, *gorm.DB, LegacyAutomation) (string, error)
}
