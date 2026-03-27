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
	log "github.com/sirupsen/logrus"
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

	log.Infof("SCIM [%s] Create: provisioning user email=%s externalID=%v", orgID, email, ext)

	var out scim.Resource
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		// If a user with this email already exists, just create a SCIM mapping for them
		// rather than erroring — this handles pre-existing users (e.g. the org creator).
		existingUser, lookupErr := models.FindActiveUserByEmailInTransaction(tx, orgID, email)
		if lookupErr == nil && existingUser != nil {
			log.Infof("SCIM [%s] Create: existing user found id=%s, linking SCIM mapping", orgID, existingUser.ID)
			var mappingCount int64
			if e := tx.Model(&models.OrganizationScimUserMapping{}).
				Where("organization_id = ? AND user_id = ?", orgID, existingUser.ID).
				Count(&mappingCount).Error; e != nil {
				return e
			}
			if mappingCount > 0 {
				log.Warnf("SCIM [%s] Create: user id=%s already has a SCIM mapping, returning uniqueness error", orgID, existingUser.ID)
				return scimerrors.ScimErrorUniqueness
			}
			orgUUID, e := uuid.Parse(orgID)
			if e != nil {
				return scimerrors.ScimErrorBadRequest("invalid organization id")
			}
			if e := models.CreateOrganizationScimUserMappingInTransaction(tx, orgUUID, existingUser.ID, ext); e != nil {
				log.Errorf("SCIM [%s] Create: failed to create SCIM mapping for existing user id=%s: %v", orgID, existingUser.ID, e)
				return e
			}
			log.Infof("SCIM [%s] Create: linked SCIM mapping for existing user id=%s", orgID, existingUser.ID)
			now := time.Now()
			out = scim.Resource{
				ID:         existingUser.ID.String(),
				ExternalID: externalIDOptional(ext),
				Attributes: userAttributes(existingUser, email, ext, true),
				Meta: scim.Meta{
					Created:      &now,
					LastModified: &now,
					Version:      strconv.FormatInt(now.Unix(), 10),
				},
			}
			return nil
		}

		if ext != nil {
			var m int64
			if e := tx.Model(&models.OrganizationScimUserMapping{}).
				Where("organization_id = ? AND external_id = ?", orgID, *ext).
				Count(&m).Error; e != nil {
				return e
			}
			if m > 0 {
				log.Warnf("SCIM [%s] Create: externalId already mapped, returning uniqueness error", orgID)
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
			log.Errorf("SCIM [%s] Create: failed to create account for email=%s: %v", orgID, email, e)
			return e
		}

		user, e := models.CreateUserInTransaction(tx, orgUUID, account.ID, email, name)
		if e != nil {
			log.Errorf("SCIM [%s] Create: failed to create user for email=%s: %v", orgID, email, e)
			return e
		}

		if e := models.CreateOrganizationScimUserMappingInTransaction(tx, orgUUID, user.ID, ext); e != nil {
			log.Errorf("SCIM [%s] Create: failed to create SCIM mapping for new user id=%s: %v", orgID, user.ID, e)
			return e
		}

		if e := h.Auth.AssignRole(user.ID.String(), models.RoleOrgViewer, orgID, models.DomainTypeOrganization); e != nil {
			log.Errorf("SCIM [%s] Create: failed to assign role for user id=%s: %v", orgID, user.ID, e)
			return e
		}

		log.Infof("SCIM [%s] Create: successfully provisioned user id=%s email=%s", orgID, user.ID, email)
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
		log.Errorf("SCIM [%s] Create: transaction failed for email=%s: %v", orgID, email, err)
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

	mapping, _ := models.FindScimMappingByOrganizationAndUserID(database.Conn(), orgID, id)

	email := user.GetEmail()
	var extID *string
	if mapping != nil {
		extID = mapping.ExternalID
	}
	return scim.Resource{
		ID:         user.ID.String(),
		ExternalID: externalIDOptional(extID),
		Attributes: userAttributes(user, email, extID, true),
		Meta:       scimMeta(user),
	}, nil
}

func (h *UserHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	orgID, err := h.orgID(r)
	if err != nil {
		return scim.Page{}, err
	}

	rows, err := models.ListUsersWithScimMappingInOrganization(database.Conn(), orgID)
	if err != nil {
		return scim.Page{}, err
	}

	total := len(rows)
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
		row := rows[i]
		email := row.GetEmail()
		resources = append(resources, scim.Resource{
			ID:         row.ID.String(),
			ExternalID: externalIDOptional(row.ExternalID),
			Attributes: userAttributes(&row.User, email, row.ExternalID, true),
			Meta:       scimMeta(&row.User),
		})
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
			Where("account_id = ? AND organization_id = ?", account.ID, orgID).
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
		log.Warnf("SCIM [%s] Delete: user id=%s not found", orgID, id)
		return scimerrors.ScimErrorResourceNotFound(id)
	}

	log.Infof("SCIM [%s] Delete: deprovisioning user id=%s email=%s", orgID, id, user.Email)
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if e := models.DeleteOrganizationScimUserMappingInTransaction(tx, orgID, id); e != nil {
			log.Errorf("SCIM [%s] Delete: failed to delete SCIM mapping for user id=%s: %v", orgID, id, e)
			return e
		}
		if e := user.SoftDeleteInTransaction(tx); e != nil {
			log.Errorf("SCIM [%s] Delete: failed to soft-delete user id=%s: %v", orgID, id, e)
			return e
		}
		log.Infof("SCIM [%s] Delete: successfully deprovisioned user id=%s email=%s", orgID, id, user.Email)
		return nil
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
