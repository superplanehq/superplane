package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/taskowner"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const TaskOwnerHistoryUnavailable = "SuperPlane could not read the code history. Use the members list."

// taskOwnerHistoryFor is the code-history source. Tests replace it.
var taskOwnerHistoryFor = TaskOwnerHistory

func SuggestTaskOwners(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.SuggestTaskOwnersRequest,
) (*pb.SuggestTaskOwnersResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to suggest task owners")
	}

	db := database.DB(ctx)
	factoryModel, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to suggest task owners")
	}
	order, err := findWorkOrder(db, factoryModel, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to suggest task owners")
	}

	candidates, err := models.ListTaskOwnerCandidates(db, orgID, factoryModel.ID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to suggest task owners")
	}
	AddTaskOwnerRoles(ctx, deps.AuthService, orgID, candidates)

	repository := ""
	if order.Repository != nil {
		repository = *order.Repository
	}
	paths := taskowner.NormalizePaths(req.GetPaths())
	suggestion := SuggestCodeOwners(ctx, deps, factoryModel, order, repository, paths, candidates)

	return &pb.SuggestTaskOwnersResponse{
		Paths:         paths,
		CurrentOwners: taskOwnerMembersToProto(currentTaskOwnerMembers(order, candidates)),
		CodeOwners:    taskCodeOwnersToProto(suggestion.CodeOwners),
		Members:       taskOwnerMembersToProto(suggestion.Members),
		HistoryNote:   suggestion.HistoryNote,
	}, nil
}

// SuggestCodeOwners ranks code owners for the paths. With no paths it returns
// the members only. A missing history source returns the members and a note.
func SuggestCodeOwners(
	ctx context.Context,
	deps IntakeDependencies,
	factoryModel *models.Factory,
	order *models.FactoryWorkOrder,
	repository string,
	paths []string,
	candidates []models.TaskOwnerCandidate,
) taskowner.Suggestion {
	suggestion := taskowner.Suggestion{CodeOwners: []taskowner.CodeOwner{}, Members: taskowner.Members(candidates)}
	if len(paths) == 0 {
		return suggestion
	}

	history, err := taskOwnerHistoryFor(ctx, deps, factoryModel, repository)
	if err == nil {
		suggestion, err = taskowner.Suggest(ctx, history, paths, candidates, time.Now())
	}
	if err != nil {
		log.WithError(err).Warnf("task owner history unavailable for order %s", order.ID)
		suggestion.HistoryNote = TaskOwnerHistoryUnavailable
	}
	return suggestion
}

// AddTaskOwnerRoles adds the organization roles that the Members page shows.
// A lookup error leaves the roles empty: the members stay usable.
func AddTaskOwnerRoles(ctx context.Context, auth authorization.Authorization, orgID uuid.UUID, candidates []models.TaskOwnerCandidate) {
	rolesByUser, err := organizationRolesByUser(ctx, auth, orgID.String())
	if err != nil {
		log.WithError(err).Warnf("task owner roles unavailable for organization %s", orgID)
	}
	for i := range candidates {
		roles := []string{}
		if candidates[i].IsOwner {
			roles = append(roles, models.DisplayNameOwner)
		}
		candidates[i].Roles = append(roles, rolesByUser[candidates[i].UserID.String()]...)
	}
}

func organizationRolesByUser(ctx context.Context, auth authorization.Authorization, orgID string) (map[string][]string, error) {
	if auth == nil {
		return nil, errors.New("authorization service is unavailable")
	}
	definitions, err := auth.GetAllRoleDefinitions(ctx, models.DomainTypeOrganization, orgID)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		names = append(names, definition.Name)
	}
	metadata, err := models.FindRoleMetadataByNames(names, models.DomainTypeOrganization, orgID)
	if err != nil {
		return nil, err
	}

	rolesByUser := map[string][]string{}
	for _, definition := range definitions {
		userIDs, err := auth.GetOrgUsersForRole(ctx, definition.Name, orgID)
		if err != nil {
			return nil, err
		}
		displayName := definition.Name
		if roleMetadata, ok := metadata[definition.Name]; ok && roleMetadata.DisplayName != "" {
			displayName = roleMetadata.DisplayName
		}
		for _, userID := range userIDs {
			rolesByUser[userID] = append(rolesByUser[userID], displayName)
		}
	}
	return rolesByUser, nil
}

// TaskOwnerHistory reads code history through the GitHub integration of the
// factory. Bitbucket factories have no history source yet.
func TaskOwnerHistory(ctx context.Context, deps IntakeDependencies, factoryModel *models.Factory, repository string) (taskowner.History, error) {
	config := factoryModel.OnboardingConfigValue()
	if !strings.EqualFold(effectiveTaskOwnerProvider(config.VCSProvider), models.ProviderGitHub) {
		return nil, fmt.Errorf("provider %q has no code history source", config.VCSProvider)
	}
	repository = strings.TrimSpace(repository)
	if repository == "" {
		repository = strings.TrimSpace(config.AppRepository)
	}
	if repository == "" {
		return nil, errors.New("factory has no repository")
	}
	integrationID, err := uuid.Parse(strings.TrimSpace(config.VCSIntegrationID))
	if err != nil {
		return nil, errors.New("factory has no GitHub integration")
	}
	if deps.Registry == nil {
		return nil, errors.New("integration registry is unavailable")
	}

	instance, err := models.FindIntegration(factoryModel.OrganizationID, integrationID)
	if err != nil {
		return nil, fmt.Errorf("load integration: %w", err)
	}
	if instance.AppName != "github" || instance.State != models.IntegrationStateReady {
		return nil, fmt.Errorf("integration %s is not a ready GitHub integration", integrationID)
	}
	integrationCtx := contexts.NewIntegrationContext(database.DB(ctx), nil, instance, deps.Encryptor, deps.Registry, nil)
	client, err := common.NewClient(integrationCtx, deps.Registry.HTTPContext())
	if err != nil {
		return nil, fmt.Errorf("build GitHub client: %w", err)
	}
	return taskowner.GitHubHistory{Client: client, Repository: repository}, nil
}

func effectiveTaskOwnerProvider(provider string) string {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return models.ProviderGitHub
	}
	return provider
}

func currentTaskOwnerMembers(order *models.FactoryWorkOrder, candidates []models.TaskOwnerCandidate) []taskowner.Member {
	owners := []taskowner.Member{}
	for _, assigneeID := range order.AssigneeIDs() {
		owner, ok := models.FindTaskOwnerCandidate(candidates, assigneeID.String())
		if !ok {
			owner = models.TaskOwnerCandidate{UserID: assigneeID}
		}
		owners = append(owners, taskowner.MemberOf(owner))
	}
	return owners
}

func taskOwnerMembersToProto(members []taskowner.Member) []*pb.TaskOwnerMember {
	result := make([]*pb.TaskOwnerMember, 0, len(members))
	for _, member := range members {
		result = append(result, &pb.TaskOwnerMember{
			UserId:      member.UserID.String(),
			Name:        member.Name,
			Email:       member.Email,
			GithubLogin: member.GitHubLogin,
			Roles:       member.Roles,
			OpenTasks:   int32(member.OpenTasks),
		})
	}
	return result
}

func taskCodeOwnersToProto(owners []taskowner.CodeOwner) []*pb.TaskCodeOwner {
	result := make([]*pb.TaskCodeOwner, 0, len(owners))
	for _, owner := range owners {
		serialized := &pb.TaskCodeOwner{
			UserId:      owner.UserID.String(),
			Name:        owner.Name,
			GithubLogin: owner.GitHubLogin,
			Roles:       owner.Roles,
			Commits:     int32(owner.Commits),
			Paths:       owner.Paths,
			OpenTasks:   int32(owner.OpenTasks),
			Score:       owner.Score,
		}
		if !owner.LastCommitAt.IsZero() {
			serialized.LastCommitAt = timestamppb.New(owner.LastCommitAt)
		}
		result = append(result, serialized)
	}
	return result
}
