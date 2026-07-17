package authentication

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	jwtLib "github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/utils"
	"gorm.io/gorm"
)

const SignupDisabledError = "signup is currently disabled"

const (
	magicCodeLength            = 6
	magicCodeTTL               = 10 * time.Minute
	magicCodeRateLimit         = 5
	magicCodeRateWindow        = 10 * time.Minute
	magicCodeMaxVerifyAttempts = 3
	authSignupStatePrefix      = "signup:"
	authSignupResultParam      = "auth_signup_result"
	authErrorParam             = "auth_error"
	authErrorSignupRequired    = "signup_required"
	jsonContentType            = "application/json"
)

type Handler struct {
	jwtSigner            *jwt.Signer
	authService          authorization.Authorization
	encryptor            crypto.Encryptor
	isDev                bool
	templateDir          string
	blockSignup          bool
	passwordLoginEnabled bool
	magicCodeEnabled     bool
}

type ProviderConfig struct {
	Key         string
	Secret      string
	CallbackURL string
}

func NewHandler(jwtSigner *jwt.Signer, encryptor crypto.Encryptor, authService authorization.Authorization, appEnv string, templateDir string, blockSignup bool, passwordLoginEnabled bool, magicCodeEnabled bool) *Handler {
	return &Handler{
		jwtSigner:            jwtSigner,
		encryptor:            encryptor,
		authService:          authService,
		isDev:                appEnv == "development",
		templateDir:          templateDir,
		blockSignup:          blockSignup,
		passwordLoginEnabled: passwordLoginEnabled,
		magicCodeEnabled:     magicCodeEnabled,
	}
}

// PasswordLoginEnabled reports whether email/password authentication is
// currently enabled for this installation. Used by handlers outside this
// package (e.g. the change-password endpoint) that should refuse work
// when the feature is disabled.
func (a *Handler) PasswordLoginEnabled() bool {
	return a.passwordLoginEnabled
}

func (a *Handler) InitializeProviders(providers map[string]ProviderConfig) {
	var gothProviders []goth.Provider

	for providerName, config := range providers {
		if config.Key == "" || config.Secret == "" {
			log.Warnf("%s OAuth not configured - missing key/secret", providerName)
			continue
		}

		switch providerName {
		case models.ProviderGitHub:
			gothProviders = append(gothProviders, github.New(config.Key, config.Secret, config.CallbackURL, "user:email"))
			log.Infof("GitHub OAuth provider initialized")
		case models.ProviderGoogle:
			gothProviders = append(gothProviders, google.New(config.Key, config.Secret, config.CallbackURL, "email", "profile"))
			log.Infof("Google OAuth provider initialized")
		default:
			log.Warnf("Unknown provider: %s", providerName)
		}
	}

	if len(gothProviders) > 0 {
		goth.UseProviders(gothProviders...)
	} else {
		log.Warn("No OAuth providers configured")
	}
}

func (a *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/logout", a.handleLogout).Methods("GET")
	router.HandleFunc("/auth/config", a.handleAuthConfig).Methods("GET")
	if a.passwordLoginEnabled {
		router.HandleFunc("/login", a.handlePasswordLogin).Methods("POST")
		router.HandleFunc("/signup", a.handlePasswordSignup).Methods("POST")
	}
	if a.magicCodeEnabled {
		router.HandleFunc("/auth/magic-code/request", a.handleMagicCodeRequest).Methods("POST")
		router.HandleFunc("/auth/magic-code/verify", a.handleMagicCodeVerify).Methods("POST")
		router.HandleFunc("/auth/magic-code/verify", a.handleMagicLinkRedirect).Methods("GET")
	}

	//
	// If we are running the application locally,
	// we provide handlers that auto-autenticate to
	// avoid having to authenticate with GitHub every time.
	//
	if a.isDev {
		log.Info("Registering development authentication routes")
		router.HandleFunc("/auth/{provider}/callback", a.handleDevAuth).Methods("GET")
		router.HandleFunc("/auth/{provider}", a.handleDevAuth).Methods("GET")
		return
	}

	router.HandleFunc("/auth/{provider}/callback", a.handleAuthCallback).Methods("GET")
	router.HandleFunc("/auth/{provider}", a.handleAuth).Methods("GET")
}

func (a *Handler) handleAuth(w http.ResponseWriter, r *http.Request) {
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err == nil {
		a.handleSuccessfulAuth(w, r, gothUser, false)
		return
	}

	authState := getAuthState(r)
	if authState != "" {
		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		q := r2.URL.Query()
		q.Set("state", authState)
		r2.URL.RawQuery = q.Encode()
		r = r2
	}

	gothic.BeginAuthHandler(w, r)
}

func (a *Handler) handleDevAuth(w http.ResponseWriter, r *http.Request) {
	if !a.isDev {
		http.Error(w, "Not available in production", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	provider := vars["provider"]
	mockUser := goth.User{
		UserID:      "dev-user-123",
		Email:       "dev@superplane.local",
		Name:        "Dev User",
		NickName:    "devuser",
		Provider:    provider,
		AvatarURL:   "https://github.com/github.png",
		AccessToken: "dev-token-" + provider,
	}

	account, wasCreated, err := a.findOrCreateAccountForProvider(mockUser, a.allowSignupFromRequest(r))

	if err != nil {
		if errors.Is(err, errSignupRequired) {
			http.Redirect(w, r, getSignupRequiredRedirectURL(r), http.StatusSeeOther)
			return
		}

		if errorStatusForAccountError(err) == http.StatusForbidden {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		log.Errorf("Error finding/creating dev account for %s: %v", mockUser.Email, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = updateAccountProviders(a.encryptor, account, mockUser)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	a.handleSuccessfulAuth(w, r, mockUser, wasCreated)
}

func (a *Handler) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	account, wasCreated, err := a.findOrCreateAccountForProvider(gothUser, a.allowSignupFromRequest(r))
	if err != nil {
		if errors.Is(err, errSignupRequired) {
			http.Redirect(w, r, getSignupRequiredRedirectURL(r), http.StatusSeeOther)
			return
		}

		if errorStatusForAccountError(err) == http.StatusForbidden {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		log.Errorf("Error finding/creating account for %s: %v", gothUser.Email, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = updateAccountProviders(a.encryptor, account, gothUser)
	if err != nil {
		log.Errorf("Error updating account providers for %s: %v", gothUser.Email, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	a.handleSuccessfulAuth(w, r, gothUser, wasCreated)
}

func (a *Handler) handleSuccessfulAuth(w http.ResponseWriter, r *http.Request, gothUser goth.User, wasCreated bool) {
	account, err := models.FindAccountByEmail(gothUser.Email)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	if err := IssueAccountSession(w, r, a.jwtSigner, account.ID.String(), gothUser.Provider); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	redirectURL := a.getPostAuthRedirectURL(r, wasCreated)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (a *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	gothic.Logout(w, r)

	ClearAccountCookie(w, r)

	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
}

func (a *Handler) handleAuthConfig(w http.ResponseWriter, r *http.Request) {
	providers := goth.GetProviders()
	providerNames := make([]string, 0, len(providers))
	for name := range providers {
		providerNames = append(providerNames, name)
	}
	sort.Strings(providerNames)

	response := struct {
		Providers                   []string `json:"providers"`
		PasswordLoginEnabled        bool     `json:"passwordLoginEnabled"`
		SignupEnabled               bool     `json:"signupEnabled"`
		SignupsBlockedByEnvironment bool     `json:"signupsBlockedByEnvironment"`
		MagicCodeEnabled            bool     `json:"magicCodeEnabled"`
	}{
		Providers:                   providerNames,
		PasswordLoginEnabled:        a.passwordLoginEnabled,
		SignupEnabled:               a.SignupsEnabled(),
		SignupsBlockedByEnvironment: a.SignupsBlockedByEnvironment(),
		MagicCodeEnabled:            a.magicCodeEnabled,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("Error encoding auth config: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (a *Handler) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	if !a.passwordLoginEnabled {
		http.Error(w, "Password login is not enabled", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	// Find account by email
	account, err := models.FindAccountByEmail(email)
	if err != nil {
		log.Warnf("Login attempt with invalid email: %s", email)
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Find password auth for this account
	passwordAuth, err := models.FindAccountPasswordAuthByAccountID(account.ID)
	if err != nil {
		log.Warnf("Login attempt for account without password auth: %s", email)
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Verify password
	if !crypto.VerifyPassword(passwordAuth.PasswordHash, password) {
		log.Warnf("Invalid password attempt for account: %s", email)
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	if err := IssueAccountSession(w, r, a.jwtSigner, account.ID.String(), models.ProviderPassword); err != nil {
		log.Errorf("Failed to generate token for password login: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Redirect
	redirectURL := getRedirectURL(r)
	// Use StatusSeeOther (303) instead of StatusTemporaryRedirect (307) for POST requests
	// This ensures the browser uses GET for the redirect, not POST
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (a *Handler) handlePasswordSignup(w http.ResponseWriter, r *http.Request) {
	if !a.passwordLoginEnabled {
		http.Error(w, "Password login is not enabled", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	inviteToken := r.FormValue("invite_token")

	if name == "" || email == "" || password == "" {
		http.Error(w, "Name, email, and password are required", http.StatusBadRequest)
		return
	}

	if !a.SignupsEnabled() && inviteToken == "" {
		http.Error(w, SignupDisabledError, http.StatusForbidden)
		return
	}

	if inviteToken != "" {
		inviteLink, err := models.FindInviteLinkByToken(database.DB(r.Context()), inviteToken)
		if err != nil || !inviteLink.Enabled {
			http.Error(w, "invite link not found or disabled", http.StatusForbidden)
			return
		}
	}

	if _, err := models.FindAccountByEmail(email); err == nil {
		http.Error(w, "Account already exists", http.StatusConflict)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	tx := database.Conn().Begin()
	if tx.Error != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	account, err := models.CreateAccountInTransaction(tx, name, email)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to create account", http.StatusInternalServerError)
		return
	}

	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to create account", http.StatusInternalServerError)
		return
	}

	_, err = models.CreateAccountPasswordAuthInTransaction(tx, account.ID, passwordHash)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to create account", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit().Error; err != nil {
		http.Error(w, "Failed to create account", http.StatusInternalServerError)
		return
	}

	if err := IssueAccountSession(w, r, a.jwtSigner, account.ID.String(), models.ProviderPassword); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	redirectURL := a.getPostAuthRedirectURL(r, true)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (a *Handler) handleMagicCodeRequest(w http.ResponseWriter, r *http.Request) {
	if !a.magicCodeEnabled {
		http.Error(w, "Magic code login is not enabled", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	if email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	email = utils.NormalizeEmail(email)

	successResponse := func() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "If an account exists with this email, a sign-in code has been sent.",
		})
	}

	count, err := models.CountRecentMagicCodes(email, time.Now().Add(-magicCodeRateWindow))
	if err != nil {
		log.Errorf("Failed to count recent magic codes for %s: %v", email, err)
		successResponse()
		return
	}

	if count >= magicCodeRateLimit {
		log.Warnf("Magic code rate limit reached for %s", email)
		successResponse()
		return
	}

	code, err := generateMagicCode()
	if err != nil {
		log.Errorf("Failed to generate magic code: %v", err)
		successResponse()
		return
	}

	codeHash := crypto.HashToken(code)
	expiresAt := time.Now().Add(magicCodeTTL)

	_, err = models.CreateAccountMagicCode(email, codeHash, expiresAt)
	if err != nil {
		log.Errorf("Failed to store magic code for %s: %v", email, err)
		successResponse()
		return
	}

	magicLinkToken, err := a.generateMagicLinkToken(email, code)
	if err != nil {
		log.Errorf("Failed to generate magic link token for %s: %v", email, err)
		successResponse()
		return
	}

	redirectURL := strings.TrimSpace(r.FormValue("redirect"))
	msg := messages.NewMagicCodeRequestedMessage(email, code, magicLinkToken, redirectURL, isSignupIntentFromRequest(r))
	if err := msg.Publish(); err != nil {
		log.Errorf("Failed to publish magic code email request for %s: %v", email, err)
	}

	successResponse()
}

func (a *Handler) handleMagicCodeVerify(w http.ResponseWriter, r *http.Request) {
	if !a.magicCodeEnabled {
		http.Error(w, "Magic code login is not enabled", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email, code, err := a.parseMagicCodeInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 1. Validate the code exists without consuming it. This ensures that
	//    invalid-code requests always return 401 regardless of account
	//    existence, preventing email enumeration via differing status codes.
	magicCode, err := a.findValidCode(email, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Invalid or expired code", http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Check signup policy without creating any records. Only reachable
	//    with a valid code, so a 403 does not leak account existence.
	if err := a.checkSignupPolicy(email, r); err != nil {
		http.Error(w, err.Error(), errorStatusForAccountError(err))
		return
	}

	// 3. Atomically consume the code.
	if err := a.consumeCode(magicCode, email); err != nil {
		http.Error(w, err.Error(), errorStatusForCodeError(err))
		return
	}

	// 4. Find or create the account (policy already verified above).
	account, wasCreated, err := a.findOrCreateAccountForMagicCode(email, r)
	if err != nil {
		http.Error(w, err.Error(), errorStatusForAccountError(err))
		return
	}

	a.issueSessionAndRedirect(w, r, account, wasCreated)
}

func (a *Handler) parseMagicCodeInput(r *http.Request) (string, string, error) {
	var email, code string

	if magicLinkToken := strings.TrimSpace(r.FormValue("token")); magicLinkToken != "" {
		var parseErr error
		email, code, parseErr = a.parseMagicLinkToken(magicLinkToken)
		if parseErr != nil {
			log.Warnf("Invalid magic link token: %v", parseErr)
			return "", "", fmt.Errorf("Invalid or expired link")
		}
	} else {
		email = strings.TrimSpace(r.FormValue("email"))
		code = strings.TrimSpace(r.FormValue("code"))
	}

	email = utils.NormalizeEmail(email)
	code = stripNonDigits(code)

	if email == "" || code == "" {
		return "", "", fmt.Errorf("Email and code are required")
	}

	return email, code, nil
}

var errCodeUsed = fmt.Errorf("Invalid or expired code")
var errDBError = fmt.Errorf("Internal server error")

// findValidCode looks up a valid, unconsumed magic code without marking it
// as used. On failure it increments the per-code attempt counter.
func (a *Handler) findValidCode(email, code string) (*models.AccountMagicCode, error) {
	codeHash := crypto.HashToken(code)

	magicCode, err := models.FindValidAccountMagicCode(email, codeHash, magicCodeMaxVerifyAttempts)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("Database error during magic code lookup for %s: %v", email, err)
			return nil, errDBError
		}

		log.Warnf("Invalid magic code attempt for %s", email)
		_, incrErr := models.IncrementAndMaybeInvalidateCodes(email, magicCodeMaxVerifyAttempts)
		if incrErr != nil {
			log.Errorf("Failed to process verify attempt for %s: %v", email, incrErr)
		}

		return nil, gorm.ErrRecordNotFound
	}

	return magicCode, nil
}

// consumeCode atomically marks a previously-found code as used.
func (a *Handler) consumeCode(magicCode *models.AccountMagicCode, email string) error {
	marked, err := magicCode.MarkUsed()
	if err != nil {
		log.Errorf("Failed to mark magic code as used for %s: %v", email, err)
		return errDBError
	}
	if !marked {
		log.Warnf("Magic code already used (concurrent request) for %s", email)
		return errCodeUsed
	}

	return nil
}

func errorStatusForCodeError(err error) int {
	if err == errDBError {
		return http.StatusInternalServerError
	}
	return http.StatusUnauthorized
}

var errSignupDisabled = fmt.Errorf(SignupDisabledError)
var errSignupRequired = fmt.Errorf("signup must be started from the signup page")
var errInviteLinkInvalid = fmt.Errorf("invite link not found or disabled")
var errAccountError = fmt.Errorf("Internal server error")

// checkSignupPolicy verifies that a new-user signup would be allowed for
// the given email without creating any records. For existing accounts this
// is always a no-op.
func (a *Handler) checkSignupPolicy(email string, r *http.Request) error {
	_, err := models.FindAccountByEmail(email)
	if err == nil {
		return nil // existing user — no signup gate
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Errorf("Error finding account for %s: %v", email, err)
		return errAccountError
	}

	if allowSignupFromInvite(r) {
		return nil
	}

	if !isSignupIntentFromRequest(r) {
		return errSignupRequired
	}

	if !a.SignupsEnabled() {
		return errSignupDisabled
	}

	return nil
}

// findOrCreateAccountForMagicCode returns the existing account or creates a
// new one. Signup policy must already be verified by checkSignupPolicy.
func (a *Handler) findOrCreateAccountForMagicCode(email string, r *http.Request) (*models.Account, bool, error) {
	account, err := models.FindAccountByEmail(email)
	if err == nil {
		return account, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Errorf("Error finding account for %s: %v", email, err)
		return nil, false, errAccountError
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = strings.Split(email, "@")[0]
	}

	account, err = models.CreateAccount(name, email)
	if err != nil {
		account, err = models.FindAccountByEmail(email)
		if err != nil {
			log.Errorf("Failed to create or find account for %s: %v", email, err)
			return nil, false, errAccountError
		}
		return account, false, nil
	}

	return account, true, nil
}

func errorStatusForAccountError(err error) int {
	switch err {
	case errSignupDisabled, errSignupRequired, errInviteLinkInvalid:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func (a *Handler) issueSessionAndRedirect(w http.ResponseWriter, r *http.Request, account *models.Account, wasCreated bool) {
	if err := IssueAccountSession(w, r, a.jwtSigner, account.ID.String(), models.ProviderMagicCode); err != nil {
		log.Errorf("Failed to generate token for magic code login: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	redirectURL := a.getPostAuthRedirectURL(r, wasCreated)
	writePostAuthRedirect(w, r, redirectURL)
}

func writePostAuthRedirect(w http.ResponseWriter, r *http.Request, redirectURL string) {
	if strings.Contains(r.Header.Get("Accept"), jsonContentType) {
		w.Header().Set("Content-Type", jsonContentType)
		if err := json.NewEncoder(w).Encode(map[string]string{"redirectUrl": redirectURL}); err != nil {
			log.Errorf("Error encoding auth redirect response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (a *Handler) getPostAuthRedirectURL(r *http.Request, wasCreated bool) string {
	redirectURL := getRedirectURL(r)
	if !wasCreated {
		if isSignupIntentFromRequest(r) {
			return addAuthSignupResult(redirectURL, "existing")
		}

		return redirectURL
	}

	if !a.magicCodeEnabled {
		return addAuthSignupResult(redirectURL, "created")
	}

	if redirectURL == "/" {
		return "/welcome"
	}

	return fmt.Sprintf("/welcome?redirect=%s", url.QueryEscape(redirectURL))
}

func addAuthSignupResult(redirectURL string, result string) string {
	parsedURL, err := url.Parse(redirectURL)
	if err != nil {
		return redirectURL
	}

	params := parsedURL.Query()
	params.Set(authSignupResultParam, result)
	parsedURL.RawQuery = params.Encode()
	return parsedURL.String()
}

func getSignupRequiredRedirectURL(r *http.Request) string {
	params := url.Values{}
	params.Set(authErrorParam, authErrorSignupRequired)

	redirectURL := getRedirectURL(r)
	if redirectURL != "/" {
		params.Set("redirect", redirectURL)
	}

	return fmt.Sprintf("/signup?%s", params.Encode())
}

func generateMagicCode() (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(magicCodeLength), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%0*d", magicCodeLength, n), nil
}

func (a *Handler) handleMagicLinkRedirect(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	authPath := "/login"
	if isSignupIntentFromRequest(r) {
		authPath = "/signup"
	}

	redirectURL := fmt.Sprintf("%s?magic_link_token=%s", authPath, url.QueryEscape(token))

	// Preserve the redirect parameter so invite context survives the
	// magic-link round-trip through email.
	if redirect := r.URL.Query().Get("redirect"); redirect != "" {
		redirectURL += "&redirect=" + url.QueryEscape(redirect)
	}
	if isSignupIntentFromRequest(r) {
		redirectURL += "&signup=true"
	}

	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func stripNonDigits(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func (a *Handler) generateMagicLinkToken(email, code string) (string, error) {
	now := time.Now()
	token := jwtLib.NewWithClaims(jwtLib.SigningMethodHS256, jwtLib.MapClaims{
		"email": email,
		"code":  code,
		"iat":   now.Unix(),
		"exp":   now.Add(magicCodeTTL).Unix(),
		"type":  "magic_link",
	})

	return token.SignedString([]byte(a.jwtSigner.Secret))
}

func (a *Handler) parseMagicLinkToken(tokenString string) (email string, code string, err error) {
	claims, err := a.jwtSigner.ValidateAndGetClaims(tokenString)
	if err != nil {
		return "", "", err
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != "magic_link" {
		return "", "", fmt.Errorf("invalid token type")
	}

	email, _ = claims["email"].(string)
	code, _ = claims["code"].(string)
	if email == "" || code == "" {
		return "", "", fmt.Errorf("missing claims")
	}

	return email, code, nil
}

func (a *Handler) FindOrCreateAccountForProvider(gothUser goth.User) (*models.Account, error) {
	account, _, err := a.findOrCreateAccountForProvider(gothUser, a.SignupsEnabled())
	return account, err
}

func (a *Handler) findOrCreateAccountForProvider(gothUser goth.User, allowSignup bool) (*models.Account, bool, error) {
	account, err := models.FindAccountByProvider(gothUser.Provider, gothUser.UserID)

	if err == nil {
		if account.Email != utils.NormalizeEmail(gothUser.Email) {
			log.Infof("Updating email for account %s from %s to %s", account.ID, account.Email, gothUser.Email)
			err = account.UpdateEmailForProvider(gothUser.Email, gothUser.Provider, gothUser.UserID)

			if err != nil {
				log.Errorf("Failed to update account email: %v", err)
				return nil, false, fmt.Errorf("failed to update account email: %w", err)
			}
		}
		return account, false, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	account, err = models.FindAccountByEmail(gothUser.Email)
	if err == nil {
		return account, false, nil
	}

	if !allowSignup {
		log.Warnf("Signup blocked for email: %s", gothUser.Email)
		return nil, false, errSignupRequired
	}

	account, err = models.CreateAccount(gothUser.Name, gothUser.Email)
	if err != nil {
		return nil, false, err
	}

	return account, true, nil
}

func (a *Handler) allowSignupFromRequest(r *http.Request) bool {
	if allowSignupFromInvite(r) {
		return true
	}

	return a.SignupsEnabled() && isSignupIntentFromRequest(r)
}

func (a *Handler) SignupsEnabled() bool {
	if a.blockSignup {
		return false
	}

	metadata, err := models.GetInstallationMetadata(database.Conn())
	return signupsEnabledFromMetadata(metadata, err)
}

func signupsEnabledFromMetadata(metadata *models.InstallationMetadata, err error) bool {
	if err != nil {
		log.Errorf("Error loading installation metadata for signup policy: %v", err)
		return true
	}

	return metadata.SignupsEnabled
}

func (a *Handler) SignupsBlockedByEnvironment() bool {
	return a.blockSignup
}

func allowSignupFromInvite(r *http.Request) bool {
	redirectURL := getRedirectURL(r)
	if !strings.HasPrefix(redirectURL, "/invite/") {
		return false
	}

	parsedURL, err := url.Parse(redirectURL)
	if err != nil {
		return false
	}

	inviteToken := strings.TrimPrefix(parsedURL.Path, "/invite/")
	if inviteToken == "" || strings.Contains(inviteToken, "/") {
		return false
	}

	inviteLink, err := models.FindInviteLinkByToken(database.DB(r.Context()), inviteToken)
	if err != nil || !inviteLink.Enabled {
		return false
	}

	return true
}

func isSignupIntentFromRequest(r *http.Request) bool {
	signupValue := strings.ToLower(strings.TrimSpace(r.FormValue("signup")))
	if signupValue == "true" || signupValue == "1" || signupValue == "yes" {
		return true
	}

	return strings.HasPrefix(r.URL.Query().Get("state"), authSignupStatePrefix)
}

func getAuthState(r *http.Request) string {
	redirectURL := getRedirectURL(r)
	if isSignupIntentFromRequest(r) {
		return authSignupStatePrefix + url.QueryEscape(redirectURL)
	}

	if redirectURL == "/" {
		return ""
	}

	return redirectURL
}

func updateAccountProviders(encryptor crypto.Encryptor, account *models.Account, gothUser goth.User) error {
	accessToken, err := encryptor.Encrypt(context.Background(), []byte(gothUser.AccessToken), []byte(gothUser.Email))
	if err != nil {
		return err
	}

	accountProvider, err := account.FindAccountProviderByID(gothUser.Provider, gothUser.UserID)

	//
	// If we already have an account provider for this provider and provider ID, we just update it.
	//
	if err == nil {
		accountProvider.AccessToken = base64.StdEncoding.EncodeToString(accessToken)
		accountProvider.Username = gothUser.NickName
		accountProvider.Email = utils.NormalizeEmail(gothUser.Email)
		accountProvider.Name = gothUser.Name
		accountProvider.AvatarURL = gothUser.AvatarURL
		accountProvider.RefreshToken = gothUser.RefreshToken
		if !gothUser.ExpiresAt.IsZero() {
			accountProvider.TokenExpiresAt = &gothUser.ExpiresAt
		}

		return database.Conn().Save(accountProvider).Error
	}

	//
	// Otherwise, we create a new account provider.
	//
	accountProvider = &models.AccountProvider{
		AccountID:      account.ID,
		Provider:       gothUser.Provider,
		ProviderID:     gothUser.UserID,
		Username:       gothUser.NickName,
		Email:          utils.NormalizeEmail(gothUser.Email),
		Name:           gothUser.Name,
		AvatarURL:      gothUser.AvatarURL,
		AccessToken:    base64.StdEncoding.EncodeToString(accessToken),
		RefreshToken:   gothUser.RefreshToken,
		TokenExpiresAt: &gothUser.ExpiresAt,
	}

	return database.Conn().Create(accountProvider).Error
}

func getRedirectURL(r *http.Request) string {
	redirectParam := r.URL.Query().Get("redirect")

	if redirectParam == "" {
		redirectParam = r.URL.Query().Get("state")
	}

	if redirectParam == "" {
		return "/"
	}

	if strings.HasPrefix(redirectParam, authSignupStatePrefix) {
		redirectParam = strings.TrimPrefix(redirectParam, authSignupStatePrefix)
		if redirectParam == "" {
			return "/"
		}
	}

	decodedURL, err := url.QueryUnescape(redirectParam)
	if err == nil && isValidRedirectURL(decodedURL) {
		return decodedURL
	}

	return "/"
}

// Validates that the redirect URL is a valid internal URL.
// It rejects external URLs and URLs with multiple slashes.
func isValidRedirectURL(redirectURL string) bool {
	if redirectURL == "" || redirectURL[0] != '/' {
		return false
	}

	if len(redirectURL) > 1 && redirectURL[1] == '/' {
		return false
	}

	return true
}

func ClearAccountCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "account_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

// SetAccountCookie writes the account_token cookie with the same flags
// used by every login path. Centralized so callers (login, signup, magic
// code, change-password reissue) cannot drift on cookie attributes.
func SetAccountCookie(w http.ResponseWriter, r *http.Request, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     "account_token",
		Value:    token,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}
