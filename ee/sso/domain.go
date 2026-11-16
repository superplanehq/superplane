package sso

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/models"
	"golang.org/x/net/idna"
	"gorm.io/gorm"
)

const domainVerificationPrefix = "superplane-domain-verification="

var publicEmailDomains = map[string]struct{}{
	"gmail.com": {}, "googlemail.com": {}, "outlook.com": {}, "hotmail.com": {},
	"live.com": {}, "yahoo.com": {}, "icloud.com": {}, "me.com": {},
	"proton.me": {}, "protonmail.com": {}, "aol.com": {}, "gmx.com": {},
	"yandex.com": {}, "mail.com": {},
}

// NormalizeDomain returns a lowercase punycode domain, or an error an admin can read.
func NormalizeDomain(raw string) (string, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	value = strings.Trim(value, ".")
	if value == "" || strings.ContainsAny(value, "/@ \\") || strings.Contains(value, "..") {
		return "", invalid("This domain is not valid.")
	}
	ascii, err := idna.Lookup.ToASCII(value)
	if err != nil || ascii == "" || !strings.Contains(ascii, ".") {
		return "", invalid("This domain is not valid.")
	}
	if _, ok := publicEmailDomains[ascii]; ok {
		return "", invalid("This email domain is public. Use a domain that your organization owns.")
	}
	return ascii, nil
}

func txtRecordName(domain string) string {
	return "_superplane-verification." + domain
}

func txtRecordValue(token string) string {
	return domainVerificationPrefix + token
}

func (s *Service) ListDomains(ctx context.Context, orgID uuid.UUID) ([]enterprise.SSODomain, error) {
	rows, err := models.ListOrganizationDomains(database.DB(ctx), orgID)
	if err != nil {
		return nil, err
	}
	views := make([]enterprise.SSODomain, 0, len(rows))
	for _, row := range rows {
		views = append(views, domainView(row))
	}
	return views, nil
}

func (s *Service) CreateDomain(ctx context.Context, orgID, accountID uuid.UUID, domain string) (*enterprise.SSODomain, error) {
	if err := s.requireLicense(ctx, orgID); err != nil {
		return nil, err
	}
	normalized, err := NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	row := &models.OrganizationDomain{
		ID:                uuid.New(),
		OrganizationID:    orgID,
		Domain:            normalized,
		VerificationToken: token,
		CreatedBy:         &accountID,
		CreatedAt:         s.now(),
		UpdatedAt:         s.now(),
	}
	if err := row.Save(database.DB(ctx)); err != nil {
		if isUnique(err) {
			return nil, conflict("This organization already has this domain.")
		}
		return nil, err
	}
	view := domainView(*row)
	return &view, nil
}

func (s *Service) VerifyDomain(ctx context.Context, orgID, domainID uuid.UUID) (*enterprise.SSODomain, error) {
	if err := s.requireLicense(ctx, orgID); err != nil {
		return nil, err
	}
	return s.markDomainVerified(ctx, orgID, domainID, uuid.Nil, models.DomainVerificationDNS, true)
}

func (s *Service) AdminVerifyDomain(ctx context.Context, orgID, domainID, accountID uuid.UUID) (*enterprise.SSODomain, error) {
	return s.markDomainVerified(ctx, orgID, domainID, accountID, models.DomainVerificationAdmin, false)
}

func (s *Service) markDomainVerified(ctx context.Context, orgID, domainID, accountID uuid.UUID, method string, lookup bool) (*enterprise.SSODomain, error) {
	row, err := models.FindOrganizationDomain(database.DB(ctx), orgID, domainID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound("This domain does not exist.")
	}
	if err != nil {
		return nil, err
	}
	if lookup {
		timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		records, err := s.Lookup.LookupTXT(timeout, txtRecordName(row.Domain))
		if err != nil || !txtMatches(records, row.VerificationToken) {
			return nil, invalid("The DNS TXT record does not match.")
		}
	}
	var saved models.OrganizationDomain
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockOrganizationDomain(tx, orgID, domainID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return notFound("This domain does not exist.")
		}
		if err != nil {
			return err
		}
		held, err := models.FindVerifiedOrganizationDomain(tx, locked.Domain)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if held != nil && held.OrganizationID != orgID {
			return conflict("Another organization already verified this domain.")
		}
		now := s.now()
		locked.VerifiedAt = &now
		locked.VerificationMethod = &method
		if accountID != uuid.Nil {
			locked.VerifiedBy = &accountID
		}
		if err := locked.MarkVerified(tx); err != nil {
			if isUnique(err) {
				return conflict("Another organization already verified this domain.")
			}
			return err
		}
		saved = *locked
		return nil
	})
	if err != nil {
		return nil, err
	}
	view := domainView(saved)
	return &view, nil
}

func (s *Service) DeleteDomain(ctx context.Context, orgID, domainID uuid.UUID) error {
	return database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := models.FindOrganizationDomain(tx, orgID, domainID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return notFound("This domain does not exist.")
		}
		if err != nil {
			return err
		}
		conn, err := models.FindSSOConnectionByOrganization(tx, orgID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			conn, err = models.LockSSOConnection(tx, conn.ID)
			if err != nil {
				return err
			}
		}
		if row.Verified() && conn != nil && conn.Status == models.SSOStatusActive {
			count, err := models.CountVerifiedOrganizationDomains(tx, orgID)
			if err != nil {
				return err
			}
			if count <= 1 {
				return precondition("Disable Single Sign-On before you delete the last verified domain.")
			}
		}
		return models.DeleteOrganizationDomain(tx, orgID, domainID)
	})
}

func domainView(row models.OrganizationDomain) enterprise.SSODomain {
	method := ""
	if row.VerificationMethod != nil {
		method = *row.VerificationMethod
	}
	return enterprise.SSODomain{
		ID:                 row.ID,
		Domain:             row.Domain,
		VerificationToken:  row.VerificationToken,
		TXTName:            txtRecordName(row.Domain),
		TXTValue:           txtRecordValue(row.VerificationToken),
		VerifiedAt:         row.VerifiedAt,
		VerificationMethod: method,
	}
}

func txtMatches(records []string, token string) bool {
	want := txtRecordValue(token)
	for _, record := range records {
		if strings.TrimSpace(record) == want {
			return true
		}
	}
	return false
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
