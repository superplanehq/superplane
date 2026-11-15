package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrganizationDomain is an email domain an organization claims.
type OrganizationDomain struct {
	ID                 uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID     uuid.UUID
	Domain             string
	VerificationToken  string
	VerifiedAt         *time.Time
	VerificationMethod *string
	VerifiedBy         *uuid.UUID
	CreatedBy          *uuid.UUID
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (OrganizationDomain) TableName() string { return "organization_domains" }

func (domain *OrganizationDomain) Verified() bool {
	return domain.VerifiedAt != nil
}

func (domain *OrganizationDomain) Save(tx *gorm.DB) error {
	domain.UpdatedAt = time.Now()
	return tx.Save(domain).Error
}

func ListOrganizationDomains(tx *gorm.DB, orgID uuid.UUID) ([]OrganizationDomain, error) {
	var domains []OrganizationDomain
	err := tx.Where("organization_id = ?", orgID).Order("created_at, id").Find(&domains).Error
	return domains, err
}

func FindOrganizationDomain(tx *gorm.DB, orgID, id uuid.UUID) (*OrganizationDomain, error) {
	var domain OrganizationDomain
	err := tx.Where("organization_id = ? AND id = ?", orgID, id).First(&domain).Error
	if err != nil {
		return nil, err
	}
	return &domain, nil
}

func FindOrganizationDomainByName(tx *gorm.DB, orgID uuid.UUID, name string) (*OrganizationDomain, error) {
	var domain OrganizationDomain
	err := tx.Where("organization_id = ? AND domain = ?", orgID, name).First(&domain).Error
	if err != nil {
		return nil, err
	}
	return &domain, nil
}

func FindVerifiedOrganizationDomain(tx *gorm.DB, name string) (*OrganizationDomain, error) {
	var domain OrganizationDomain
	err := tx.Where("domain = ? AND verified_at IS NOT NULL", name).First(&domain).Error
	if err != nil {
		return nil, err
	}
	return &domain, nil
}

func CountVerifiedOrganizationDomains(tx *gorm.DB, orgID uuid.UUID) (int64, error) {
	var count int64
	err := tx.Model(&OrganizationDomain{}).
		Where("organization_id = ? AND verified_at IS NOT NULL", orgID).
		Count(&count).Error
	return count, err
}

func DeleteOrganizationDomain(tx *gorm.DB, orgID, id uuid.UUID) error {
	return tx.Where("organization_id = ? AND id = ?", orgID, id).Delete(&OrganizationDomain{}).Error
}
