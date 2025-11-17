package authorization2

import (
	org "github.com/superplanehq/superplane/pkg/authorization2/org"
	"gorm.io/gorm"
)

//
// Exposed functions for organization-level authorization.
//

func OrgProvision(tx *gorm.DB, orgID string, ownerID string) error {
	return org.Provision(tx, orgID, ownerID)
}

func OrgVerifier(orgID string, userID string) (*org.Verifier, error) {
	return org.NewVerifier(orgID, userID)
}

func OrgUpdater(orgID string) error {
	panic("not implemented")
}

//
// Exposed functions for canvas-level authorization.
//

func CanvasProvision(tx *gorm.DB, canvasID string, orgID string) error {
	panic("not implemented")
}

func CanvasVerifier(canvasID string, userID string) (any, error) {
	panic("not implemented")
}

func CanvasUpdater(canvasID string) error {
	panic("not implemented")
}
