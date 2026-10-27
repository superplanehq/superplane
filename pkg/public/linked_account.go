package public

import (
	"errors"
	"net/http"
	"slices"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

func accountLinkedAccountResponses(linked []models.AccountLinkedAccount) []AccountLinkedAccountResponse {
	responses := make([]AccountLinkedAccountResponse, 0, len(linked))
	for _, account := range linked {
		responses = append(responses, AccountLinkedAccountResponse{
			Provider:   account.Provider,
			ProviderID: account.ProviderID,
			Username:   account.Username,
			Name:       account.Name,
			AvatarURL:  account.AvatarURL,
			Active:     account.Active,
		})
	}
	return responses
}

func (s *Server) disconnectLinkedAccount(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if hasActiveSupportSession(r) {
		http.Error(w, "Linked accounts cannot change in a read-only support session", http.StatusForbidden)
		return
	}

	provider := mux.Vars(r)["provider"]
	if !slices.Contains(models.LinkableProviders, provider) {
		http.Error(w, "Unknown linked account", http.StatusBadRequest)
		return
	}
	providerID := mux.Vars(r)["providerID"]
	if providerID == "" {
		http.Error(w, "Linked account id is required", http.StatusBadRequest)
		return
	}

	err := models.DeleteAccountLinkedAccount(database.DB(r.Context()), account.ID, provider, providerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "Linked account not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Errorf("Error removing linked %s account for %s: %v", provider, account.ID, err)
		http.Error(w, "Failed to remove linked account", http.StatusInternalServerError)
		return
	}

	if provider == models.ProviderGitHub {
		s.removeGitHubPRAuthorizationForIdentity(r, account.ID, providerID)
	}

	w.WriteHeader(http.StatusNoContent)
}

// removeGitHubPRAuthorizationForIdentity stops pull requests as a GitHub
// identity that the member no longer links.
func (s *Server) removeGitHubPRAuthorizationForIdentity(r *http.Request, accountID uuid.UUID, providerID string) {
	authorization, err := models.FindAccountGitHubAuthorization(database.DB(r.Context()), accountID)
	if err != nil || authorization.GitHubUserID != providerID {
		return
	}

	cfg, err := githubapp.Resolve(r.Context(), database.DB(r.Context()), s.encryptor)
	if err != nil {
		cfg = config.GitHubHostedAppConfig{}
	}
	revokeErr, err := githubapp.DeleteUserAuthorization(r.Context(), database.DB(r.Context()), s.encryptor, s.registry.HTTPContext(), cfg, accountID)
	if err != nil {
		log.WithError(err).Errorf("Error removing GitHub pull request authorization for %s", accountID)
		return
	}
	if revokeErr != nil {
		log.WithError(revokeErr).Warnf("Could not revoke GitHub user token for %s", accountID)
	}
}
