package contexts

import (
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/models"
)

type UserContext struct {
	user *models.User
}

func NewUserContext(user *models.User) components.UserContext {
	return &UserContext{user: user}
}

func (c *UserContext) Get() components.User {
	return components.User{
		ID:    c.user.ID.String(),
		Name:  c.user.Name,
		Email: c.user.Email,
	}
}
