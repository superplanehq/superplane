package scim

import (
	"net/http"
	"strconv"
	"time"

	"github.com/elimity-com/scim"
	scimerrors "github.com/elimity-com/scim/errors"
	"github.com/elimity-com/scim/optional"
	"github.com/elimity-com/scim/schema"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/utils"
)

// UserHandler implements SCIM /Users for one organization per request (org id in context).
type UserHandler struct {
	Auth authorization.Authorization
}

func (h *UserHandler) orgID(r *http.Request) (string, error) {
	id := OrganizationIDFromContext(r.Context())
	if id == "" {
		return "", scimerrors.ScimErrorBadRequest("missing organization context")
	}
	return id, nil
}

func (h *UserHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	orgID, err := h.orgID(r)
	if err != nil {
		return scim.Resource{}, err
	}

	userName, ok := stringFromAttributes(attributes, "userName")
	if !ok {
		return scim.Resource{}, scimerrors.ScimErrorBadRequest("userName is required")
	}
	email, err := primaryEmail(attributes)
	if err != nil {
		return scim.Resource{}, scimerrors.ScimErrorBadRequest(err.Error())
	}
	if utils.NormalizeEmail(userName) != email {
		return scim.Resource{}, scimerrors.ScimErrorBadRequest("userName must match primary email")
	}

	if !activeBool(attributes, true) {
		return scim.Resource{}, scimerrors.ScimErrorBadRequest("cannot create inactive user")
	}

	var ext *string
	if s, ok := stringFromAttributes(attributes, "externalId"); ok {
		ext = &s
	}

	var out scim.Resource
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var n int64
		if e := tx.Model(&models.User{}).
			Where("organization_id = ? AND email = ?", orgID, email).
			Where("deleted_at IS NULL").
			Count(&n).Error; e != nil {
			return e
		}
		if n > 0 {
			return scimerrors.ScimErrorUniqueness
		}

		if ext != nil {
			var m int64
			if e := tx.Model(&models.OrganizationScimUserMapping{}).
				Where("organization_id = ? AND external_id = ?", orgID, *ext).
				Count(&m).Error; e != nil {
				return e
			}
			if m > 0 {
				return scimerrors.ScimErrorUniqueness
			}
		}

		orgUUID, e := uuid.Parse(orgID)
		if e != nil {
			return scimerrors.ScimErrorBadRequest("invalid organization id")
		}

		name := displayName(attributes, userName)
		account, e := models.CreateManagedAccountInTransaction(tx, name, email, true)
		if e != nil {
			return e
		}

		user, e := models.CreateUserInTransaction(tx, orgUUID, account.ID, email, name)
		if e != nil {
			return e
		}

		if e := models.CreateOrganizationScimUserMappingInTransaction(tx, orgUUID, user.ID, ext); e != nil {
			return e
		}

		if e := h.Auth.AssignRole(user.ID.String(), models.RoleOrgViewer, orgID, models.DomainTypeOrganization); e != nil {
			return e
		}

		now := time.Now()
		out = scim.Resource{
			ID:         user.ID.String(),
			ExternalID: externalIDOptional(ext),
			Attributes: userAttributes(user, email, ext, true),
			Meta: scim.Meta{
				Created:      &now,
				LastModified: &now,
				Version:      strconv.FormatInt(now.Unix(), 10),
			},
		}
		return nil
	})
	if err != nil {
		return scim.Resource{}, err
	}
	return out, nil
}

func (h *UserHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	orgID, err := h.orgID(r)
	if err != nil {
		return scim.Resource{}, err
	}

	user, err := models.FindActiveUserByIDInTransaction(database.Conn(), orgID, id)
	if err != nil {
		return scim.Resource{}, scimerrors.ScimErrorResourceNotFound(id)
	}
	if user.IsServiceAccount() {
		return scim.Resource{}, scimerrors.ScimErrorResourceNotFound(id)
	}

	mapping, mErr := models.FindScimMappingByOrganizationAndUserID(database.Conn(), orgID, id)
	if mErr != nil {
		return scim.Resource{}, scimerrors.ScimErrorResourceNotFound(id)
	}

	email := user.GetEmail()
	return scim.Resource{
		ID:         user.ID.String(),
		ExternalID: externalIDOptional(mapping.ExternalID),
		Attributes: userAttributes(user, email, mapping.ExternalID, true),
		Meta:       scimMeta(user),
	}, nil
}

func (h *UserHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	orgID, err := h.orgID(r)
	if err != nil {
		return scim.Page{}, err
	}

	if params.FilterValidator != nil {
		return scim.Page{}, scimerrors.ScimErrorInvalidFilter
	}

	ids, err := models.ListUserIDsWithScimMappingInOrganization(database.Conn(), orgID)
	if err != nil {
		return scim.Page{}, err
	}

	total := len(ids)
	start := params.StartIndex
	if start < 1 {
		start = 1
	}
	count := params.Count
	if count <= 0 {
		count = 100
	}

	resources := make([]scim.Resource, 0)
	for i := start - 1; i < total && len(resources) < count; i++ {
		id := ids[i].String()
		res, err := h.Get(r, id)
		if err != nil {
			continue
		}
		resources = append(resources, res)
	}

	return scim.Page{
		TotalResults: total,
		Resources:    resources,
	}, nil
}

func (h *UserHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	orgID, err := h.orgID(r)
	if err != nil {
		return scim.Resource{}, err
	}

	user, err := models.FindActiveUserByIDInTransaction(database.Conn(), orgID, id)
	if err != nil {
		return scim.Resource{}, scimerrors.ScimErrorResourceNotFound(id)
	}
	if user.IsServiceAccount() {
		return scim.Resource{}, scimerrors.ScimErrorResourceNotFound(id)
	}

	_, err = models.FindScimMappingByOrganizationAndUserID(database.Conn(), orgID, id)
	if err != nil {
		return scim.Resource{}, scimerrors.ScimErrorResourceNotFound(id)
	}

	userName, ok := stringFromAttributes(attributes, "userName")
	if !ok {
		return scim.Resource{}, scimerrors.ScimErrorBadRequest("userName is required")
	}
	email, err := primaryEmail(attributes)
	if err != nil {
		return scim.Resource{}, scimerrors.ScimErrorBadRequest(err.Error())
	}
	if utils.NormalizeEmail(userName) != email {
		return scim.Resource{}, scimerrors.ScimErrorBadRequest("userName must match primary email")
	}

	name := displayName(attributes, userName)
	active := activeBool(attributes, true)

	var ext *string
	if s, ok := stringFromAttributes(attributes, "externalId"); ok {
		ext = &s
	}

	var out scim.Resource
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		if user.AccountID == nil {
			return scimerrors.ScimErrorInternal
		}
		var account models.Account
		if e := tx.Where("id = ?", *user.AccountID).First(&account).Error; e != nil {
			return e
		}

		normalizedEmail := utils.NormalizeEmail(email)
		if e := tx.Model(&account).Updates(map[string]interface{}{
			"email":      normalizedEmail,
			"name":       name,
			"updated_at": time.Now(),
		}).Error; e != nil {
			return e
		}

		if e := tx.Model(&models.User{}).
			Where("account_id = ?", account.ID).
			Updates(map[string]interface{}{"email": normalizedEmail, "updated_at": time.Now()}).Error; e != nil {
			return e
		}

		if e := tx.Model(user).Updates(map[string]interface{}{
			"name":       name,
			"email":      normalizedEmail,
			"updated_at": time.Now(),
		}).Error; e != nil {
			return e
		}

		updates := map[string]interface{}{"updated_at": time.Now()}
		if ext != nil {
			updates["external_id"] = *ext
		}
		if e := tx.Model(&models.OrganizationScimUserMapping{}).
			Where("organization_id = ? AND user_id = ?", orgID, id).
			Updates(updates).Error; e != nil {
			return e
		}

		if !active {
			if e := models.DeleteOrganizationScimUserMappingInTransaction(tx, orgID, id); e != nil {
				return e
			}
			if e := user.SoftDeleteInTransaction(tx); e != nil {
				return e
			}
			now := time.Now()
			out = scim.Resource{
				ID:         id,
				ExternalID: externalIDOptional(ext),
				Attributes: userAttributes(user, normalizedEmail, ext, false),
				Meta: scim.Meta{
					LastModified: &now,
					Version:      strconv.FormatInt(now.Unix(), 10),
				},
			}
			return nil
		}

		u2, e := models.FindActiveUserByIDInTransaction(tx, orgID, id)
		if e != nil {
			return e
		}
		mapping, e := models.FindScimMappingByOrganizationAndUserID(tx, orgID, id)
		if e != nil {
			return e
		}

		out = scim.Resource{
			ID:         id,
			ExternalID: externalIDOptional(mapping.ExternalID),
			Attributes: userAttributes(u2, u2.GetEmail(), mapping.ExternalID, true),
			Meta:       scimMeta(u2),
		}
		return nil
	})
	if err != nil {
		return scim.Resource{}, err
	}
	return out, nil
}

func (h *UserHandler) Delete(r *http.Request, id string) error {
	orgID, err := h.orgID(r)
	if err != nil {
		return err
	}

	user, err := models.FindActiveUserByIDInTransaction(database.Conn(), orgID, id)
	if err != nil {
		return scimerrors.ScimErrorResourceNotFound(id)
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if e := models.DeleteOrganizationScimUserMappingInTransaction(tx, orgID, id); e != nil {
			return e
		}
		return user.SoftDeleteInTransaction(tx)
	})
}

func (h *UserHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	orgID, err := h.orgID(r)
	if err != nil {
		return scim.Resource{}, err
	}

	user, err := models.FindActiveUserByIDInTransaction(database.Conn(), orgID, id)
	if err != nil {
		return scim.Resource{}, scimerrors.ScimErrorResourceNotFound(id)
	}

	_, err = models.FindScimMappingByOrganizationAndUserID(database.Conn(), orgID, id)
	if err != nil {
		return scim.Resource{}, scimerrors.ScimErrorResourceNotFound(id)
	}

	for _, op := range operations {
		if op.Op != scim.PatchOperationReplace || op.Path == nil {
			continue
		}
		if op.Path.String() == "active" {
			active, ok := op.Value.(bool)
			if !ok {
				return scim.Resource{}, scimerrors.ScimErrorInvalidValue
			}
			if !active {
				err := database.Conn().Transaction(func(tx *gorm.DB) error {
					if e := models.DeleteOrganizationScimUserMappingInTransaction(tx, orgID, id); e != nil {
						return e
					}
					return user.SoftDeleteInTransaction(tx)
				})
				if err != nil {
					return scim.Resource{}, err
				}
				return scim.Resource{}, nil
			}
		}
	}

	return h.Get(r, user.ID.String())
}

func userAttributes(user *models.User, email string, externalID *string, active bool) scim.ResourceAttributes {
	attrs := scim.ResourceAttributes{
		"schemas":     []interface{}{schema.UserSchema},
		"userName":    email,
		"name":        map[string]interface{}{"formatted": user.Name},
		"displayName": user.Name,
		"active":      active,
		"emails": []interface{}{
			map[string]interface{}{"value": email, "primary": true},
		},
	}
	if externalID != nil && *externalID != "" {
		attrs["externalId"] = *externalID
	}
	return attrs
}

func scimMeta(user *models.User) scim.Meta {
	ver := strconv.FormatInt(user.UpdatedAt.Unix(), 10)
	return scim.Meta{
		Created:      &user.CreatedAt,
		LastModified: &user.UpdatedAt,
		Version:      ver,
	}
}

func externalIDOptional(ext *string) optional.String {
	if ext == nil || *ext == "" {
		return optional.String{}
	}
	return optional.NewString(*ext)
}
