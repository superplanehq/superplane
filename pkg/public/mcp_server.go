package public

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/mcpserver"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

func (s *Server) mcpOrigin(r *http.Request) string {
	return mcpserver.PublicOrigin(r, s.mcpOAuthBaseURL())
}

func (s *Server) handleMCPProtectedResource(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, mcpserver.ProtectedResourceMetadata(s.mcpOrigin(r)))
}

func (s *Server) handleMCPAuthorizationServer(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, mcpserver.AuthorizationServerMetadata(s.mcpOrigin(r)))
}

func (s *Server) handleMCPAuthorize(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetEffectiveAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	origin := s.mcpOrigin(r)
	resource := mcpserver.ResourceURL(origin)
	httpClient := s.mcpHTTPClient()
	db := database.DB(r.Context())

	if r.Method == http.MethodGet {
		req, oauthErr := mcpserver.ParseAuthorizeRequest(r.URL.Query(), resource)
		if oauthErr != nil {
			http.Error(w, oauthErr.Description, oauthErr.Status)
			return
		}
		client, oauthErr := mcpserver.ResolveClient(r.Context(), db, httpClient, req.ClientID, req.RedirectURI)
		if oauthErr != nil {
			http.Error(w, oauthErr.Description, oauthErr.Status)
			return
		}
		consent, err := mcpserver.MintConsentToken(s.jwt, mcpserver.ConsentClaims{
			ClientID:            req.ClientID,
			RedirectURI:         req.RedirectURI,
			State:               req.State,
			CodeChallenge:       req.CodeChallenge,
			CodeChallengeMethod: req.CodeChallengeMethod,
			Resource:            req.Resource,
			ResponseType:        req.ResponseType,
		})
		if err != nil {
			http.Error(w, "failed to start authorization", http.StatusInternalServerError)
			return
		}
		s.writeConsentPage(w, r, account, client.Name, consent, "", http.StatusOK)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	consent, err := mcpserver.ParseConsentToken(s.jwt, r.FormValue("consent"))
	if err != nil {
		http.Error(w, "authorization request expired", http.StatusBadRequest)
		return
	}
	client, oauthErr := mcpserver.ResolveClient(r.Context(), db, httpClient, consent.ClientID, consent.RedirectURI)
	if oauthErr != nil {
		http.Error(w, oauthErr.Description, oauthErr.Status)
		return
	}
	userID, orgID, factoryID, err := mcpserver.ResolveConsentWorkspace(r.Context(), account, r.FormValue("factory_id"))
	if err != nil {
		token, mintErr := mcpserver.MintConsentToken(s.jwt, *consent)
		if mintErr != nil {
			http.Error(w, "failed to start authorization", http.StatusInternalServerError)
			return
		}
		s.writeConsentPage(w, r, account, client.Name, token, "Choose a workspace this account can use.", http.StatusBadRequest)
		return
	}

	code, err := mcpserver.CreateAuthorizationCode(db, &mcpserver.AuthorizeRequest{
		ClientID:            consent.ClientID,
		RedirectURI:         consent.RedirectURI,
		State:               consent.State,
		CodeChallenge:       consent.CodeChallenge,
		CodeChallengeMethod: consent.CodeChallengeMethod,
		Resource:            consent.Resource,
		ResponseType:        consent.ResponseType,
	}, userID, orgID, factoryID)
	if err != nil {
		http.Error(w, "failed to create authorization code", http.StatusInternalServerError)
		return
	}
	redirect, err := mcpserver.AuthorizationRedirect(consent.RedirectURI, code, consent.State)
	if err != nil {
		http.Error(w, "redirect_uri is not valid", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, redirect, http.StatusFound)
}

func (s *Server) writeConsentPage(
	w http.ResponseWriter,
	r *http.Request,
	account *models.Account,
	clientName, consent, errMessage string,
	status int,
) {
	workspaces, err := mcpserver.ListConsentWorkspaces(r.Context(), account)
	if err != nil {
		http.Error(w, "failed to load workspaces", http.StatusInternalServerError)
		return
	}
	page, err := mcpserver.RenderConsentPage(clientName, consent, errMessage, workspaces)
	if err != nil {
		http.Error(w, "failed to render authorization page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(page)
}

func (s *Server) handleMCPToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, &mcpserver.OAuthError{Code: "invalid_request", Description: "form is not valid", Status: http.StatusBadRequest})
		return
	}
	resource := mcpserver.ResourceURL(s.mcpOrigin(r))
	db := database.DB(r.Context())

	var issue *mcpserver.TokenIssue
	var refresh string
	var oauthErr *mcpserver.OAuthError
	switch r.FormValue("grant_type") {
	case "authorization_code":
		txErr := db.Transaction(func(tx *gorm.DB) error {
			var inner *mcpserver.OAuthError
			issue, refresh, inner = mcpserver.ExchangeAuthorizationCode(tx, r.PostForm, resource)
			oauthErr = inner
			if inner != nil && inner.Status >= http.StatusInternalServerError {
				return inner
			}
			return nil
		})
		if txErr != nil && oauthErr == nil {
			writeOAuthError(w, &mcpserver.OAuthError{Code: "server_error", Description: "failed to exchange code", Status: http.StatusInternalServerError})
			return
		}
	case "refresh_token":
		txErr := db.Transaction(func(tx *gorm.DB) error {
			var inner *mcpserver.OAuthError
			issue, refresh, inner = mcpserver.RefreshTokens(tx, r.PostForm, resource)
			oauthErr = inner
			if inner != nil && inner.Status >= http.StatusInternalServerError {
				return inner
			}
			return nil
		})
		if txErr != nil && oauthErr == nil {
			writeOAuthError(w, &mcpserver.OAuthError{Code: "server_error", Description: "failed to refresh token", Status: http.StatusInternalServerError})
			return
		}
	default:
		writeOAuthError(w, &mcpserver.OAuthError{Code: "unsupported_grant_type", Description: "grant_type must be authorization_code or refresh_token", Status: http.StatusBadRequest})
		return
	}
	if oauthErr != nil {
		writeOAuthError(w, oauthErr)
		return
	}

	access, err := mcpserver.MintAccessToken(s.jwt, mcpserver.AccessClaims{
		UserID:    issue.UserID,
		OrgID:     issue.OrgID,
		FactoryID: issue.FactoryID,
		ClientID:  issue.ClientID,
		Resource:  issue.Resource,
		Scopes:    issue.Scopes,
	}, mcpserver.AccessTokenTTL)
	if err != nil {
		writeOAuthError(w, &mcpserver.OAuthError{Code: "server_error", Description: "failed to mint access token", Status: http.StatusInternalServerError})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	respondJSON(w, map[string]any{
		"access_token":  access,
		"token_type":    "Bearer",
		"expires_in":    int(mcpserver.AccessTokenTTL / time.Second),
		"refresh_token": refresh,
		"scope":         strings.Join(issue.Scopes, " "),
	})
}

func (s *Server) handleMCPRegister(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeOAuthError(w, &mcpserver.OAuthError{Code: "invalid_client_metadata", Description: "request body is not valid", Status: http.StatusBadRequest})
		return
	}
	client, oauthErr := mcpserver.RegisterClient(database.DB(r.Context()), body)
	if oauthErr != nil {
		writeOAuthError(w, oauthErr)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"client_id":                  client.ID,
		"client_name":                client.Name,
		"redirect_uris":              client.RedirectURIs,
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"token_endpoint_auth_method": "none",
	})
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	origin := s.mcpOrigin(r)
	resource := mcpserver.ResourceURL(origin)
	token := bearerToken(r.Header.Get("Authorization"))
	claims, apiTokenID, err := s.mcpAccessClaims(r, token, resource)
	if err != nil {
		w.Header().Set("WWW-Authenticate", mcpserver.WWWAuthenticate(origin))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !mcpserver.OrganizationAllowsPublicMCP(claims.OrgID) {
		http.NotFound(w, r)
		return
	}
	if apiTokenID != uuid.Nil {
		_ = models.TouchMCPAPITokenLastUsed(database.DB(r.Context()), apiTokenID, time.Now())
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	message, err := mcpserver.ParseJSONRPC(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"error":   map[string]any{"code": -32700, "message": "Parse error"},
		})
		return
	}
	runtime := &mcpserver.Runtime{
		Auth: s.authService,
		Intake: factoryactions.IntakeDependencies{
			Registry:    s.registry,
			Encryptor:   s.encryptor,
			AuthService: s.authService,
		},
	}
	response := mcpserver.HandleJSONRPC(r.Context(), runtime, claims, message)
	if response == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) mcpAccessClaims(r *http.Request, token, resource string) (*mcpserver.AccessClaims, uuid.UUID, error) {
	db := database.DB(r.Context())
	if mcpserver.IsAPIToken(token) {
		return mcpserver.ClaimsForAPIToken(db, token, resource)
	}
	claims, err := mcpserver.ParseAccessToken(s.jwt, token, resource)
	if err != nil {
		return nil, uuid.Nil, err
	}
	if mcpserver.UserAccountBlocked(db, claims.UserID) {
		return nil, uuid.Nil, errUnauthorizedMCP
	}
	if !mcpserver.AccessGrantIsActive(db, claims, time.Now()) {
		return nil, uuid.Nil, errUnauthorizedMCP
	}
	return claims, uuid.Nil, nil
}

var errUnauthorizedMCP = errors.New("unauthorized")

func (s *Server) mcpHTTPClient() mcp.HTTPDoer {
	if s.registry != nil {
		return mcp.DoerFromCore(s.registry.HTTPContext())
	}
	return mcp.DoerFromCore(nil)
}

func writeOAuthError(w http.ResponseWriter, err *mcpserver.OAuthError) {
	status := err.Status
	if status == 0 {
		status = http.StatusBadRequest
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":             err.Code,
		"error_description": err.Description,
	})
}
