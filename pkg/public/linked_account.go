package public

import (
	"net/http"
	"slices"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

// linkableProviders lists the services a member can link. A linked account
// records who the member is on that service, so SuperPlane can attribute the
// activity it finds there. Sign-in methods are a separate concern.
var linkableProviders = []string{models.ProviderGitHub}

func accountLinkedAccountResponses(linked []models.AccountLinkedAccount) []AccountLinkedAccountResponse {
	responses := make([]AccountLinkedAccountResponse, 0, len(linked))
	for _, account := range linked {
		responses = append(responses, AccountLinkedAccountResponse{
			Provider:  account.Provider,
			Username:  account.Username,
			Name:      account.Name,
			AvatarURL: account.AvatarURL,
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
	if hasActiveImpersonation(r) {
		http.Error(w, "Linked accounts cannot change while impersonating", http.StatusForbidden)
		return
	}

	provider := mux.Vars(r)["provider"]
	if !slices.Contains(linkableProviders, provider) {
		http.Error(w, "Unknown linked account", http.StatusBadRequest)
		return
	}

	err := models.DeleteAccountLinkedAccount(database.DB(r.Context()), account.ID, provider)
	if err != nil {
		log.Errorf("Error removing linked %s account for %s: %v", provider, account.ID, err)
		http.Error(w, "Failed to remove linked account", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
