package contexts

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/models"
)

type AuthContext struct {
	orgID             uuid.UUID
	authenticatedUser *models.User
}

func NewAuthContext(orgID uuid.UUID, authenticatedUser *models.User) components.AuthContext {
	return &AuthContext{orgID: orgID, authenticatedUser: authenticatedUser}
}

func (c *AuthContext) AuthenticatedUser() *components.User {
	if c.authenticatedUser == nil {
		return nil
	}

	return &components.User{
		ID:    c.authenticatedUser.ID.String(),
		Name:  c.authenticatedUser.Name,
		Email: c.authenticatedUser.Email,
	}
}

func (c *AuthContext) GetUser(id uuid.UUID) (*components.User, error) {
	user, err := models.FindActiveUserByID(c.orgID.String(), id.String())
	if err != nil {
		return nil, err
	}

	return &components.User{
		ID:    user.ID.String(),
		Name:  user.Name,
		Email: user.Email,
	}, nil
}
