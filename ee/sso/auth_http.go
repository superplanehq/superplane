package sso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/ee/sso/core"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

const loginStateCookie = "sso_login_state"

func (s *Service) AuthHandler() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/auth/sso/discover", s.discover).Methods(http.MethodPost)
	router.HandleFunc("/auth/sso/organizations/{slug}/start", s.organizationStart).Methods(http.MethodGet)
	router.HandleFunc("/auth/sso/{connectionID}/backchannel-logout", s.backchannelLogout).Methods(http.MethodPost)
	router.HandleFunc("/auth/sso/{connectionID}/callback", s.callback).Methods(http.MethodGet)
	router.HandleFunc("/auth/sso/{connectionID}/start", s.start).Methods(http.MethodGet)
	return router
}

func (s *Service) CheckOrganizationAccess(ctx context.Context, user *models.User, session enterprise.SSOAccessSession) (*enterprise.SSORequired, error) {
	decision, err := s.pipeline().CheckOrganizationAccess(ctx, user, authentication.AccountSession{
		AccountID: session.AccountID.String(),
		SSO:       session.SSO,
	})
	if err != nil || decision == nil || !decision.Required {
		return nil, err
	}
	return &enterprise.SSORequired{StartURL: decision.StartURL}, nil
}

func (s *Service) ActiveConnectionExists(ctx context.Context) (bool, error) {
	return models.ActiveSSOConnectionExists(database.DB(ctx))
}

func (s *Service) LogoutRedirect(ctx context.Context, accountID uuid.UUID, session enterprise.SSOAccessSession) (string, bool) {
	sessions := authentication.ParseSSOSessions(session.SSO)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].AuthUnix > sessions[j].AuthUnix
	})
	for _, entry := range sessions {
		conn, err := models.FindSSOConnectionByID(database.DB(ctx), entry.ConnectionID)
		if err != nil || !conn.IDPLogoutEnabled {
			continue
		}
		identity, err := models.FindSSOIdentityByAccount(database.DB(ctx), conn.ID, accountID)
		if err != nil {
			continue
		}
		hint, err := s.pipeline().DecryptIDToken(ctx, identity)
		if err != nil || hint == "" {
			continue
		}
		target, ok, err := s.adapter.BeginLogout(ctx, conn, hint)
		if err != nil || !ok {
			continue
		}
		logging.AddRequestLogFields(ctx, map[string]any{
			"action":            "sso_logout",
			"object":            "sso_connection",
			"outcome":           "success",
			"sso_connection_id": conn.ID.String(),
			"organization_id":   conn.OrganizationID.String(),
		})
		return target, true
	}
	return "", false
}

func (s *Service) discover(w http.ResponseWriter, r *http.Request) {
	if !s.limits.allow(middleware.ClientIP(r)) {
		http.Error(w, "Too many requests.", http.StatusTooManyRequests)
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body)
	response := map[string]any{"ssoAvailable": false}
	domain, ok := core.EmailDomain(body.Email)
	if ok {
		verified, err := models.FindVerifiedOrganizationDomain(database.DB(r.Context()), domain)
		if err == nil {
			conn, err := models.FindActiveSSOConnection(database.DB(r.Context()), verified.OrganizationID)
			if err == nil {
				response["ssoAvailable"] = true
				response["startUrl"] = "/auth/sso/" + conn.ID.String() + "/start"
				response["required"] = conn.Enforcement == models.SSOEnforcementRequired
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Service) organizationStart(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	org, err := models.FindOrganizationByIDOrSlug(database.DB(r.Context()), slug)
	if err != nil {
		s.failLogin(w, r, core.CodeNotAvailable, "organization", nil)
		return
	}
	conn, err := models.FindSSOConnectionByOrganization(database.DB(r.Context()), org.ID)
	if err != nil || conn.Status != models.SSOStatusActive {
		s.failLogin(w, r, core.CodeNotAvailable, "connection", conn)
		return
	}
	target, err := internalStartURL(conn.ID, r.URL.Query())
	if err != nil {
		s.failLogin(w, r, core.CodeFailed, "redirect", conn)
		return
	}
	http.Redirect(w, r, target, http.StatusFound) // #nosec G710 -- the path is an internal SSO route and the query allows only login_hint, redirect, and iss.
}

func internalStartURL(id uuid.UUID, query url.Values) (string, error) {
	parsed, err := url.Parse("/auth/sso/" + id.String() + "/start")
	if err != nil {
		return "", err
	}
	parsed.RawQuery = allowedStartQuery(query).Encode()
	if parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/auth/sso/") {
		return "", errors.New("the start path is not internal")
	}
	return parsed.RequestURI(), nil
}

func allowedStartQuery(query url.Values) url.Values {
	values := url.Values{}
	for _, key := range []string{"login_hint", "redirect", "iss"} {
		if value := query.Get(key); value != "" {
			values.Set(key, value)
		}
	}
	return values
}

func (s *Service) start(w http.ResponseWriter, r *http.Request) {
	conn, ok := s.connectionFromPath(w, r)
	if !ok {
		return
	}
	test := r.URL.Query().Get("test") == "1" || r.URL.Query().Get("test") == "true"
	if test {
		if !s.allowTest(w, r, conn) {
			return
		}
	} else if conn.Status != models.SSOStatusActive {
		s.failLogin(w, r, core.CodeNotAvailable, "status", conn)
		return
	}
	cfg, err := conn.OIDCConfig()
	if err != nil {
		s.failLogin(w, r, core.CodeFailed, "config", conn)
		return
	}
	if iss := strings.TrimRight(r.URL.Query().Get("iss"), "/"); iss != "" && !strings.EqualFold(iss, strings.TrimRight(cfg.Issuer, "/")) {
		s.failLogin(w, r, core.CodeFailed, "issuer", conn)
		return
	}
	redirect := r.URL.Query().Get("redirect")
	if !authentication.IsValidRedirectURL(redirect) {
		redirect = ""
	}
	target, state, err := s.adapter.BeginLogin(r.Context(), conn, core.LoginRequest{
		LoginHint: r.URL.Query().Get("login_hint"),
		Redirect:  redirect,
		Test:      test,
	})
	if err != nil {
		s.failLogin(w, r, core.CodeFailed, "begin", conn)
		return
	}
	if err := s.writeLoginState(w, r, state); err != nil {
		s.failLogin(w, r, core.CodeFailed, "state", conn)
		return
	}
	http.Redirect(w, r, target, http.StatusFound) // #nosec G710 -- target is the authorization URL from the identity provider discovery document.
}

func (s *Service) callback(w http.ResponseWriter, r *http.Request) {
	conn, ok := s.connectionFromPath(w, r)
	if !ok {
		return
	}
	state, ok := s.readLoginState(w, r, conn.ID)
	if !ok {
		s.failLogin(w, r, core.CodeFailed, "state", conn)
		return
	}
	if !state.Test && conn.Status != models.SSOStatusActive {
		s.failLogin(w, r, core.CodeNotAvailable, "status", conn)
		return
	}
	identity, err := s.adapter.HandleCallback(r.Context(), conn, r, state)
	if state.Test {
		s.finishTest(w, r, conn, identity, err)
		return
	}
	if err != nil {
		s.failLogin(w, r, core.CodeFailed, "callback", conn)
		return
	}
	var existing *core.ExistingSession
	if session, err := authentication.AccountSessionFromRequest(r, s.Signer); err == nil {
		if accountID, err := uuid.Parse(session.AccountID); err == nil {
			existing = &core.ExistingSession{AccountID: accountID, SessionStart: session.SessionStart, SSO: session.SSO}
		}
	}
	result, err := s.pipeline().SignIn(r.Context(), conn, *identity, state.Redirect, existing)
	if err != nil {
		code := core.CodeOf(err)
		if code == "" {
			code = core.CodeFailed
		}
		s.failLogin(w, r, code, code, conn)
		return
	}
	if err := authentication.IssueAccountSessionWithClaims(w, r, s.Signer, result.AccountID, result.SessionStart, map[string]string{
		authentication.SSOSessionClaim: result.SSOClaim,
	}); err != nil {
		s.failLogin(w, r, core.CodeFailed, "session", conn)
		return
	}
	logging.AddRequestLogFields(r.Context(), map[string]any{
		"action": "sso_login", "object": "sso_connection", "outcome": "success",
		"sso_connection_id": conn.ID.String(), "organization_id": conn.OrganizationID.String(),
	})
	http.Redirect(w, r, result.Redirect, http.StatusFound)
}

func (s *Service) finishTest(w http.ResponseWriter, r *http.Request, conn *models.SSOConnection, identity *core.ExternalIdentity, err error) {
	outcome := "success"
	query := "ok"
	if err != nil {
		outcome = "failure"
		query = "failed"
		_ = s.recordTest(r.Context(), conn, false, err.Error(), "", "", "")
	} else {
		_ = s.recordTest(r.Context(), conn, true, "", identity.Subject, identity.Email, identity.Name)
	}
	logging.AddRequestLogFields(r.Context(), map[string]any{
		"action": "sso_test", "object": "sso_connection", "outcome": outcome,
		"sso_connection_id": conn.ID.String(), "organization_id": conn.OrganizationID.String(),
	})
	org, orgErr := models.FindOrganizationByIDInTransaction(database.DB(r.Context()), conn.OrganizationID.String())
	target := "/login?sso_test=" + query
	if orgErr == nil {
		target = "/" + org.Slug + "/settings/sso?sso_test=" + query
	}
	http.Redirect(w, r, target, http.StatusFound) // #nosec G710 -- target is an internal settings path built from the organization slug.
}

func (s *Service) backchannelLogout(w http.ResponseWriter, r *http.Request) {
	connectionID, err := uuid.Parse(mux.Vars(r)["connectionID"])
	if err != nil {
		writeBackchannelError(w)
		return
	}
	conn, err := models.FindSSOConnectionByID(database.DB(r.Context()), connectionID)
	if err != nil {
		writeBackchannelError(w)
		return
	}
	_ = r.ParseForm()
	raw := r.Form.Get("logout_token")
	subject, sessionID, err := s.adapter.VerifyLogoutToken(r.Context(), conn, raw)
	if err != nil {
		logging.AddRequestLogFields(r.Context(), map[string]any{
			"action": "sso_backchannel_logout", "outcome": "failure", "sso_failure_reason": "token",
			"sso_connection_id": conn.ID.String(), "organization_id": conn.OrganizationID.String(),
		})
		writeBackchannelError(w)
		return
	}
	identity, err := findLogoutIdentity(r.Context(), conn.ID, subject, sessionID, s.now())
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		writeBackchannelError(w)
		return
	}
	if identity != nil {
		if err := identity.RevokeSessions(database.DB(r.Context()), s.now()); err != nil {
			writeBackchannelError(w)
			return
		}
	}
	fields := map[string]any{
		"action": "sso_backchannel_logout", "object": "sso_identity", "outcome": "success",
		"sso_connection_id": conn.ID.String(), "organization_id": conn.OrganizationID.String(),
	}
	if identity == nil {
		fields["sso_logout_identity_found"] = "false"
	}
	logging.AddRequestLogFields(r.Context(), fields)
	w.WriteHeader(http.StatusOK)
}

func findLogoutIdentity(ctx context.Context, connectionID uuid.UUID, subject, sessionID string, now time.Time) (*models.SSOIdentity, error) {
	if subject != "" {
		identity, err := models.FindSSOIdentityByIDPSubject(database.DB(ctx), connectionID, subject)
		if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) || sessionID == "" {
			return identity, err
		}
	}
	if sessionID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return models.FindSSOIdentityByIDPSession(database.DB(ctx), connectionID, sessionID, now)
}

func (s *Service) connectionFromPath(w http.ResponseWriter, r *http.Request) (*models.SSOConnection, bool) {
	connectionID, err := uuid.Parse(mux.Vars(r)["connectionID"])
	if err != nil {
		s.failLogin(w, r, core.CodeNotAvailable, "connection", nil)
		return nil, false
	}
	conn, err := models.FindSSOConnectionByID(database.DB(r.Context()), connectionID)
	if err != nil {
		s.failLogin(w, r, core.CodeNotAvailable, "connection", nil)
		return nil, false
	}
	return conn, true
}

func (s *Service) allowTest(w http.ResponseWriter, r *http.Request, conn *models.SSOConnection) bool {
	session, err := authentication.AccountSessionFromRequest(r, s.Signer)
	if err != nil {
		s.failLogin(w, r, core.CodeFailed, "test_session", conn)
		return false
	}
	account, err := models.FindAccountByID(session.AccountID)
	if err != nil {
		s.failLogin(w, r, core.CodeFailed, "test_account", conn)
		return false
	}
	user, err := models.FindActiveUserByEmail(conn.OrganizationID.String(), account.Email)
	if err != nil {
		s.failLogin(w, r, core.CodeFailed, "test_member", conn)
		return false
	}
	allowed, err := s.Auth.CheckOrganizationPermission(r.Context(), user.ID.String(), conn.OrganizationID.String(), "sso", "update")
	if err != nil || !allowed {
		s.failLogin(w, r, core.CodeFailed, "test_permission", conn)
		return false
	}
	return true
}

func (s *Service) writeLoginState(w http.ResponseWriter, r *http.Request, state core.LoginState) error {
	test := "0"
	if state.Test {
		test = "1"
	}
	token, err := s.Signer.GenerateWithClaims(10*time.Minute, map[string]string{
		"cid":   state.ConnectionID.String(),
		"st":    state.State,
		"nonce": state.Nonce,
		"vf":    state.Verifier,
		"rd":    state.Redirect,
		"test":  test,
	})
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- local HTTP development must accept the cookie; forwarded HTTPS is detected.
		Name: loginStateCookie, Value: token, Path: "/auth/sso", HttpOnly: true,
		Secure: authentication.RequestUsesHTTPS(r), SameSite: http.SameSiteLaxMode, MaxAge: 600,
	})
	return nil
}

func (s *Service) readLoginState(w http.ResponseWriter, r *http.Request, connectionID uuid.UUID) (core.LoginState, bool) {
	s.clearLoginState(w, r)
	cookie, err := r.Cookie(loginStateCookie)
	if err != nil {
		return core.LoginState{}, false
	}
	claims, err := s.Signer.ValidateAndGetClaims(cookie.Value)
	if err != nil {
		return core.LoginState{}, false
	}
	cid, _ := claims["cid"].(string)
	if cid != connectionID.String() {
		return core.LoginState{}, false
	}
	state, _ := claims["st"].(string)
	nonce, _ := claims["nonce"].(string)
	verifier, _ := claims["vf"].(string)
	redirect, _ := claims["rd"].(string)
	test, _ := claims["test"].(string)
	parsed, err := uuid.Parse(cid)
	if err != nil || state == "" || nonce == "" || verifier == "" {
		return core.LoginState{}, false
	}
	return core.LoginState{
		ConnectionID: parsed,
		State:        state,
		Nonce:        nonce,
		Verifier:     verifier,
		Redirect:     redirect,
		Test:         test == "1",
	}, true
}

func (s *Service) clearLoginState(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- local HTTP development must accept the cookie; forwarded HTTPS is detected.
		Name: loginStateCookie, Value: "", Path: "/auth/sso", HttpOnly: true,
		Secure: authentication.RequestUsesHTTPS(r), SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
}

func (s *Service) failLogin(w http.ResponseWriter, r *http.Request, code, reason string, conn *models.SSOConnection) {
	fields := map[string]any{
		"action": "sso_login", "object": "sso_connection", "outcome": "failure", "sso_failure_reason": reason,
	}
	if conn != nil {
		fields["sso_connection_id"] = conn.ID.String()
		fields["organization_id"] = conn.OrganizationID.String()
	}
	logging.AddRequestLogFields(r.Context(), fields)
	http.Redirect(w, r, "/login?auth_error="+url.QueryEscape(code), http.StatusFound)
}

func writeBackchannelError(w http.ResponseWriter) {
	http.Error(w, "invalid logout token", http.StatusBadRequest)
}

type ipLimiter struct {
	limit rate.Limit
	burst int
	mu    sync.Mutex
	byIP  map[string]*rate.Limiter
}

func newIPLimiter() *ipLimiter {
	return &ipLimiter{
		limit: rate.Limit(10.0 / 60.0),
		burst: 10,
		byIP:  map[string]*rate.Limiter{},
	}
}

func (l *ipLimiter) allow(ip string) bool {
	if ip == "" {
		ip = "unknown"
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.byIP) > 10000 {
		l.byIP = map[string]*rate.Limiter{}
	}
	limiter := l.byIP[ip]
	if limiter == nil {
		limiter = rate.NewLimiter(l.limit, l.burst)
		l.byIP[ip] = limiter
	}
	return limiter.Allow()
}
