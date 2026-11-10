package authentication

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
)

func logAuthentication(r *http.Request, action, outcome string, account *models.Account) {
	ctx := r.Context()
	identity := logging.SecurityIdentityFrom(ctx)
	if identity == nil {
		identity = &logging.SecurityIdentity{ClientIP: logging.ClientIP(r)}
		ctx = logging.WithSecurityIdentity(ctx, identity)
	}
	// An attempted email address or an unvalidated cookie is not an actor.
	if account != nil {
		identity.AccountID = account.ID.String()
	}
	targetType, targetID := "session", ""
	if account != nil {
		targetType, targetID = "account", account.ID.String()
	}
	logging.SecurityEvent(ctx, action, outcome, targetType, targetID, nil)
}
