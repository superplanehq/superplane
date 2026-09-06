package workers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

const (
	// How often a workspace is revisited. A tick every few minutes keeps the
	// report near live without walking GitHub on every page load.
	velocitySyncEvery = 5 * time.Minute

	// How long a claim lasts. It must exceed the longest expected backfill so
	// two workers never sync one workspace, and it bounds how long a workspace
	// waits after a worker dies mid-sync.
	velocitySyncLease = 15 * time.Minute

	// How much history a workspace gets on its first sync, so Velocity is
	// useful on the day the workspace connects.
	velocitySyncBackfillDays = 60

	velocitySyncPageSize      = 100
	velocitySyncFactoriesTick = 200

	// Pages read per repository. This bounds a backfill of a busy repository;
	// the REST budget is 5000 requests an hour, so the cap can be generous.
	velocitySyncMaxPages = 50

	// Repositories read at once, which bounds how fast the worker spends the
	// installation's request budget.
	velocitySyncConcurrency = 4

	// How far back an incremental sync recomputes. The window is recomputed
	// rather than appended to, so this is also how long the sync has to notice
	// a merge GitHub indexed late, or a SuperPlane pull request recorded after
	// its merge was first seen.
	velocitySyncRecomputeWindow = 7 * 24 * time.Hour

	// How far past the merge window commits are read. A commit is dated when it
	// was written, which can be well before the merge that brought it in.
	velocitySyncCommitDateSlack = 14 * 24 * time.Hour

	// The co-author trailer the agent signs its work with. It marks agent output
	// whichever SuperPlane instance opened the pull request, which is what
	// factory_pull_requests cannot know.
	velocityAgentCoAuthorEmail = "superplaneagent@superplane.com"

	// How recently a workspace may have synced before a user-triggered refresh
	// declines to repeat the work. It debounces a button a user can press
	// repeatedly, without making them wait out the full lease.
	velocitySyncOnDemandGuard = 20 * time.Second
)

// FactoryVelocitySyncWorker keeps factory_velocity_repository_merges current
// for every workspace that selected a GitHub integration and an app repository.
//
// Velocity used to call GitHub inside the request that rendered the page,
// which made the people half of the report disappear on any API failure.
// This worker moves that traffic off the request path and makes the data durable.
type FactoryVelocitySyncWorker struct {
	semaphore   *semaphore.Weighted
	logger      *log.Entry
	encryptor   crypto.Encryptor
	registry    *registry.Registry
	rabbitMQURL string
	now         func() time.Time
}

func NewFactoryVelocitySyncWorker(rabbitMQURL string, encryptor crypto.Encryptor, reg *registry.Registry) *FactoryVelocitySyncWorker {
	return &FactoryVelocitySyncWorker{
		semaphore:   semaphore.NewWeighted(velocitySyncConcurrency),
		logger:      log.WithFields(log.Fields{"worker": "FactoryVelocitySyncWorker"}),
		encryptor:   encryptor,
		registry:    reg,
		rabbitMQURL: rabbitMQURL,
		now:         time.Now,
	}
}

func (w *FactoryVelocitySyncWorker) Name() string {
	return "FactoryVelocitySyncWorker"
}

func (w *FactoryVelocitySyncWorker) Start(ctx context.Context) {
	// A user asking for a fresh report cannot wait for the next tick, so
	// requests arrive as messages and are served as they land.
	go w.startSyncRequestedConsumer(ctx)

	w.tick(ctx)

	ticker := time.NewTicker(velocitySyncEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			drainTasks(w.semaphore, velocitySyncConcurrency)
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *FactoryVelocitySyncWorker) startSyncRequestedConsumer(ctx context.Context) {
	routingKey := messages.FactoryVelocitySyncRequestedRoutingKey
	options := tackle.Options{
		URL:            w.rabbitMQURL,
		ConnectionName: w.Name(),
		RemoteExchange: messages.CanvasExchange,
		Service:        messages.CanvasExchange + "." + routingKey + "." + w.Name(),
		RoutingKey:     routingKey,
	}

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logging.NewTackleLogger(w.logger))

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := consumer.Start(&options, w.consumeSyncRequested); err != nil {
			w.logger.Errorf("Error consuming messages from %s: %v", routingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		w.logger.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", routingKey)
		time.Sleep(5 * time.Second)
	}
}

func (w *FactoryVelocitySyncWorker) consumeSyncRequested(delivery tackle.Delivery) error {
	data := &pb.FactoryVelocitySyncRequestedMessage{}
	if err := proto.Unmarshal(delivery.Body(), data); err != nil {
		return fmt.Errorf("unmarshal velocity sync request: %w", err)
	}

	factoryID, err := uuid.Parse(data.GetFactoryId())
	if err != nil {
		return fmt.Errorf("parse factory id: %w", err)
	}

	// The sync reads one repository, so it holds a slot for the same reason a
	// scheduled sync does: to keep the GitHub request budget bounded.
	ctx := context.Background()
	if err := w.semaphore.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("acquire semaphore: %w", err)
	}
	defer w.semaphore.Release(1)

	synced, err := w.SyncFactory(ctx, factoryID)
	if err != nil {
		return fmt.Errorf("sync velocity for workspace %s: %w", factoryID, err)
	}
	if !synced {
		w.logger.WithField("factory", factoryID).Info("Requested velocity sync had nothing to read")
	}
	return nil
}

// SyncFactory rebuilds the report of one workspace now, whatever its schedule
// says.
//
// A user who just merged something should not have to wait for the next
// scheduled run, so this bypasses the staleness check. It keeps a short guard
// against a sync that is already running, which also stops repeated requests
// from spending the request budget twice over.
//
// It re-reads the whole history window, not only recent days: a user who asks
// for a fresh report expects every number on the page to be rebuilt, including
// merges whose classification changed since they were first stored.
//
// It reports whether there was anything to sync: a workspace with no repository
// or no version control integration has nothing to read.
func (w *FactoryVelocitySyncWorker) SyncFactory(ctx context.Context, factoryID uuid.UUID) (bool, error) {
	target, err := models.FindFactoryVelocitySyncTarget(database.Conn(), factoryID)
	if err != nil {
		return false, fmt.Errorf("load velocity sync target: %w", err)
	}
	if target == nil {
		return false, nil
	}

	now := w.now()
	err = w.claimAndSync(ctx, *target, now.Add(-velocitySyncOnDemandGuard), velocitySyncBackfillStart(now))
	if err != nil {
		return false, err
	}
	return true, nil
}

func (w *FactoryVelocitySyncWorker) tick(ctx context.Context) {
	now := w.now()
	targets, err := models.ListFactoryVelocitySyncTargets(
		database.Conn(),
		now.Add(-velocitySyncEvery),
		now.Add(-velocitySyncLease),
		velocitySyncFactoriesTick,
	)
	if err != nil {
		w.logger.Errorf("Error listing workspaces due for velocity sync: %v", err)
		return
	}
	if len(targets) == 0 {
		return
	}

	w.logger.Infof("Found %d workspaces due for velocity sync", len(targets))

	for _, target := range targets {
		if err := w.semaphore.Acquire(ctx, 1); err != nil {
			return
		}

		go func(target models.FactoryVelocitySyncTarget) {
			defer w.semaphore.Release(1)

			now := w.now()
			from := velocitySyncWindow(target, now)
			if err := w.claimAndSync(ctx, target, now.Add(-velocitySyncLease), from); err != nil {
				w.logger.Errorf("Error syncing velocity for workspace %s: %v", target.FactoryID, err)
			}
		}(target)
	}
}

// claimAndSync reads one workspace's repository and stores the merges.
//
// It claims the workspace, then talks to GitHub without holding a transaction.
// A workspace another worker already holds is skipped.
func (w *FactoryVelocitySyncWorker) claimAndSync(
	ctx context.Context,
	target models.FactoryVelocitySyncTarget,
	claimableBefore time.Time,
	from time.Time,
) error {
	if _, _, ok := splitOwnerRepo(target.Repository); !ok {
		return fmt.Errorf("repository %q is not owner/name", target.Repository)
	}

	sync, err := models.ClaimFactoryVelocitySync(database.Conn(), target.FactoryID, claimableBefore)
	if err != nil {
		return fmt.Errorf("claim velocity sync: %w", err)
	}
	if sync == nil {
		return nil
	}

	now := w.now()
	to := now.Add(time.Hour)
	merged, err := w.listTargetMerges(ctx, target, from, to)
	if err != nil {
		w.recordSyncError(target, sync, err)
		return nil
	}

	if err := w.storeMerges(target, sync, from, merged, now); err != nil {
		w.recordSyncError(target, sync, err)
		return nil
	}

	w.logger.Infof(
		"Synced %d merged pull requests of %s for workspace %s",
		len(merged), target.Repository, target.FactoryID,
	)
	return nil
}

func (w *FactoryVelocitySyncWorker) listTargetMerges(
	ctx context.Context,
	target models.FactoryVelocitySyncTarget,
	from, to time.Time,
) ([]repositoryMerge, error) {
	client, err := w.githubClient(target.OrganizationID, target.IntegrationID)
	if err != nil {
		return nil, err
	}
	return listRepositoryMerges(ctx, client, target.Repository, from, to)
}

func (w *FactoryVelocitySyncWorker) recordSyncError(
	target models.FactoryVelocitySyncTarget,
	sync *models.FactoryVelocitySync,
	cause error,
) {
	w.logger.Warnf("Velocity sync failed for workspace %s: %v", target.FactoryID, cause)
	if err := sync.RecordError(database.Conn(), cause.Error()); err != nil {
		w.logger.Errorf("Error recording velocity sync failure: %v", err)
	}
}

// storeMerges writes the merges of one workspace. SuperPlane pull requests this
// instance opened stay in factory_pull_requests and are not stored again.
func (w *FactoryVelocitySyncWorker) storeMerges(
	target models.FactoryVelocitySyncTarget,
	sync *models.FactoryVelocitySync,
	from time.Time,
	merged []repositoryMerge,
	now time.Time,
) error {
	repositoryChanged := target.SyncedRepository != "" && !sameRepository(target)
	// The GitHub window ends at now, but the stored window must reach past it
	// so a merge later today replaces the right rows on the next tick.
	windowEnd := now.Add(time.Hour)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		if repositoryChanged {
			if err := models.DeleteFactoryVelocityRepositoryMerges(tx, target.FactoryID); err != nil {
				return err
			}
		}

		superplane, err := models.ListFactoryPullRequestNumbers(tx, target.FactoryID, target.Repository)
		if err != nil {
			return err
		}

		rows := repositoryMergeRows(target, merged, superplane)
		return models.ReplaceFactoryVelocityRepositoryMerges(tx, target.FactoryID, from, windowEnd, rows)
	})
	if err != nil {
		return fmt.Errorf("store repository merges: %w", err)
	}

	backfilledFrom := earliestBackfill(target, from, repositoryChanged)
	return sync.RecordSuccess(database.Conn(), target.Repository, now, backfilledFrom)
}

// repositoryMergeRows keeps the merges SuperPlane did not open. Subtracting here
// rather than at read time keeps factory_velocity_repository_merges disjoint
// from factory_pull_requests, so the report never has to reconcile the two.
func repositoryMergeRows(
	target models.FactoryVelocitySyncTarget,
	merged []repositoryMerge,
	superplaneNumbers []int64,
) []models.FactoryVelocityRepositoryMerge {
	owned := make(map[int64]bool, len(superplaneNumbers))
	for _, number := range superplaneNumbers {
		owned[number] = true
	}

	rows := make([]models.FactoryVelocityRepositoryMerge, 0, len(merged))
	for _, merge := range merged {
		if owned[merge.number] {
			continue
		}
		row := models.NewFactoryVelocityRepositoryMerge(
			target.OrganizationID,
			target.FactoryID,
			merge.repository,
			merge.number,
			merge.source,
			merge.mergedAt,
		)
		row.AuthorLogin = merge.authorLogin
		row.AuthorName = merge.authorName
		row.AuthorAvatarURL = merge.authorAvatarURL
		rows = append(rows, row)
	}
	return rows
}

func (w *FactoryVelocitySyncWorker) githubClient(orgID, integrationID uuid.UUID) (*common.Client, error) {
	if w.registry == nil {
		return nil, errors.New("integration registry is unavailable")
	}

	instance, err := models.FindIntegration(orgID, integrationID)
	if err != nil {
		return nil, fmt.Errorf("load integration: %w", err)
	}
	if instance.AppName != "github" {
		return nil, fmt.Errorf("integration %s is not a GitHub integration", integrationID)
	}
	if instance.State != models.IntegrationStateReady {
		return nil, fmt.Errorf("integration %s is not ready", integrationID)
	}

	integrationCtx := contexts.NewIntegrationContext(
		database.Conn(),
		nil,
		instance,
		w.encryptor,
		w.registry,
		nil,
	)

	client, err := common.NewClient(integrationCtx, w.registry.HTTPContext())
	if err != nil {
		return nil, fmt.Errorf("build GitHub client: %w", err)
	}
	return client, nil
}

// velocitySyncWindow returns the earliest merge date to ask GitHub for. A
// workspace that already covers the full history window only recomputes recent
// days. Anything else backfills the full window.
func velocitySyncWindow(target models.FactoryVelocitySyncTarget, now time.Time) time.Time {
	backfillFrom := velocitySyncBackfillStart(now)

	if target.SyncedAt == nil || !sameRepository(target) {
		return backfillFrom
	}
	// A stored window that starts later than the wanted one does not cover the
	// full history yet, so keep backfilling instead of only recomputing.
	if target.BackfilledFrom == nil || target.BackfilledFrom.After(backfillFrom) {
		return backfillFrom
	}
	return now.Add(-velocitySyncRecomputeWindow)
}

// velocitySyncBackfillStart is the start of the whole history window a report
// covers.
func velocitySyncBackfillStart(now time.Time) time.Time {
	return now.AddDate(0, 0, -velocitySyncBackfillDays)
}

// earliestBackfill keeps the earliest merge date the workspace has ever
// collected, so an incremental sync does not shrink the recorded history.
func earliestBackfill(
	target models.FactoryVelocitySyncTarget,
	from time.Time,
	repositoryChanged bool,
) time.Time {
	if repositoryChanged || target.BackfilledFrom == nil {
		return from
	}
	if target.BackfilledFrom.Before(from) {
		return *target.BackfilledFrom
	}
	return from
}

func sameRepository(target models.FactoryVelocitySyncTarget) bool {
	return strings.EqualFold(
		strings.TrimSpace(target.SyncedRepository),
		strings.TrimSpace(target.Repository),
	)
}

type repositoryMerge struct {
	repository      string
	number          int64
	source          string
	authorLogin     string
	authorName      string
	authorAvatarURL string
	mergedAt        time.Time
}

// listRepositoryMerges returns the pull requests of a repository that merged in
// [from, to), each classified by who wrote it.
//
// It pages the repository's closed pull requests newest-updated first and stops
// at the first one untouched since the window opened. A merge always updates its
// pull request, so nothing inside the window sorts after that point.
//
// Classification needs the merge commit message, which a pull request listing
// omits, so the commits of the window are read once and joined on the merge
// commit. That keeps the cost at a few pages per repository rather than one
// request per pull request.
func listRepositoryMerges(
	ctx context.Context,
	client *common.Client,
	repository string,
	from, to time.Time,
) ([]repositoryMerge, error) {
	agentCommits, err := listAgentCommits(ctx, client, repository, from, to)
	if err != nil {
		return nil, err
	}

	opts := &github.PullRequestListOptions{
		State:       "closed",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: github.ListOptions{PerPage: velocitySyncPageSize},
	}

	var merges []repositoryMerge
	for page := 0; page < velocitySyncMaxPages; page++ {
		pullRequests, resp, err := client.ListPullRequests(ctx, repository, opts)
		if err != nil {
			return nil, fmt.Errorf("list merged pull requests: %w", err)
		}

		for _, pullRequest := range pullRequests {
			if pullRequest.GetUpdatedAt().Time.Before(from) {
				return merges, nil
			}
			if merge, ok := toRepositoryMerge(pullRequest, repository, agentCommits, from, to); ok {
				merges = append(merges, merge)
			}
		}

		if resp == nil || resp.NextPage == 0 {
			return merges, nil
		}
		opts.Page = resp.NextPage
	}

	return merges, nil
}

// listAgentCommits returns the commits of the window the agent co-authored,
// keyed by commit sha.
//
// The window is widened at both ends, because a commit is dated when it was
// written rather than when it merged, and a merge inside the window can carry an
// older commit date.
func listAgentCommits(
	ctx context.Context,
	client *common.Client,
	repository string,
	from, to time.Time,
) (map[string]bool, error) {
	opts := &github.CommitsListOptions{
		Since:       from.Add(-velocitySyncCommitDateSlack),
		Until:       to.Add(velocitySyncCommitDateSlack),
		ListOptions: github.ListOptions{PerPage: velocitySyncPageSize},
	}

	agent := map[string]bool{}
	for page := 0; page < velocitySyncMaxPages; page++ {
		commits, resp, err := client.ListCommits(ctx, repository, opts)
		if err != nil {
			return nil, fmt.Errorf("list repository commits: %w", err)
		}

		for _, commit := range commits {
			if commit == nil || commit.GetSHA() == "" {
				continue
			}
			if hasAgentCoAuthor(commit.GetCommit().GetMessage()) {
				agent[commit.GetSHA()] = true
			}
		}

		if resp == nil || resp.NextPage == 0 {
			return agent, nil
		}
		opts.Page = resp.NextPage
	}

	return agent, nil
}

// hasAgentCoAuthor reports whether a commit message credits the SuperPlane
// agent. The agent signs the work it writes with a co-author trailer, so this
// recognizes agent output even when another SuperPlane instance opened the pull
// request and this database has no record of it.
func hasAgentCoAuthor(message string) bool {
	for _, line := range strings.Split(message, "\n") {
		line = strings.ToLower(strings.TrimSpace(line))
		if !strings.HasPrefix(line, "co-authored-by:") {
			continue
		}
		if strings.Contains(line, velocityAgentCoAuthorEmail) {
			return true
		}
	}
	return false
}

func toRepositoryMerge(
	pullRequest *github.PullRequest,
	repository string,
	agentCommits map[string]bool,
	from, to time.Time,
) (repositoryMerge, bool) {
	if pullRequest == nil || pullRequest.Number == nil {
		return repositoryMerge{}, false
	}

	// Closed pull requests include the ones nobody merged. Only a merge is
	// output.
	mergedAt := pullRequest.GetMergedAt().Time
	if mergedAt.IsZero() || mergedAt.Before(from) || !mergedAt.Before(to) {
		return repositoryMerge{}, false
	}

	merge := repositoryMerge{
		repository:      repository,
		number:          int64(pullRequest.GetNumber()),
		source:          models.FactoryVelocityMergeSourcePeople,
		authorLogin:     pullRequest.GetUser().GetLogin(),
		authorName:      pullRequest.GetUser().GetName(),
		authorAvatarURL: pullRequest.GetUser().GetAvatarURL(),
		mergedAt:        mergedAt,
	}

	if agentCommits[pullRequest.GetMergeCommitSHA()] {
		merge.source = models.FactoryVelocityMergeSourceAgent
		return merge, true
	}

	// Automation that is not the agent, such as a dependency updater, is neither
	// SuperPlane output nor work a person wrote. Counting it as either would
	// misstate both series, so it is dropped.
	if isBotAuthor(pullRequest.GetUser()) {
		return repositoryMerge{}, false
	}

	return merge, true
}

// isBotAuthor reports whether a pull request author is a machine.
//
// The People series must show what the team wrote by hand. Agents and
// dependency updaters act through a GitHub App, so GitHub reports the author as
// type "Bot" with a "[bot]" login suffix rather than as the machine user that
// signed the commits. The list endpoint omits the type on some payloads, so the
// suffix is checked as well.
//
// Pull requests SuperPlane opened itself are already excluded through
// factory_pull_requests. This drops agent merges from outside this instance,
// which that table cannot know about.
func isBotAuthor(user *github.User) bool {
	if user == nil {
		return false
	}

	if strings.EqualFold(user.GetType(), "Bot") {
		return true
	}

	login := strings.ToLower(strings.TrimSpace(user.GetLogin()))
	return strings.HasSuffix(login, "[bot]")
}

func splitOwnerRepo(repository string) (owner, repo string, ok bool) {
	parts := strings.Split(strings.TrimSpace(repository), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return strings.ToLower(parts[0]), strings.ToLower(parts[1]), true
}
