package factories

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/bitbucketapp"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func linkedBitbucketAccountID(ctx context.Context, db *gorm.DB, organizationID string) (string, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return "", grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	user, err := models.FindActiveUserByIDInTransaction(db, organizationID, userID)
	if err != nil {
		return "", factoryErrorToStatus(err, "failed to load user")
	}
	if user.AccountID == nil {
		return "", grpcerrors.FailedPrecondition(nil, "connect a Bitbucket account first")
	}
	linked, err := models.FindAccountLinkedAccount(db, *user.AccountID, models.ProviderBitbucket)
	if err != nil {
		return "", grpcerrors.FailedPrecondition(err, "connect a Bitbucket account first")
	}
	accountID, err := bitbucket.NormalizeAccountID(linked.ProviderID)
	if err != nil {
		return "", grpcerrors.FailedPrecondition(err, "linked Bitbucket account has an invalid user id")
	}
	return accountID, nil
}

func visibleBitbucketRepositories(ctx context.Context, db *gorm.DB, accountUUID string) ([]bitbucket.VisibleRepository, error) {
	installations, err := models.ListActiveBitbucketForgeInstallations(db)
	if err != nil {
		return nil, err
	}
	refs := make([]bitbucket.InstallationRef, 0, len(installations))
	for _, installation := range installations {
		refs = append(refs, bitbucket.InstallationRef{
			ID:            installation.InstallationID,
			WorkspaceUUID: installation.WorkspaceUUID,
			WorkspaceSlug: installation.WorkspaceSlug,
		})
	}
	return bitbucket.CurrentDirectory().RepositoriesVisibleTo(ctx, accountUUID, refs, func(installationID string) (string, error) {
		token, _, tokenErr := bitbucketapp.CurrentSystemToken(installationID)
		return token, tokenErr
	})
}

func findVisibleBitbucketRepository(ctx context.Context, db *gorm.DB, accountUUID, repositoryName string) (*bitbucket.VisibleRepository, error) {
	repositories, err := visibleBitbucketRepositories(ctx, db, accountUUID)
	if err != nil {
		return nil, err
	}
	for index := range repositories {
		if strings.EqualFold(repositories[index].FullName, repositoryName) {
			return &repositories[index], nil
		}
	}
	return nil, nil
}

func forgeInstallationID(integration *models.Integration) string {
	metadata := integration.Metadata.Data()
	if metadata == nil {
		return ""
	}
	value, _ := metadata["forgeInstallationId"].(string)
	return strings.TrimSpace(value)
}
