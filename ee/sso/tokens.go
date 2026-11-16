package sso

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func (s *Service) ListSCIMTokens(ctx context.Context, orgID uuid.UUID) ([]enterprise.SCIMToken, error) {
	conn, err := s.connectionForOrg(ctx, orgID)
	if status.Code(err) == codes.NotFound {
		return []enterprise.SCIMToken{}, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := models.ListActiveSCIMTokens(database.DB(ctx), conn.ID)
	if err != nil {
		return nil, err
	}
	tokens := make([]enterprise.SCIMToken, 0, len(rows))
	for _, row := range rows {
		tokens = append(tokens, enterprise.SCIMToken{
			ID:          row.ID,
			Description: row.Description,
			CreatedAt:   row.CreatedAt,
			LastUsedAt:  row.LastUsedAt,
		})
	}
	return tokens, nil
}

func (s *Service) CreateSCIMToken(ctx context.Context, orgID, accountID uuid.UUID, description string) (*enterprise.CreatedSCIMToken, error) {
	if err := s.requireLicense(ctx, orgID); err != nil {
		return nil, err
	}
	conn, err := s.connectionForOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	secret, err := newSCIMSecret()
	if err != nil {
		return nil, err
	}
	row := &models.SCIMToken{
		ID:           uuid.New(),
		ConnectionID: conn.ID,
		TokenHash:    crypto.HashToken(secret),
		Description:  description,
		CreatedBy:    &accountID,
		CreatedAt:    s.now(),
	}
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := models.CreateSCIMToken(tx, row); err != nil {
			if errors.Is(err, models.ErrSCIMTokenLimit) {
				return conflict("This organization already has five active SCIM tokens.")
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &enterprise.CreatedSCIMToken{
		Token: enterprise.SCIMToken{
			ID:          row.ID,
			Description: row.Description,
			CreatedAt:   row.CreatedAt,
		},
		Secret: secret,
	}, nil
}

func (s *Service) RevokeSCIMToken(ctx context.Context, orgID, tokenID uuid.UUID) error {
	conn, err := s.connectionForOrg(ctx, orgID)
	if err != nil {
		return err
	}
	token, err := models.FindActiveSCIMToken(database.DB(ctx), conn.ID, tokenID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return notFound("This SCIM token does not exist.")
	}
	if err != nil {
		return err
	}
	return token.Revoke(database.DB(ctx), s.now())
}

func newSCIMSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "sp_scim_" + hex.EncodeToString(buf), nil
}
