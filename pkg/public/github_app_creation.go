package public

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *Server) gitHubAppCreationCallbacks(r *http.Request, w http.ResponseWriter, integration *models.Integration) *core.IntegrationCreationCallbacks {
	db := database.DB(r.Context())
	var claim string
	var claimedCtx core.HTTPRequestContext
	return &core.IntegrationCreationCallbacks{
		Claim: func(ctx core.HTTPRequestContext, state string) (core.HTTPRequestContext, error) {
			fresh, identifier, err := integration.ClaimGitHubAppCreation(db, state)
			if err != nil {
				return ctx, err
			}
			claim = identifier
			ctx.Integration = contexts.NewIntegrationContext(db, nil, fresh, s.encryptor, s.registry, nil)
			ctx.IntegrationSetup = contexts.NewIntegrationSetupContext(fresh)
			claimedCtx = ctx
			return ctx, nil
		},
		Finalize: func(write func(core.HTTPRequestContext)) error {
			buffer := httptest.NewRecorder()
			err := db.Transaction(func(tx *gorm.DB) error {
				fresh, err := integration.LockClaimedGitHubAppCreation(tx, claim)
				if err != nil {
					return err
				}
				ctx := claimedCtx
				ctx.Response = buffer
				ctx.Integration = contexts.NewIntegrationContext(tx, nil, fresh, s.encryptor, s.registry, nil)
				ctx.IntegrationSetup = contexts.NewIntegrationSetupContext(fresh)
				write(ctx)
				if buffer.Code != http.StatusSeeOther {
					return fmt.Errorf("GitHub App creation failed with status %d", buffer.Code)
				}
				if !fresh.IsLegacy() {
					setup := fresh.SetupState.Data()
					setup.PreviousSteps = append(setup.PreviousSteps, *setup.CurrentStep)
					setup.CurrentStep = &core.SetupStep{
						Type:           core.SetupStepTypeRedirectPrompt,
						Name:           github.SetupStepInstallApp,
						Label:          "Install the GitHub app",
						RedirectPrompt: &core.RedirectPrompt{URL: buffer.Header().Get("Location"), Method: "GET"},
					}
					updated := datatypes.NewJSONType(setup)
					fresh.SetupState = &updated
				}
				fresh.ClearGitHubAppCreationClaim()
				return tx.Save(fresh).Error
			})
			if err != nil {
				return err
			}
			for key, values := range buffer.Header() {
				w.Header()[key] = values
			}
			w.WriteHeader(buffer.Code)
			_, err = w.Write(buffer.Body.Bytes())
			return err
		},
	}
}
