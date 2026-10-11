package mcpserver

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func IsAPIToken(token string) bool {
	return strings.HasPrefix(token, models.MCPAPITokenPrefix)
}

func ClaimsForAPIToken(tx *gorm.DB, rawToken, resource string) (*AccessClaims, uuid.UUID, error) {
	stored, err := models.FindMCPAPITokenByHash(tx, crypto.HashToken(rawToken))
	if err != nil {
		return nil, uuid.Nil, err
	}
	if strings.TrimSpace(stored.Resource) != strings.TrimSpace(resource) {
		return nil, uuid.Nil, models.ErrMCPAPITokenNotFound
	}
	if _, err := models.FindFactory(tx, stored.OrganizationID, stored.FactoryID); err != nil {
		if errors.Is(err, models.ErrFactoryNotFound) {
			return nil, uuid.Nil, models.ErrMCPAPITokenNotFound
		}
		return nil, uuid.Nil, err
	}
	user, err := models.FindActiveUserByIDInTransaction(tx, stored.OrganizationID.String(), stored.UserID.String())
	if err != nil || !user.IsHuman() {
		return nil, uuid.Nil, models.ErrMCPAPITokenNotFound
	}
	if UserAccountBlocked(tx, stored.UserID) {
		return nil, uuid.Nil, models.ErrMCPAPITokenNotFound
	}
	return &AccessClaims{
		UserID:    stored.UserID,
		OrgID:     stored.OrganizationID,
		FactoryID: stored.FactoryID,
		ClientID:  stored.ID.String(),
		Resource:  stored.Resource,
		Scopes:    append([]string{}, stored.Scopes...),
	}, stored.ID, nil
}
