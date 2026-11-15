package sso

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/models"
)

func (s *Service) AdminGet(ctx context.Context, orgID uuid.UUID) (*enterprise.AdminSSO, error) {
	view := &enterprise.AdminSSO{}
	conn, err := s.GetConnection(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if conn.Exists {
		view.Connection = enterprise.AdminSSOConnection{
			Exists:      true,
			ID:          conn.ID,
			Status:      conn.Status,
			Enforcement: conn.Enforcement,
			DisplayName: conn.DisplayName,
			Protocol:    conn.Protocol,
		}
	}
	rows, err := models.ListOrganizationDomains(database.DB(ctx), orgID)
	if err != nil {
		return nil, err
	}
	view.Domains = make([]enterprise.AdminSSODomain, 0, len(rows))
	for _, row := range rows {
		method := ""
		if row.VerificationMethod != nil {
			method = *row.VerificationMethod
		}
		view.Domains = append(view.Domains, enterprise.AdminSSODomain{
			ID:                 row.ID,
			Domain:             row.Domain,
			Verified:           row.Verified(),
			VerificationMethod: method,
		})
	}
	return view, nil
}

func (s *Service) AdminDisable(ctx context.Context, orgID uuid.UUID) (*enterprise.AdminSSO, error) {
	if _, err := s.DisableConnection(ctx, orgID); err != nil {
		return nil, err
	}
	return s.AdminGet(ctx, orgID)
}
