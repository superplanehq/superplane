package changesets

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

func (p *CanvasPublisher) validateHumanReviewPolicy(ctx context.Context) error {
	wasHumanReview := models.FactoryAppTemplateID(p.live.Nodes) == models.FactoryAppTemplateHumanReviewID
	isHumanReview := models.FactoryAppTemplateID(p.draft.Nodes) == models.FactoryAppTemplateHumanReviewID
	if !wasHumanReview && !isHumanReview {
		return nil
	}
	if !p.options.BuiltInAutomationReset {
		if err := p.requireHumanReviewAdministrator(ctx); err != nil {
			return err
		}
	}
	if p.canvas.FactoryID != nil {
		factory, err := models.FindFactory(p.tx, p.canvas.OrganizationID, *p.canvas.FactoryID)
		if err != nil {
			return err
		}
		if err := factory.LockHumanReviewPolicy(p.tx); err != nil {
			return err
		}
		if isHumanReview {
			other, err := factory.HasOtherHumanReviewAutomation(p.tx, p.canvas.ID)
			if err != nil {
				return err
			}
			if other {
				return grpcerrors.AlreadyExists(nil, "This Factory already has a Human review automation")
			}
		}
		if err := factory.LockHumanReviewPublications(p.tx); err != nil {
			return err
		}
	}
	settings, present := models.HumanReviewSettingsFromNodes(p.draft.Nodes)
	if !present {
		return grpcerrors.FailedPrecondition(nil, "Keep the managed Human review trigger. Disable Human review in the automation settings")
	}
	if err := settings.Validate(); err != nil {
		return grpcerrors.InvalidArgument(err, err.Error())
	}
	if p.options.BuiltInAutomationReset {
		return nil
	}
	saved, _ := models.HumanReviewSettingsFromNodes(p.live.Nodes)
	if !settings.Enabled || (saved.Enabled && saved.DefaultReviewer == settings.DefaultReviewer && saved.FallbackTeam == settings.FallbackTeam) {
		return nil
	}
	return p.validateHumanReviewFallbackReviewer(ctx, settings)
}

func (p *CanvasPublisher) requireHumanReviewAdministrator(ctx context.Context) error {
	return RequireHumanReviewAdministrator(ctx, p.options.AuthService, p.canvas.OrganizationID)
}

func RequireHumanReviewAdministrator(ctx context.Context, authService authorization.Authorization, organizationID uuid.UUID) error {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return grpcerrors.PermissionDenied(nil, "Only administrators can change Human review settings")
	}
	roles, err := authService.GetUserRolesForOrg(ctx, userID, organizationID.String())
	if err != nil {
		return err
	}
	for _, role := range roles {
		if role.Name == models.RoleOrgAdmin {
			return nil
		}
	}
	return grpcerrors.PermissionDenied(nil, "Only administrators can change Human review settings")
}

func (p *CanvasPublisher) validateHumanReviewFallbackReviewer(ctx context.Context, settings models.HumanReviewSettings) error {
	if p.canvas.FactoryID == nil {
		return grpcerrors.InvalidArgument(nil, "Human review must belong to a Factory")
	}
	factory, err := models.FindFactory(p.tx, p.canvas.OrganizationID, *p.canvas.FactoryID)
	if err != nil {
		return err
	}
	config := factory.OnboardingConfigValue()
	if config.EffectiveVCSProvider() != models.ProviderGitHub {
		return grpcerrors.InvalidArgument(nil, "Human review requires GitHub")
	}
	integrationID, err := uuid.Parse(config.VCSIntegrationID)
	if err != nil {
		return grpcerrors.FailedPrecondition(err, "Connect GitHub before enabling Human review")
	}
	integration, err := models.FindIntegrationInTransaction(p.tx, p.canvas.OrganizationID, integrationID)
	if err != nil {
		return err
	}
	if integration.AppName != models.ProviderGitHub || integration.State != models.IntegrationStateReady {
		return grpcerrors.FailedPrecondition(nil, "Connect GitHub before enabling Human review")
	}
	client, err := common.NewClient(
		contexts.NewIntegrationContext(p.tx, nil, integration, p.options.Encryptor, p.options.Registry, nil),
		p.options.Registry.HTTPContextInTransaction(p.tx),
	)
	if err != nil {
		return grpcerrors.FailedPrecondition(err, "Could not connect to GitHub to validate the fallback reviewer")
	}
	if settings.DefaultReviewer != "" {
		return validateHumanReviewDefaultReviewer(ctx, client, config.AppRepository, settings.DefaultReviewer)
	}
	team := settings.FallbackTeam
	organization, slug, _ := strings.Cut(team, "/")
	found, _, err := client.GetTeamBySlug(ctx, organization, slug)
	if err != nil || found == nil || found.GetID() == 0 {
		return grpcerrors.FailedPrecondition(err, fmt.Sprintf("Cannot access fallback team %s. Check the team and GitHub App permissions", team))
	}
	return nil
}

func validateHumanReviewDefaultReviewer(ctx context.Context, client *common.Client, repository, reviewer string) error {
	if strings.TrimSpace(repository) == "" {
		return grpcerrors.FailedPrecondition(nil, "Select a repository before enabling Human review")
	}
	permission, _, err := client.GetRepositoryPermissionLevel(ctx, repository, reviewer)
	if err != nil {
		return grpcerrors.FailedPrecondition(err, fmt.Sprintf("Cannot access default reviewer %s. Check repository access and GitHub App permissions", reviewer))
	}
	user := permission.GetUser()
	if user.GetID() == 0 || user.GetType() != "User" || strings.HasSuffix(strings.ToLower(user.GetLogin()), "[bot]") || !slices.Contains([]string{"admin", "maintain", "write"}, permission.GetPermission()) {
		return grpcerrors.FailedPrecondition(nil, fmt.Sprintf("Default reviewer %s must be a human with repository write access", reviewer))
	}
	return nil
}
