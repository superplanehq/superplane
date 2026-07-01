package organizations

import (
	"github.com/superplanehq/superplane/pkg/authorization"
	"gorm.io/gorm"
)

type mockAuthService struct {
	authorization.Authorization
	Error error
}

func (m *mockAuthService) AssignRole(userID, role, domainID string, domainType string) error {
	if m.Error != nil {
		return m.Error
	}
	return m.Authorization.AssignRole(userID, role, domainID, domainType)
}

func (m *mockAuthService) DestroyOrganization(tx *gorm.DB, orgID string) error {
	if m.Error != nil {
		return m.Error
	}
	return m.Authorization.DestroyOrganization(tx, orgID)
}
