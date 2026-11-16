package sso

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func (s *Service) GetConnection(ctx context.Context, orgID uuid.UUID) (*enterprise.SSOConnection, error) {
	conn, err := models.FindSSOConnectionByOrganization(database.DB(ctx), orgID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &enterprise.SSOConnection{}, nil
	}
	if err != nil {
		return nil, err
	}
	view, err := s.connectionView(conn)
	return view, err
}

func (s *Service) PutConnection(ctx context.Context, orgID, accountID uuid.UUID, input enterprise.SSOConnectionInput) (*enterprise.SSOConnection, error) {
	if err := s.requireLicense(ctx, orgID); err != nil {
		return nil, err
	}
	normalized, err := normalizeInput(input)
	if err != nil {
		return nil, err
	}

	var saved *models.SSOConnection
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := models.FindSSOConnectionByOrganization(tx, orgID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if existing != nil {
			existing, err = models.LockSSOConnection(tx, existing.ID)
			if err != nil {
				return err
			}
		}
		conn := existing
		if conn == nil {
			if strings.TrimSpace(normalized.ClientSecret) == "" {
				return invalid("The client secret is required.")
			}
			conn = &models.SSOConnection{
				ID:             uuid.New(),
				OrganizationID: orgID,
				Protocol:       models.SSOProtocolOIDC,
				Status:         models.SSOStatusDraft,
				CreatedBy:      &accountID,
				CreatedAt:      s.now(),
			}
		}
		previous, err := conn.OIDCConfig()
		if err != nil {
			return err
		}
		if err := s.applyInput(ctx, conn, normalized); err != nil {
			return err
		}
		next, err := conn.OIDCConfig()
		if err != nil {
			return err
		}
		if existing != nil && previous.Issuer != "" && previous.Issuer != normalized.Issuer {
			count, err := models.CountSSOIdentities(tx, conn.ID)
			if err != nil {
				return err
			}
			if count > 0 {
				return precondition("Delete this connection and create a new one before you change the issuer.")
			}
		}
		if previous.Issuer != next.Issuer || previous.ClientID != next.ClientID {
			conn.LastTestedAt = nil
			conn.LastTestResult = nil
			if conn.Status == models.SSOStatusActive {
				conn.Status = models.SSOStatusDraft
			}
		}
		if !conn.IDPLogoutEnabled {
			if err := models.ClearSSOIdentityIDTokens(tx, conn.ID); err != nil {
				return err
			}
		}
		if err := conn.Save(tx); err != nil {
			return err
		}
		saved = conn
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.connectionView(saved)
}

func (s *Service) ActivateConnection(ctx context.Context, orgID uuid.UUID) (*enterprise.SSOConnection, error) {
	if err := s.requireLicense(ctx, orgID); err != nil {
		return nil, err
	}
	current, err := s.connectionForOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	var saved *models.SSOConnection
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		conn, err := models.LockSSOConnection(tx, current.ID)
		if err != nil {
			return err
		}
		result, err := conn.TestResult()
		if err != nil {
			return err
		}
		if conn.LastTestedAt == nil || result == nil || !result.OK {
			return precondition("Test the connection before you activate it.")
		}
		count, err := models.CountVerifiedOrganizationDomains(tx, orgID)
		if err != nil {
			return err
		}
		if count < 1 {
			return precondition("Verify a domain before you activate Single Sign-On.")
		}
		conn.Status = models.SSOStatusActive
		if err := conn.Save(tx); err != nil {
			return err
		}
		saved = conn
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.connectionView(saved)
}

func (s *Service) DisableConnection(ctx context.Context, orgID uuid.UUID) (*enterprise.SSOConnection, error) {
	current, err := s.connectionForOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	var saved *models.SSOConnection
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		conn, err := models.LockSSOConnection(tx, current.ID)
		if err != nil {
			return err
		}
		conn.Status = models.SSOStatusDisabled
		if err := conn.Save(tx); err != nil {
			return err
		}
		saved = conn
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.connectionView(saved)
}

func (s *Service) DeleteConnection(ctx context.Context, orgID uuid.UUID) error {
	conn, err := s.connectionForOrg(ctx, orgID)
	if err != nil {
		return err
	}
	return models.DeleteSSOConnection(database.DB(ctx), conn.ID)
}

func (s *Service) connectionForOrg(ctx context.Context, orgID uuid.UUID) (*models.SSOConnection, error) {
	conn, err := models.FindSSOConnectionByOrganization(database.DB(ctx), orgID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound("This organization has no Single Sign-On connection.")
	}
	return conn, err
}

func normalizeInput(input enterprise.SSOConnectionInput) (enterprise.SSOConnectionInput, error) {
	input.Preset = models.NormalizeSSOPreset(input.Preset)
	if !models.ValidSSOPreset(input.Preset) {
		return input, invalid("The identity provider preset is not valid.")
	}
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.DisplayName == "" {
		return input, invalid("The display name is required.")
	}
	input.Issuer = strings.TrimRight(strings.TrimSpace(input.Issuer), "/")
	if input.Issuer == "" {
		return input, invalid("The issuer URL is required.")
	}
	input.ClientID = strings.TrimSpace(input.ClientID)
	if input.ClientID == "" {
		return input, invalid("The client ID is required.")
	}
	if input.DefaultRole == "" {
		input.DefaultRole = models.RoleOrgOperator
	}
	if !models.ValidSSORole(input.DefaultRole) {
		return input, invalid("The default role is not valid.")
	}
	if input.Enforcement == "" {
		input.Enforcement = models.SSOEnforcementOptional
	}
	if !models.ValidSSOEnforcement(input.Enforcement) {
		return input, invalid("The enforcement mode is not valid.")
	}
	if input.MaxSessionAgeSeconds == 0 {
		input.MaxSessionAgeSeconds = models.SSODefaultSessionAge
	}
	if input.MaxSessionAgeSeconds < models.SSOMinSessionAgeSeconds || input.MaxSessionAgeSeconds > models.SSOMaxSessionAgeSeconds {
		return input, invalid("The session age must be from 15 minutes to 7 days.")
	}
	if len(input.Scopes) == 0 {
		input.Scopes = []string{"openid", "email", "profile"}
	}
	defaults := models.DefaultOIDCConfig(input.Preset)
	if strings.TrimSpace(input.SubjectClaim) == "" {
		input.SubjectClaim = defaults.SubjectClaim
	}
	if strings.TrimSpace(input.EmailClaim) == "" {
		input.EmailClaim = defaults.EmailClaim
	}
	if strings.TrimSpace(input.NameClaim) == "" {
		input.NameClaim = defaults.NameClaim
	}
	return input, nil
}

func (s *Service) applyInput(ctx context.Context, conn *models.SSOConnection, input enterprise.SSOConnectionInput) error {
	cfg := models.OIDCConnectionConfig{
		Issuer:       input.Issuer,
		ClientID:     input.ClientID,
		Scopes:       input.Scopes,
		SubjectClaim: input.SubjectClaim,
		EmailClaim:   input.EmailClaim,
		NameClaim:    input.NameClaim,
	}
	conn.Preset = input.Preset
	conn.DisplayName = input.DisplayName
	conn.Enforcement = input.Enforcement
	conn.AllowOwnerBypass = input.AllowOwnerBypass
	conn.JITEnabled = input.JITEnabled
	conn.DefaultRole = input.DefaultRole
	conn.MaxSessionAgeSeconds = int(input.MaxSessionAgeSeconds)
	conn.IDPLogoutEnabled = input.IDPLogoutEnabled
	if err := conn.SetOIDCConfig(cfg); err != nil {
		return err
	}
	if err := s.adapter.ValidateConfig(ctx, conn); err != nil {
		return invalid(err.Error())
	}
	secret := strings.TrimSpace(input.ClientSecret)
	if secret == "" {
		if len(conn.ClientSecret) == 0 {
			return invalid("The client secret is required.")
		}
		return nil
	}
	encrypted, err := s.Encryptor.Encrypt(ctx, []byte(secret), []byte(models.SSOClientSecretAssociatedData))
	if err != nil {
		return err
	}
	conn.ClientSecret = encrypted
	return nil
}

func (s *Service) connectionView(conn *models.SSOConnection) (*enterprise.SSOConnection, error) {
	cfg, err := conn.OIDCConfig()
	if err != nil {
		return nil, err
	}
	test, err := conn.TestResult()
	if err != nil {
		return nil, err
	}
	var last *enterprise.SSOTestResult
	if test != nil {
		last = &enterprise.SSOTestResult{OK: test.OK, Error: test.Error, Subject: test.Subject, Email: test.Email, Name: test.Name}
	}
	id := conn.ID.String()
	return &enterprise.SSOConnection{
		Exists:                true,
		ID:                    conn.ID,
		OrganizationID:        conn.OrganizationID,
		Protocol:              conn.Protocol,
		Preset:                conn.Preset,
		DisplayName:           conn.DisplayName,
		Status:                conn.Status,
		Enforcement:           conn.Enforcement,
		AllowOwnerBypass:      conn.AllowOwnerBypass,
		JITEnabled:            conn.JITEnabled,
		DefaultRole:           conn.DefaultRole,
		MaxSessionAgeSeconds:  int32(conn.MaxSessionAgeSeconds),
		IDPLogoutEnabled:      conn.IDPLogoutEnabled,
		Issuer:                cfg.Issuer,
		ClientID:              cfg.ClientID,
		Scopes:                cfg.Scopes,
		SubjectClaim:          cfg.SubjectClaim,
		EmailClaim:            cfg.EmailClaim,
		NameClaim:             cfg.NameClaim,
		ClientSecretSet:       len(conn.ClientSecret) > 0,
		LastTestedAt:          conn.LastTestedAt,
		LastTest:              last,
		RedirectURI:           s.BaseURL + "/auth/sso/" + id + "/callback",
		InitiateLoginURI:      s.BaseURL + "/auth/sso/" + id + "/start",
		PostLogoutRedirectURI: s.BaseURL + "/login",
		BackChannelLogoutURI:  s.BaseURL + "/auth/sso/" + id + "/backchannel-logout",
		SCIMBaseURL:           s.BaseURL + "/scim/v2/" + id,
	}, nil
}

func (s *Service) recordTest(ctx context.Context, conn *models.SSOConnection, ok bool, reason, subject, email, name string) error {
	if len(reason) > 180 || strings.Contains(strings.ToLower(reason), "token") || strings.Contains(reason, "eyJ") {
		reason = "The sign-in test failed."
	}
	document := models.SSOTestResultDocument{OK: ok, Error: reason, Subject: subject, Email: email, Name: name}
	raw, err := json.Marshal(document)
	if err != nil {
		return err
	}
	now := s.now()
	if err := models.RecordSSOConnectionTest(database.DB(ctx), conn.ID, now, raw); err != nil {
		return err
	}
	conn.LastTestedAt = &now
	conn.LastTestResult = raw
	return nil
}
