package pulls

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	OpenedAsUser = "user"
	OpenedAsBot  = "bot"
)

// userPullRequest is a pull request that a member's GitHub account opened.
type userPullRequest struct {
	pullRequest *github.PullRequest
	login       string
}

// createPullRequestAsUsers tries each user in order and returns the first pull
// request that a user's GitHub account opens. It returns nil when no user can
// open it, so the caller opens it as the app.
func createPullRequestAsUsers(
	ctx core.ExecutionContext,
	appClient *common.Client,
	repository string,
	request *github.NewPullRequest,
	userIDs []string,
) *userPullRequest {
	if len(userIDs) == 0 || !common.UsesHostedApp(ctx.Integration) {
		return nil
	}

	appConfig, err := githubapp.ResolveProcess(context.Background())
	if err != nil || !appConfig.UserAuthorizationEnabled() {
		return nil
	}
	granted, err := appClient.HostedRepositoryGranted(ctx.Integration, repository)
	if err != nil {
		logOpenAsUser(ctx, "", err)
		return nil
	}
	if !granted {
		logOpenAsUser(ctx, "", fmt.Errorf("repository %q is not granted to this workspace", repository))
		return nil
	}
	encryptor, err := crypto.ProcessEncryptor()
	if err != nil {
		logOpenAsUser(ctx, "", err)
		return nil
	}

	for _, user := range assigneeUsers(ctx, userIDs) {
		pr, login, err := createPullRequestAsUser(ctx, appClient, appConfig, encryptor, user, repository, request)
		if err != nil {
			logOpenAsUser(ctx, user.ID.String(), err)
			continue
		}
		return &userPullRequest{pullRequest: pr, login: login}
	}
	return nil
}

func createPullRequestAsUser(
	ctx core.ExecutionContext,
	appClient *common.Client,
	appConfig config.GitHubHostedAppConfig,
	encryptor crypto.Encryptor,
	user models.User,
	repository string,
	request *github.NewPullRequest,
) (*github.PullRequest, string, error) {
	background := context.Background()
	token, err := githubapp.ResolveUserAccessToken(background, database.DB(background), encryptor, ctx.HTTP, appConfig, *user.AccountID)
	if err != nil {
		return nil, "", err
	}

	pr, _, err := appClient.WithUserToken(ctx.HTTP, token.Token).CreatePullRequest(background, repository, request)
	if common.StatusCode(err) == http.StatusUnauthorized {
		if invalidateErr := githubapp.InvalidateUserAuthorization(database.DB(background), token); invalidateErr != nil {
			logOpenAsUser(ctx, user.ID.String(), invalidateErr)
		}
	}
	if err != nil {
		return nil, "", explainGitHubError(err)
	}
	return pr, token.Login, nil
}

// assigneeUsers keeps the given order and drops users that are not active
// human members of this organization, or whose account is blocked.
func assigneeUsers(ctx core.ExecutionContext, userIDs []string) []models.User {
	users, err := models.FindUsersByIDsInOrganization(database.DB(context.Background()), ctx.OrganizationID, userIDs)
	if err != nil {
		logOpenAsUser(ctx, "", err)
		return nil
	}

	byID := make(map[string]models.User, len(users))
	for _, user := range users {
		if user.DeletedAt.Valid || !user.IsHuman() || user.AccountID == nil {
			continue
		}
		byID[user.ID.String()] = user
	}

	ordered := make([]models.User, 0, len(byID))
	for _, id := range userIDs {
		user, ok := byID[id]
		if !ok {
			continue
		}
		account, err := models.FindAccountByID(user.AccountID.String())
		if err != nil || account.IsBlocked() {
			continue
		}
		ordered = append(ordered, user)
	}
	return ordered
}

// parseOpenAsUsers reads a comma-separated list of user IDs. It keeps the first
// occurrence of each ID and drops values that are not IDs.
func parseOpenAsUsers(value string) []string {
	seen := map[string]bool{}
	ids := []string{}
	for _, part := range strings.Split(value, ",") {
		id, err := uuid.Parse(strings.TrimSpace(part))
		if err != nil || id == uuid.Nil || seen[id.String()] {
			continue
		}
		seen[id.String()] = true
		ids = append(ids, id.String())
	}
	return ids
}

func logOpenAsUser(ctx core.ExecutionContext, userID string, err error) {
	if ctx.Logger == nil {
		return
	}
	message := "cannot open the pull request with a member GitHub account"
	if userID != "" {
		message = fmt.Sprintf("cannot open the pull request with the GitHub account of user %s", userID)
	}
	ctx.Logger.WithError(err).Info(message)
}
