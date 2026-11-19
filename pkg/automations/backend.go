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

// Backend checks and manages the executable form of a public definition.
// Apply and Remove write through the supplied transaction. They can return a
// callback for notifications that the service sends after a successful commit.
type Backend interface {
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
