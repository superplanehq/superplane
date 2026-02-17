package contexts

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type AuthContext struct {
	tx                *gorm.DB
	orgID             uuid.UUID
	authService       authorization.Authorization
	authenticatedUser *models.User
}

func NewAuthContext(tx *gorm.DB, orgID uuid.UUID, authService authorization.Authorization, authenticatedUser *models.User) *AuthContext {
	return &AuthContext{
		tx:                tx,
		orgID:             orgID,
		authService:       authService,
		authenticatedUser: authenticatedUser,
	}
}

func (c *AuthContext) AuthenticatedUser() *core.User {
	if c.authenticatedUser == nil {
		return nil
	}

	return &core.User{
		ID:    c.authenticatedUser.ID.String(),
		Name:  c.authenticatedUser.Name,
		Email: c.authenticatedUser.GetEmail(),
	}
}

func (c *AuthContext) GetUser(id uuid.UUID) (*core.User, error) {
	user, err := models.FindActiveUserByIDInTransaction(c.tx, c.orgID.String(), id.String())
	if err != nil {
		return nil, err
	}

	return &core.User{
		ID:    user.ID.String(),
		Name:  user.Name,
		Email: user.GetEmail(),
	}, nil
}

func (c *AuthContext) HasRole(role string) (bool, error) {
	if c.authenticatedUser == nil {
		return false, fmt.Errorf("user not authenticated")
	}

	roles, err := c.authService.GetUserRolesForOrg(c.authenticatedUser.ID.String(), c.orgID.String())
	if err != nil {
		return false, fmt.Errorf("error finding users for role %s: %v", role, err)
	}

	for _, r := range roles {
		if r.Name == role {
			return true, nil
		}
	}

	return false, nil
}

func (c *AuthContext) InGroup(group string) (bool, error) {
	if c.authenticatedUser == nil {
		return false, fmt.Errorf("user not authenticated")
	}

	userIDs, err := c.authService.GetGroupUsers(c.orgID.String(), models.DomainTypeOrganization, group)
	if err != nil {
		return false, fmt.Errorf("error finding users in group %s: %v", group, err)
	}

	if slices.Contains(userIDs, c.authenticatedUser.ID.String()) {
		return true, nil
	}

	return false, nil
}
