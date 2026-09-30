package public

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

func accountContext(account *models.Account) context.Context {
	return context.WithValue(context.Background(), middleware.AccountContextKey, account)
}
